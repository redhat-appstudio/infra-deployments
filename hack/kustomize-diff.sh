#!/usr/bin/env bash
#
# kustomize-diff.sh - Compare kustomize builds between two git refs
#
# This script builds all kustomize manifests and shows the diff between
# the merge-base (common ancestor) of base_ref and compare_ref.
# This ensures you only see changes introduced by compare_ref, not changes
# that happened on base_ref after compare_ref was forked.
#
# Usage:
#   ./hack/kustomize-diff.sh [base_ref] [compare_ref] [options]
#
# Arguments:
#   base_ref     - The base git ref to compare against (default: main)
#   compare_ref  - The git ref to compare (default: HEAD)
#
# Options:
#   --keep-temp      Preserve temp directories for inspection
#   --no-merge-base  Compare refs directly without finding merge-base
#   -h, --help       Show this help message
#
# Examples:
#   # Compare current changes against main (uses merge-base)
#   ./hack/kustomize-diff.sh
#
#   # Compare a specific branch against main
#   ./hack/kustomize-diff.sh main feature/my-changes
#
#   # Compare two specific commits directly (no merge-base)
#   ./hack/kustomize-diff.sh abc123 def456 --no-merge-base
#
# Requirements:
#   - kustomize (https://kustomize.io/)
#   - helm (https://helm.sh/) - for charts
#   - yq (optional, for normalizing YAML order) - https://github.com/mikefarah/yq
#   - dyff (optional, for better YAML diffs) - https://github.com/homeport/dyff
#
# Output:
#   - Prints diff summary to stdout
#   - Creates temp directories with rendered manifests (cleaned up on exit)
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
KEEP_TEMP=false
USE_MERGE_BASE=true
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
POSITIONAL_ARGS=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-temp)
            KEEP_TEMP=true
            shift
            ;;
        --no-merge-base)
            USE_MERGE_BASE=false
            shift
            ;;
        --help|-h)
            sed -n '2,/^$/p' "$0" | sed 's/^# //' | sed 's/^#//'
            exit 0
            ;;
        -*)
            echo "Unknown option: $1"
            exit 1
            ;;
        *)
            POSITIONAL_ARGS+=("$1")
            shift
            ;;
    esac
done

# Set positional arguments
BASE_REF="${POSITIONAL_ARGS[0]:-main}"
COMPARE_REF="${POSITIONAL_ARGS[1]:-HEAD}"

# Create temp directories
TEMP_DIR=$(mktemp -d)
BASE_DIR="$TEMP_DIR/base"
COMPARE_DIR="$TEMP_DIR/compare"
BASE_MANIFESTS="$TEMP_DIR/base-manifests"
COMPARE_MANIFESTS="$TEMP_DIR/compare-manifests"
DIFF_OUTPUT="$TEMP_DIR/diff"

mkdir -p "$BASE_DIR" "$COMPARE_DIR" "$BASE_MANIFESTS" "$COMPARE_MANIFESTS" "$DIFF_OUTPUT"

# Cleanup function
cleanup() {
    if [ "$KEEP_TEMP" = false ]; then
        rm -rf "$TEMP_DIR"
    else
        echo -e "${BLUE}Temp directories preserved at: $TEMP_DIR${NC}"
        echo "  Base manifests:    $BASE_MANIFESTS"
        echo "  Compare manifests: $COMPARE_MANIFESTS"
        echo "  Diff output:       $DIFF_OUTPUT"
    fi
}
trap cleanup EXIT

