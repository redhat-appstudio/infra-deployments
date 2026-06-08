#!/usr/bin/env bash
# Bootstrap the claimed OpenShift cluster with development-operator overlay preview.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS_DIR="${KONFLUX_CI_SECRETS_DIR:-/usr/local/konflux-ci-secrets-new/redhat-appstudio-qe}"

: "${INFRA_DEPLOYMENTS_ROOT:?INFRA_DEPLOYMENTS_ROOT must be set by the CI entrypoint}"

# shellcheck source=ci-common.sh
source "${SCRIPT_DIR}/ci-common.sh"

cd "${INFRA_DEPLOYMENTS_ROOT}"

echo "[INFO] Configuring preview install environment from QE secrets..."
ci_export_preview_install_env "${SECRETS_DIR}"

echo "[INFO] Marking control-plane nodes schedulable (small CI clusters)..."
oc patch scheduler cluster --type=merge -p '{"spec":{"mastersSchedulable":true}}' 2>&1 \
  || echo "[WARN] Could not patch scheduler (may be HyperShift)"

echo "[INFO] Running hack/bootstrap-cluster.sh preview --operator-overlay..."
./hack/bootstrap-cluster.sh preview --operator-overlay

echo "[INFO] Creating e2e-secrets/quay-repository..."
ci_create_e2e_quay_secret "${QUAY_TOKEN}"

echo "[INFO] Registering PAC with SprayProxy..."
ci_register_sprayproxy_pac_route "${SECRETS_DIR}"

echo "[INFO] install.sh complete (development-operator overlay)"
