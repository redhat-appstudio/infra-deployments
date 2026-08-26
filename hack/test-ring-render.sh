#!/usr/bin/env bash
#
# test-ring-render.sh - Ring-equality check for -rd ring-deployment migrations.
#
# For a given tier, verifies that each ring in components/<comp>-rd renders
# (kustomize build) identically to the original overlay it replaces. Local
# reviewer aid for KFLUXINFRA-4497; does not touch clusters or run chainsaw.
#
# Usage:
#   hack/test-ring-render.sh <component-rd> <tier>
#     <tier> = dev | staging | production | ring-N | all
#
# Ring -> environment mapping:
#   ring-0            -> development (renders from ring-0/base)
#   ring-1            -> staging     (per-cluster overlays)
#   ring-2,3,4        -> production  (per-cluster overlays)
#
# Exit 0 if every checked comparison is identical, 1 on any diff or setup error.

set -euo pipefail

MAX_DIFF_LINES="${MAX_DIFF_LINES:-80}"

usage() {
  echo "Usage: $0 <component-rd> <tier>" >&2
  echo "  <tier> = dev | staging | production | ring-N | all" >&2
  exit 2
}

[ $# -eq 2 ] || usage

COMPONENT_RD="$1"
TIER="$2"

# Repo root is the parent of hack/.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

COMPONENT="${COMPONENT_RD%-rd}"
if [ "$COMPONENT" = "$COMPONENT_RD" ]; then
  echo "ERROR: component '$COMPONENT_RD' does not end in '-rd'." >&2
  exit 1
fi

RD_DIR="components/${COMPONENT_RD}"
ORIG_DIR="components/${COMPONENT}"
[ -d "$RD_DIR" ] || { echo "ERROR: $RD_DIR not found." >&2; exit 1; }
[ -d "$ORIG_DIR" ] || { echo "ERROR: $ORIG_DIR not found." >&2; exit 1; }

PASS=0
FAIL=0

# Scratch file for kustomize stderr; removed on exit.
ERR_TMP="$(mktemp)"
trap 'rm -f "$ERR_TMP"' EXIT

# build <path> -> stdout rendered manifests; nonzero on kustomize error.
build() {
  kustomize build --enable-helm "$1"
}

# ring_env <ring-name> -> development|staging|production
ring_env() {
  case "$1" in
    ring-0) echo development ;;
    ring-1) echo staging ;;
    ring-2|ring-3|ring-4) echo production ;;
    *) echo "" ;;
  esac
}

# env_top <env> -> deployment-level overlay dir for a non-per-cluster env (dev).
# Some components place the top-level kustomization at <env>/base, others at <env>.
# Prints the resolved path, or empty if neither has a kustomization.yaml.
env_top() {
  local env="$1"
  if [ -f "$ORIG_DIR/$env/base/kustomization.yaml" ]; then
    echo "$ORIG_DIR/$env/base"
  elif [ -f "$ORIG_DIR/$env/kustomization.yaml" ]; then
    echo "$ORIG_DIR/$env"
  else
    echo ""
  fi
}

# compare <label> <rd-path> <orig-path>
# Renders both sides and reports PASS/FAIL with a truncated unified diff.
compare() {
  local label="$1" rd_path="$2" orig_path="$3"

  if [ ! -d "$rd_path" ]; then
    echo "FAIL: $label"
    echo "      missing rd path: $rd_path"
    FAIL=$((FAIL + 1))
    return
  fi
  if [ ! -d "$orig_path" ]; then
    echo "FAIL: $label"
    echo "      missing original overlay: $orig_path"
    FAIL=$((FAIL + 1))
    return
  fi

  local rd_out orig_out
  if ! rd_out="$(build "$rd_path" 2>"$ERR_TMP")"; then
    echo "FAIL: $label"
    echo "      kustomize build failed: $rd_path"
    sed 's/^/        /' "$ERR_TMP" >&2 || true
    FAIL=$((FAIL + 1))
    return
  fi
  if ! orig_out="$(build "$orig_path" 2>"$ERR_TMP")"; then
    echo "FAIL: $label"
    echo "      kustomize build failed: $orig_path"
    sed 's/^/        /' "$ERR_TMP" >&2 || true
    FAIL=$((FAIL + 1))
    return
  fi

  local d
  d="$(diff -u \
        --label "$orig_path (original)" \
        --label "$rd_path (ring)" \
        <(printf '%s\n' "$orig_out") \
        <(printf '%s\n' "$rd_out") || true)"

  if [ -z "$d" ]; then
    echo "PASS: $label"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $label"
    print_diff "$d"
    FAIL=$((FAIL + 1))
  fi
}

# print_diff <diff-text> - indent and truncate long diffs.
print_diff() {
  local text="$1" n
  n="$(printf '%s\n' "$text" | wc -l)"
  if [ "$n" -gt "$MAX_DIFF_LINES" ]; then
    # awk (not head) consumes all input, so the upstream printf never gets
    # SIGPIPE under 'set -o pipefail'.
    printf '%s\n' "$text" | awk -v m="$MAX_DIFF_LINES" 'NR <= m' | sed 's/^/      /'
    echo "      ... $((n - MAX_DIFF_LINES)) more lines - too many changes to show"
  else
    printf '%s\n' "$text" | sed 's/^/      /'
  fi
}

# check_ring <ring-name> - one ring across all its comparisons.
check_ring() {
  local ring="$1"
  local ring_dir="$RD_DIR/rings/$ring"
  if [ ! -d "$ring_dir" ]; then
    echo "FAIL: $ring - not present under $RD_DIR/rings/"
    FAIL=$((FAIL + 1))
    return
  fi

  if [ "$ring" = "ring-0" ]; then
    local dev_top
    dev_top="$(env_top development)"
    if [ -z "$dev_top" ]; then
      echo "FAIL: ring-0 (development)"
      echo "      no development overlay found ($ORIG_DIR/development[/base])"
      FAIL=$((FAIL + 1))
      return
    fi
    compare "ring-0 (development)" "$ring_dir/base" "$dev_top"
    return
  fi

  local env
  env="$(ring_env "$ring")"
  local found=0 cluster cdir
  for cdir in "$ring_dir"/*/; do
    [ -d "$cdir" ] || continue
    cluster="$(basename "$cdir")"
    case "$cluster" in
      base|base-snapshot) continue ;;
    esac
    found=1
    compare "$ring/$cluster ($env)" "$ring_dir/$cluster" "$ORIG_DIR/$env/$cluster"
  done
  if [ "$found" -eq 0 ]; then
    echo "FAIL: $ring - no cluster overlays found (only base/base-snapshot)"
    FAIL=$((FAIL + 1))
  fi
}

# rings_present -> sorted list of ring-* dirs actually present.
rings_present() {
  local d
  for d in "$RD_DIR"/rings/ring-*/; do
    if [ -d "$d" ]; then basename "$d"; fi
  done | sort -V
}

# Resolve tier -> list of rings to check.
RINGS=()
case "$TIER" in
  dev|development) RINGS=(ring-0) ;;
  staging) RINGS=(ring-1) ;;
  production|prod) RINGS=(ring-2 ring-3 ring-4) ;;
  ring-*) RINGS=("$TIER") ;;
  all) mapfile -t RINGS < <(rings_present) ;;
  *) echo "ERROR: unknown tier '$TIER'." >&2; usage ;;
esac

echo "Component: $COMPONENT_RD (original: $COMPONENT)   Tier: $TIER"
echo "Rings: ${RINGS[*]:-<none>}"
echo

for ring in "${RINGS[@]}"; do
  check_ring "$ring"
done

echo
echo "Summary: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
