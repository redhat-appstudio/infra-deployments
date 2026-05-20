#!/usr/bin/env bash
# OpenShift CI entrypoint for development-operator overlay E2E (conformance).
# Invoked by openshift/release after cloning infra-deployments at the PR revision.
#
# Phase 1 (current): placeholder — wiring validation only.
# Phase 2: bootstrap (preview --operator-overlay), konflux-ci conformance @ ref from
#          invariant/kustomization.yaml, default-tenant namespace (no deploy-test-resources.sh).
set -euo pipefail

echo "[openshift-operator-overlay-e2e] placeholder: noop (implementation pending)"
echo "  Expected later: bootstrap development-operator, konflux-ci conformance only,"
echo "  E2E_APPLICATIONS_NAMESPACE=default-tenant (KonfluxDefaultTenant), no deploy-test-resources.sh"
exit 0
