#!/bin/bash

set -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

function extra_params() {
  case "$1" in
    -kn|--kcp-name)
      shift
      KCP_INSTANCE_NAME=$1
      shift
      ;;
    *)
     echo "ERROR: '$1' is not a recognized flag!" >&2
     user_help >&2
     exit 1
     ;;
  esac
}

function extra_help() {
  echo "-kn, --kcp-name               The name of the kcp instance - eg. kcp-stable, kcp-unstable (default is 'dev')"
  echo "--insecure                    Disable SSL check for KCP syncer when set to 'true'"
}

source ${ROOT}/hack/flags.sh "The configure-kcp.sh configures the kcp instance with the needed workspaces and a workload cluster. The current context of the kcp kubeconfig should point to the kcp instance." extra_params extra_help
parse_flags $@

REDHAT_APPSTUDIO_URL=${REDHAT_APPSTUDIO_URL:-"dev"}
# improve kubeconfig thing when this is addressed https://github.com/kcp-dev/kcp/issues/1689
echo "Using the '${ROOT_WORKSPACE}' workspace as the root"
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}
echo

if [[ ${ROOT_WORKSPACE} == "root" ]]
then
  COMPUTE_WORKSPACE=${COMPUTE_WORKSPACE:-"redhat-appstudio-internal-compute"}
else
  COMPUTE_WORKSPACE=${COMPUTE_WORKSPACE:-"compute"}
fi

echo "Creating and accessing '${COMPUTE_WORKSPACE}' for compute:"
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws create ${COMPUTE_WORKSPACE} --type root:universal --ignore-existing || true
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${COMPUTE_WORKSPACE}
echo

SYNC_TARGET=appstudio-internal
if [[ -z "$(kubectl get synctargets.workload.kcp.dev ${SYNC_TARGET} --kubeconfig ${KCP_KUBECONFIG} 2>/dev/null)" ]]; then
  echo "Creating SyncTarget..."
  KUBECONFIG=${KCP_KUBECONFIG} kubectl kcp workload sync ${SYNC_TARGET} --syncer-image ghcr.io/kcp-dev/kcp/syncer:main --resources=services,routes.route.openshift.io -o /tmp/${SYNC_TARGET}-syncer.yaml
  if grep -q "insecure-skip-tls-verify: true" ${KCP_KUBECONFIG}; then
    sed -i 's/certificate-authority-data: .*/insecure-skip-tls-verify: true/' /tmp/${SYNC_TARGET}-syncer.yaml
  fi
  kubectl apply -f /tmp/${SYNC_TARGET}-syncer.yaml --kubeconfig ${CLUSTER_KUBECONFIG}
fi

BIND_SCOPE="system:authenticated"
if [[ ${KCP_INSTANCE_NAME} == "kcp-stable" ]] || [[ ${KCP_INSTANCE_NAME} == "kcp-unstable" ]] && [[ ${ROOT_WORKSPACE} == "root" ]]
then
  BIND_SCOPE="rh-sso:16270929"
fi

echo "Creating ClusterRole(Binding) to make the APIExport of the compute bindable for the group '${BIND_SCOPE}':"
cat <<EOF | kubectl apply --kubeconfig ${KCP_KUBECONFIG} -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: can-bind-${SYNC_TARGET}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: bind-${SYNC_TARGET}
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: ${BIND_SCOPE}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: bind-${SYNC_TARGET}
rules:
- apiGroups:
  - apis.kcp.dev
  resourceNames:
  - kubernetes
  resources:
  - apiexports
  verbs:
  - bind
EOF
echo

echo -n "Waiting for SyncTarget to be ready: "
while [[ -z "$(kubectl get synctargets.workload.kcp.dev ${SYNC_TARGET} -o wide --kubeconfig ${KCP_KUBECONFIG} | grep True)" ]]; do
  echo -n "."
  sleep 1
done
echo " OK"
echo

APPSTUDIO_WORKSPACE=${APPSTUDIO_WORKSPACE:-"redhat-appstudio"}
echo "Creating and accessing '${APPSTUDIO_WORKSPACE}' for AppStudio controllers:"
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}

