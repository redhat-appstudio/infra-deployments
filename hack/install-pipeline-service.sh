#!/bin/bash -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

source ${ROOT}/hack/flags.sh "The install-pipeline-service.sh installs Pipeline Service for development and testing on non-production clusters / kcp instances."
MODE=preview parse_flags $@


PIPELINE_SERVICE_DIR=$(mktemp -d)
git clone --depth 1 https://github.com/openshift-pipelines/pipeline-service/ $PIPELINE_SERVICE_DIR
export WORK_DIR="${PIPELINE_SERVICE_DIR}/gitops/sre/"
export WORKSPACE_DIR=$WORK_DIR

if [ "$ROOT_WORKSPACE" == "~" ]; then
  ROOT_WORKSPACE=$(KUBECONFIG=${KCP_KUBECONFIG} kubectl ws '~' | cut -f2 -d'"')
fi

KUBECONFIG=$KCP_KUBECONFIG $PIPELINE_SERVICE_DIR/images/access-setup/content/bin/setup_kcp.sh --kcp-workspace pipeline-service --kcp-org $ROOT_WORKSPACE

KUBECONFIG=$CLUSTER_KUBECONFIG $PIPELINE_SERVICE_DIR/images/access-setup/content/bin/setup_compute.sh

$PIPELINE_SERVICE_DIR/images/cluster-setup/bin/install.sh

$PIPELINE_SERVICE_DIR/images/kcp-registrar/register.sh --kcp-org $ROOT_WORKSPACE --kcp-workspace pipeline-service --kcp-sync-tag v0.8.2

rm -rf "$PIPELINE_SERVICE_DIR"

KUBECONFIG=${KCP_KUBECONFIG} kubectl ws $ROOT_WORKSPACE

cat << EOF > /tmp/pipeline-service-binding.yaml
apiVersion: apis.kcp.dev/v1alpha1
kind: APIBinding
metadata:
  name: pipeline-service
spec:
  reference:
    workspace:
      exportName: kubernetes
      path: $ROOT_WORKSPACE:pipeline-service
EOF

echo
echo "APIBinding for your workspace available in /tmp/pipeline-service-binding.yaml, content:"
echo
cat /tmp/pipeline-service-binding.yaml
