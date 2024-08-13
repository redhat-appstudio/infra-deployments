
#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
TOOLCHAIN_E2E_TEMP_DIR="/tmp/toolchain-e2e"

$ROOT/hack/reduce-gitops-cpu-requests.sh

echo
echo "Installing the Toolchain (Sandbox) operators in dev environment:"
rm -rf ${TOOLCHAIN_E2E_TEMP_DIR} 2>/dev/null || true
git clone --depth=1 https://github.com/codeready-toolchain/toolchain-e2e.git ${TOOLCHAIN_E2E_TEMP_DIR}
CGO_ENABLED=0 go install github.com/kubesaw/ksctl/cmd/ksctl@master
make -C ${TOOLCHAIN_E2E_TEMP_DIR} appstudio-dev-deploy-latest SHOW_CLEAN_COMMAND="make -C ${TOOLCHAIN_E2E_TEMP_DIR} appstudio-cleanup" USE_INSTALLED_KSCTL=true
