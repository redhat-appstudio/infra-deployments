#!/usr/bin/env bash
# Run konflux-ci conformance tests against the operator overlay cluster.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INVARIANT_KUSTOMIZATION="components/konflux-operator/development/invariant/kustomization.yaml"
KONFLUX_CI_REPO="${KONFLUX_CI_REPO:-konflux-ci/konflux-ci}"

: "${INFRA_DEPLOYMENTS_ROOT:?INFRA_DEPLOYMENTS_ROOT must be set by the CI entrypoint}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN must be set by the CI entrypoint}"

# shellcheck source=ci-common.sh
source "${SCRIPT_DIR}/ci-common.sh"

cd "${INFRA_DEPLOYMENTS_ROOT}"

if ! command -v kubectl &>/dev/null && command -v oc &>/dev/null; then
  kubectl() { oc "$@"; }
  export -f kubectl
fi

KONFLUX_CI_REF="$(ci_parse_konflux_ci_ref "${INVARIANT_KUSTOMIZATION}")"
echo "[INFO] konflux-ci ref from ${INVARIANT_KUSTOMIZATION}: ${KONFLUX_CI_REF}"

KONFLUX_CI_DIR="$(mktemp -d)/konflux-ci"
echo "[INFO] Cloning ${KONFLUX_CI_REPO}..."
git clone --origin upstream --branch main \
  "https://${GITHUB_TOKEN}@github.com/${KONFLUX_CI_REPO}.git" "${KONFLUX_CI_DIR}"

cd "${KONFLUX_CI_DIR}"
if [[ "${KONFLUX_CI_REF}" != "main" ]]; then
  git fetch --tags upstream 2>/dev/null || git fetch upstream
  git checkout "${KONFLUX_CI_REF}" || {
    echo "ERROR: failed to checkout konflux-ci ref ${KONFLUX_CI_REF}" >&2
    exit 1
  }
fi

export GH_ORG="${MY_GITHUB_ORG:-redhat-appstudio-qe}"
export GH_TOKEN="${GITHUB_TOKEN}"
export E2E_APPLICATIONS_NAMESPACE="${E2E_APPLICATIONS_NAMESPACE:-default-tenant}"

if [[ -n "${GINKGO_LABEL_FILTER:-}" ]]; then
  export E2E_CONFORMANCE_GO_TEST_EXTRA_ARGS="-ginkgo.label-filter=${GINKGO_LABEL_FILTER}"
fi

echo "[INFO] Preparing conformance environment (pipeline bundle pin)..."
# prepare-conformance-env.sh prints export lines when not in GitHub Actions.
eval "$(bash scripts/operator-e2e/prepare-conformance-env.sh "${KONFLUX_CI_DIR}")"

JUNIT_REPORT="${ARTIFACT_DIR:-/tmp}/junit-conformance.xml"
mkdir -p "$(dirname "${JUNIT_REPORT}")"

echo "[INFO] Running conformance tests (namespace=${E2E_APPLICATIONS_NAMESPACE})..."
bash scripts/operator-e2e/run-conformance-tests.sh "${KONFLUX_CI_DIR}" "${JUNIT_REPORT}"

echo "[INFO] run-e2e.sh complete"