if [ "$ROOT_WORKSPACE" == "~" ]; then
  CURRENT_WS=$(KUBECONFIG=${KCP_KUBECONFIG} kubectl ws . | cut -f2 -d'"')
  COMPUTE_WORKSPACE_PATH=${CURRENT_WS}:${COMPUTE_WORKSPACE}
else
  COMPUTE_WORKSPACE_PATH=${ROOT_WORKSPACE}:${COMPUTE_WORKSPACE}
fi


KUBECONFIG=${KCP_KUBECONFIG} kubectl ws create ${APPSTUDIO_WORKSPACE} --ignore-existing --type root:universal || true
REDHAT_APPSTUDIO_URL=$(KUBECONFIG=${KCP_KUBECONFIG} kubectl get workspaces ${APPSTUDIO_WORKSPACE} -o jsonpath='{.status.URL}')
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${APPSTUDIO_WORKSPACE}

echo "Creating APIBinding '${SYNC_TARGET}' for the compute"
cat <<EOF | kubectl apply --kubeconfig ${KCP_KUBECONFIG} -f -
apiVersion: apis.kcp.dev/v1alpha1
kind: APIBinding
metadata:
  name: ${SYNC_TARGET}
spec:
  reference:
    workspace:
      exportName: kubernetes
      path: $COMPUTE_WORKSPACE_PATH
EOF

echo -n "Waiting for APIBinding '${SYNC_TARGET}' to be bound:"
while [[ -z "$(kubectl get apibindings.apis.kcp.dev ${SYNC_TARGET} -o jsonpath="{.status.phase}" --kubeconfig ${KCP_KUBECONFIG} | grep Bound)" ]]; do
  echo -n "."
  sleep 1
done
echo " OK"
echo

echo "Adding Role/RoleBindings for OpenShift GitOps in ${APPSTUDIO_WORKSPACE} workspace:"
kubectl apply --kustomize $ROOT/openshift-gitops/in-kcp --kubeconfig ${KCP_KUBECONFIG}
echo

echo "Getting a token for argocd SA (in ${APPSTUDIO_WORKSPACE} workspace) - kubectl 1.24.x or newer needs to be used."
SA_TOKEN=$(kubectl create token argocd --duration 876000h -n controllers-argocd-manager --kubeconfig ${KCP_KUBECONFIG})
echo

echo "Creating ArgoCD secret representing '${APPSTUDIO_WORKSPACE}' workspace with URL '${REDHAT_APPSTUDIO_URL}' in the compute OpenShift cluster:"
cat <<EOF | kubectl apply --kubeconfig ${CLUSTER_KUBECONFIG} -f -
apiVersion: v1
kind: Secret
metadata:
  name: ${COMPUTE_WORKSPACE}-workspace-${KCP_INSTANCE_NAME}
  namespace: openshift-gitops
  labels:
    argocd.argoproj.io/secret-type: cluster
type: Opaque
stringData:
  name: redhat-appstudio-workspace-${KCP_INSTANCE_NAME}
  server: '${REDHAT_APPSTUDIO_URL}'
  config: |
    {
      "bearerToken": "${SA_TOKEN}",
      "tlsClientConfig": {
        "insecure": true
      }
    }
EOF

echo
echo "Triggering hard refresh of all Applications:"
for APP in $(kubectl get apps -n openshift-gitops -o name --kubeconfig ${CLUSTER_KUBECONFIG}); do
  kubectl patch ${APP} -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}' --kubeconfig ${CLUSTER_KUBECONFIG}
done

echo "Triggering replace sync of all Applications:"
for APP in $(kubectl get apps -n openshift-gitops -o name --kubeconfig ${CLUSTER_KUBECONFIG}); do
  kubectl patch ${APP} -n openshift-gitops --type merge -p='{"operation": {"sync":{"syncOptions": ["Replace=true"]}}}' --kubeconfig ${CLUSTER_KUBECONFIG}
done
