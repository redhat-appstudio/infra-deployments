#!/bin/bash
# Setup ring-specific overlays for pipeline-service production clusters
# This script adds ring overlay references to each cluster's kustomization.yaml

set -euo pipefail
# nullglob allows for easier error handling when listing all clusters
shopt -s nullglob

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROD_DIR="${SCRIPT_DIR}"
RINGS_FILE="${PROD_DIR}/ring-mappings.yaml"

# Validate that all clusters are defined in ring-mappings.yaml
function validate_all_clusters_mapped() {
  echo "Validating cluster coverage..."
  mapped_clusters=$(yq eval '.[] | .[]' "$RINGS_FILE" | sort -u)
  missing_clusters=()

  for cluster in $mapped_clusters; do
    if [[ ! -d "${PROD_DIR}/${cluster}" ]]; then
      missing_clusters+=("$cluster doesn't exist")
    fi
  done

  for cluster_dir in "${PROD_DIR}"/*/; do
    cluster=$(basename "$cluster_dir")
    # Skip non-cluster directories
    [[ "$cluster" == "base" || "$cluster" == "rings" ]] && continue

    if ! echo "$mapped_clusters" | grep -qx "$cluster"; then
      missing_clusters+=("$cluster unmapped")
    fi
  done


  if [[ ${#missing_clusters[@]} -gt 0 ]]; then
    echo "❌ Error: Cluster Ring configuration validation failed:" >&2
    printf '  - %s\n' "${missing_clusters[@]}" >&2
    return 1
  fi

  echo "✓ All clusters are mapped in ring-mappings.yaml"
  echo ""
}


function sync_rings() {
  # First pass: remove all existing ring references from all clusters
  echo "Removing existing ring references..."
  for kustomization in "${PROD_DIR}"/*/resources/kustomization.yaml; do
    cluster=$(basename "$(dirname "$(dirname "$kustomization")")")
    echo "  - Removing from ${cluster}"
    yq eval -i '.resources = (.resources // [] | map(select(. | test("rings/ring-") | not)))' "$kustomization"
  done

  echo "Setting up pipeline-service ring overlays..."

  # Second pass: add correct ring reference to each cluster
  for ring in $(yq eval 'keys | .[]' "$RINGS_FILE"); do
    echo "  - Processing ${ring}..."

    # Get all clusters for this ring
    clusters=$(yq eval ".${ring}[]" "$RINGS_FILE")

    for cluster in $clusters; do
      kustomization="${PROD_DIR}/${cluster}/resources/kustomization.yaml"

      if [[ ! -f "$kustomization" ]]; then
        echo "    ⚠️  Skipping ${cluster}: kustomization.yaml not found" >&2
        continue
      fi

      # Add the ring overlay at the beginning of resources array
      yq eval -i ".resources = [\"../../rings/${ring}\"] + (.resources // [])" "$kustomization"
      echo "    ✓ Configured ${cluster} with ${ring}"
    done
  done

  echo ""
  echo "✅ Ring overlays configured. Run hack/generate-deploy-config.sh to regenerate deploy.yaml files"
}

validate_all_clusters_mapped
sync_rings
