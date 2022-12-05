
#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
TOOLCHAIN_E2E_TEMP_DIR="/tmp/toolchain-e2e"

$ROOT/hack/reduce-gitops-cpu-requests.sh

echo
echo "Installing the Toolchain (Sandbox) operators in dev environment:"
rm -rf ${TOOLCHAIN_E2E_TEMP_DIR} 2>/dev/null || true
git clone --depth=1 https://github.com/codeready-toolchain/toolchain-e2e.git ${TOOLCHAIN_E2E_TEMP_DIR}
make -C ${TOOLCHAIN_E2E_TEMP_DIR} appstudio-dev-deploy-latest SHOW_CLEAN_COMMAND="make -C ${TOOLCHAIN_E2E_TEMP_DIR} appstudio-cleanup"


## Patch sandbox config to use provided keycloak

BASE_URL=$(oc get ingresses.config.openshift.io/cluster -o jsonpath={.spec.domain})
RHSSO_URL="https://keycloak-appstudio-sso.$BASE_URL"

oc patch ToolchainConfig/config -n toolchain-host-operator --type=merge --patch-file=/dev/stdin << EOF
spec:
  host:
    registrationService:
      auth:
        authClientConfigRaw: '{
                  "realm": "testrealm",
                  "auth-server-url": "$RHSSO_URL/auth",
                  "ssl-required": "nones",
                  "resource": "sandbox-public",
                  "clientId": "sandbox-public",
                  "public-client": true
                }'
        authClientLibraryURL: $RHSSO_URL/auth/js/keycloak.js
        authClientPublicKeysURL: $RHSSO_URL/auth/realms/testrealm/protocol/openid-connect/certs
EOF