# Check for required tools
check_requirements() {
    local missing=()

    if ! command -v kustomize &> /dev/null; then
        missing+=("kustomize")
    fi

    if ! command -v helm &> /dev/null; then
        missing+=("helm")
    fi

    if [ ${#missing[@]} -ne 0 ]; then
        echo -e "${RED}Error: Missing required tools: ${missing[*]}${NC}"
        echo "Please install them before running this script."
        exit 1
    fi

    # yq is optional but recommended for normalizing YAML order
    if ! command -v yq &> /dev/null; then
        echo -e "${YELLOW}Note: 'yq' not found. YAML documents won't be sorted before comparison.${NC}"
        echo "This may cause false positives when resources are reordered."
        echo "Install yq: https://github.com/mikefarah/yq"
        echo ""
        USE_YQ=false
    else
        USE_YQ=true
    fi

    # dyff is optional but recommended
    if ! command -v dyff &> /dev/null; then
        echo -e "${YELLOW}Note: 'dyff' not found. Using standard diff instead.${NC}"
        echo "Install dyff for better YAML-aware diffs: https://github.com/homeport/dyff"
        echo ""
        USE_DYFF=false
    else
        USE_DYFF=true
    fi
}


# Clone/checkout a specific ref
checkout_ref() {
    local ref="$1"
    local target_dir="$2"

    echo -e "${BLUE}Checking out $ref...${NC}"

    cd "$REPO_ROOT"
    git archive "$ref" | tar -x -C "$target_dir"
}

# Build kustomize manifests
build_manifests() {
    local source_dir="$1"
    local output_dir="$2"
    local label="$3"

    echo -e "${BLUE}Building kustomize manifests for $label (parallel)...${NC}"

    cd "$source_dir"

    # Find all kustomization files and build concurrently (using same approach as CI)
    find argo-cd-apps components configs -name 'kustomization.yaml' \
        ! -path '*/k-components/*' \
        ! -path 'components/repository-validator/staging/*' \
        ! -path 'components/repository-validator/production/*' \
        ! -path 'components/monitoring/blackbox/staging/*' \
        ! -path 'components/monitoring/blackbox/production/*' \
        ! -path 'components/*/chainsaw/*' \
        2>/dev/null | \
        xargs -I {} -n1 -P8 bash -c 'dir=$(dirname "{}"); output_file=$(echo $dir | tr / -).yaml; kustomize build --enable-helm "$dir" -o "'"$output_dir"'/$output_file" 2>/dev/null || true'

    local total=$(ls -1 "$output_dir" 2>/dev/null | wc -l)
    echo -e "${GREEN}  Built $total manifests${NC}"
}

# Normalize/sort YAML documents in manifest files
# This ensures resources are in consistent order for comparison
normalize_manifests() {
    local manifest_dir="$1"

    if [ "$USE_YQ" != true ]; then
        return 0
    fi

    echo -e "${BLUE}Normalizing manifest order...${NC}"

    # Sort each multi-document YAML file by kind/namespace/name
    for manifest in "$manifest_dir"/*.yaml; do
        [ -f "$manifest" ] || continue

        # Sort documents by kind, namespace, name
        # This ensures consistent ordering regardless of kustomize output order
        yq eval-all '
            [.] | sort_by(.kind + "/" + (.metadata.namespace // "") + "/" + (.metadata.name // "")) | .[]
        ' "$manifest" > "$manifest.sorted" 2>/dev/null && mv "$manifest.sorted" "$manifest" || rm -f "$manifest.sorted"
    done
}

# Generate diff between manifests
generate_diff() {
    echo ""
    echo -e "${BLUE}Generating diff...${NC}"
    echo ""

    # Get all unique manifest files and categorize them
    local all_manifests
    all_manifests=$(( ls "$BASE_MANIFESTS" 2>/dev/null; ls "$COMPARE_MANIFESTS" 2>/dev/null ) | sort -u)

    # Create empty changes file
    > "$DIFF_OUTPUT/changes.txt"

    # Process each manifest
    while IFS= read -r manifest; do
        [ -z "$manifest" ] && continue

        base_file="$BASE_MANIFESTS/$manifest"
        compare_file="$COMPARE_MANIFESTS/$manifest"

        if [ ! -f "$base_file" ] && [ -f "$compare_file" ]; then
            echo "NEW:$manifest" >> "$DIFF_OUTPUT/changes.txt"
            cp "$compare_file" "$DIFF_OUTPUT/NEW-$manifest"
        elif [ -f "$base_file" ] && [ ! -f "$compare_file" ]; then
            echo "DELETED:$manifest" >> "$DIFF_OUTPUT/changes.txt"
            cp "$base_file" "$DIFF_OUTPUT/DELETED-$manifest"
        elif [ -f "$base_file" ] && [ -f "$compare_file" ]; then
            if ! diff -q "$base_file" "$compare_file" > /dev/null 2>&1; then
                echo "CHANGED:$manifest" >> "$DIFF_OUTPUT/changes.txt"
                if [ "$USE_DYFF" = true ]; then
                    dyff between "$base_file" "$compare_file" > "$DIFF_OUTPUT/CHANGED-$manifest.diff" 2>/dev/null || \
                        diff -u "$base_file" "$compare_file" > "$DIFF_OUTPUT/CHANGED-$manifest.diff" || true
                else
                    diff -u "$base_file" "$compare_file" > "$DIFF_OUTPUT/CHANGED-$manifest.diff" || true
                fi
            fi
        fi
    done <<< "$all_manifests"

    # Count changes using wc -l (more reliable than grep -c)
    # Use { grep || true; } to prevent set -e from exiting on no matches
    local new_count deleted_count changed_count
    new_count=$({ grep "^NEW:" "$DIFF_OUTPUT/changes.txt" 2>/dev/null || true; } | wc -l | tr -d ' ')
    deleted_count=$({ grep "^DELETED:" "$DIFF_OUTPUT/changes.txt" 2>/dev/null || true; } | wc -l | tr -d ' ')
    changed_count=$({ grep "^CHANGED:" "$DIFF_OUTPUT/changes.txt" 2>/dev/null || true; } | wc -l | tr -d ' ')

    # Ensure counts are integers
    new_count=${new_count:-0}
    deleted_count=${deleted_count:-0}
    changed_count=${changed_count:-0}

    # Print summary
    echo "=========================================="
    echo -e "${BLUE}KUSTOMIZE DIFF SUMMARY${NC}"
    echo "=========================================="
    echo ""
    echo "Comparing: $BASE_REF -> $COMPARE_REF"
    echo ""

    if [ "$new_count" -eq 0 ] && [ "$deleted_count" -eq 0 ] && [ "$changed_count" -eq 0 ]; then
        echo -e "${GREEN}✅ No changes detected${NC}"
        echo ""
        echo "The rendered Kubernetes manifests are identical."
        return 0
    fi

    echo -e "${YELLOW}Changes detected:${NC}"
    echo ""

    if [ "$new_count" -gt 0 ]; then
        echo -e "${GREEN}🆕 New manifests ($new_count):${NC}"
        grep "^NEW:" "$DIFF_OUTPUT/changes.txt" | cut -d: -f2- | while read -r f; do
            echo "   - $f"
        done
        echo ""
    fi

    if [ "$deleted_count" -gt 0 ]; then
        echo -e "${RED}🗑️  Deleted manifests ($deleted_count):${NC}"
        grep "^DELETED:" "$DIFF_OUTPUT/changes.txt" | cut -d: -f2- | while read -r f; do
            echo "   - $f"
        done
        echo ""
    fi

    if [ "$changed_count" -gt 0 ]; then
        echo -e "${YELLOW}📝 Modified manifests ($changed_count):${NC}"
        grep "^CHANGED:" "$DIFF_OUTPUT/changes.txt" | cut -d: -f2- | while read -r f; do
            echo "   - $f"
        done
        echo ""
    fi

    # Print detailed diffs
    echo "=========================================="
    echo -e "${BLUE}DETAILED CHANGES${NC}"
    echo "=========================================="
    echo ""

    for diff_file in "$DIFF_OUTPUT"/CHANGED-*.diff; do
        [ -f "$diff_file" ] || continue

        manifest_name=$(basename "$diff_file" .diff | sed 's/^CHANGED-//')
        echo -e "${YELLOW}━━━ $manifest_name ━━━${NC}"
        echo ""
        cat "$diff_file"
        echo ""
    done

    return 0
}

# Find merge base between two refs
find_merge_base() {
    local base="$1"
    local compare="$2"

    cd "$REPO_ROOT"
    git merge-base "$base" "$compare" 2>/dev/null
}

# Main execution
main() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║     Kustomize Diff Tool                ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
    echo ""

    check_requirements

    # Determine the actual base ref to use
    local actual_base_ref="$BASE_REF"

    if [ "$USE_MERGE_BASE" = true ]; then
        echo -e "${BLUE}Finding merge-base between $BASE_REF and $COMPARE_REF...${NC}"
        local merge_base
        merge_base=$(find_merge_base "$BASE_REF" "$COMPARE_REF")

        if [ -n "$merge_base" ]; then
            actual_base_ref="$merge_base"
            echo -e "Merge-base: ${GREEN}${merge_base:0:12}${NC}"
            echo ""
            echo "This ensures you only see changes introduced by your branch,"
            echo "not changes that happened on $BASE_REF after your branch was forked."
            echo ""
        else
            echo -e "${YELLOW}Warning: Could not find merge-base, using $BASE_REF directly${NC}"
            echo ""
        fi
    fi

    echo "Base ref:    $actual_base_ref"
    echo "Compare ref: $COMPARE_REF"
    echo ""

    # Checkout both refs
    checkout_ref "$actual_base_ref" "$BASE_DIR"
    checkout_ref "$COMPARE_REF" "$COMPARE_DIR"

    # Build manifests for both
    build_manifests "$BASE_DIR" "$BASE_MANIFESTS" "$actual_base_ref"
    build_manifests "$COMPARE_DIR" "$COMPARE_MANIFESTS" "$COMPARE_REF"

    # Normalize manifest order to avoid false positives from reordering
    normalize_manifests "$BASE_MANIFESTS"
    normalize_manifests "$COMPARE_MANIFESTS"

    # Generate and show diff
    generate_diff
}

main "$@"
