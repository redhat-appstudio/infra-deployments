#!/bin/bash
set -eo pipefail

NAMESPACES=(
  "image-controller"
  "integration-service"
  "release-service"
  "build-service"
)

SEALIGHTS_TOKEN=${SEALIGHTS_TOKEN:-""}

for namespace in "${NAMESPACES[@]}"; do
  # Create the namespace if it doesn't exist
  if ! kubectl get namespace "$namespace" >/dev/null 2>&1; then
    echo "[WARN] Namespace '$namespace' does not exist. Creating it..."
    kubectl create namespace "$namespace"
  fi

  if kubectl get secret sealights-token -n "$namespace" >/dev/null 2>&1; then
    echo "[INFO] Updating existing secret 'sealights-token' in namespace '$namespace'."
    kubectl delete secret sealights-token -n "$namespace"
  fi

  kubectl create secret generic sealights-token \
    --from-literal=token="$SEALIGHTS_TOKEN" \
    -n "$namespace"

  echo "[INFO] Secret 'sealights-token' has been created/updated in namespace '$namespace'."
done
