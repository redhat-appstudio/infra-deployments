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
}

configure_compute_workspace() {
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
    KUBECONFIG=${KCP_KUBECONFIG} kubectl kcp workload sync ${SYNC_TARGET} --syncer-image ghcr.io/kcp-dev/kcp/syncer:v0.9.0 --resources=services,routes.route.openshift.io -o /tmp/${SYNC_TARGET}-syncer.yaml
    if grep -q "insecure-skip-tls-verify: true" ${KCP_KUBECONFIG}; then
      sed -i.bak 's/certificate-authority-data: .*/insecure-skip-tls-verify: true/' /tmp/${SYNC_TARGET}-syncer.yaml && rm /tmp/${SYNC_TARGET}-syncer.yaml.bak
    fi
    kubectl apply -f /tmp/${SYNC_TARGET}-syncer.yaml --kubeconfig ${CLUSTER_KUBECONFIG}
  fi
  
  BIND_SCOPE="system:authenticated"
  if [[ ${KCP_INSTANCE_NAME} == "kcp-stable" ]] || [[ ${KCP_INSTANCE_NAME} == "kcp-unstable" ]] && [[ ${ROOT_WORKSPACE} == "root" ]]
  then
    # This "bind scope" represents a RH SSO group that has admin access in service provider workspaces as well as in the internal compute workspace.
    # This scope should be used only in CPS when applying in "root" workspace.
    # This makes sure that the admin who runs the script can bind the compute in service provider workspaces but no one else anywhere in CPS.
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
}


configure_service_provider_workspace() {
  echo "Creating and accessing '${SP_WORKSPACE_NAME}':"
  KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}
  COMPUTE_WORKSPACE_PATH=${ROOT_WORKSPACE}:${COMPUTE_WORKSPACE}
  
  KUBECONFIG=${KCP_KUBECONFIG} kubectl ws create ${SP_WORKSPACE_NAME} --ignore-existing --type root:universal || true
  SP_WORKSPACE_URL=$(KUBECONFIG=${KCP_KUBECONFIG} kubectl get workspaces ${SP_WORKSPACE_NAME} -o jsonpath='{.status.URL}')
  KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${SP_WORKSPACE_NAME}
  
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
      path: ${COMPUTE_WORKSPACE_PATH}
EOF

  echo -n "Waiting for APIBinding '${SYNC_TARGET}' to be bound:"
  while [[ -z "$(kubectl get apibindings.apis.kcp.dev ${SYNC_TARGET} -o jsonpath="{.status.phase}" --kubeconfig ${KCP_KUBECONFIG} | grep Bound)" ]]; do
    echo -n "."
    sleep 1
  done
  echo " OK"
  echo
  
  echo "Adding Role/RoleBindings for OpenShift GitOps in ${SP_WORKSPACE_NAME} workspace:"
  kubectl apply --kustomize $ROOT/openshift-gitops/in-kcp --kubeconfig ${KCP_KUBECONFIG}
  echo
  
  echo "Getting a token for argocd SA (in ${SP_WORKSPACE_NAME} workspace) - kubectl 1.24.x or newer needs to be used."
  SA_TOKEN=$(kubectl create token argocd --duration 876000h -n controllers-argocd-manager --kubeconfig ${KCP_KUBECONFIG})
  echo

  SECRET_NAME=${CLUSTER_SECRET_NAME_PREFIX}-workspace-${KCP_INSTANCE_NAME}
  echo "Creating ArgoCD secret with the name '${SECRET_NAME}' representing '${SP_WORKSPACE_NAME}' workspace with URL '${SP_WORKSPACE_URL}' in the compute OpenShift cluster for ${KCP_INSTANCE_NAME}:"
  cat <<EOF | kubectl apply --kubeconfig ${CLUSTER_KUBECONFIG} -f -
apiVersion: v1
kind: Secret
metadata:
  name: ${SECRET_NAME}
  namespace: openshift-gitops
  labels:
    argocd.argoproj.io/secret-type: cluster
type: Opaque
stringData:
  name: ${SECRET_NAME}
  server: '${SP_WORKSPACE_URL}'
  config: |
    {
      "bearerToken": "${SA_TOKEN}",
      "tlsClientConfig": {
        "insecure": true
      }
    }
EOF
}

source ${ROOT}/hack/flags.sh "The configure-kcp.sh configures the kcp instance with the needed workspaces and a workload cluster. The current context of the kcp kubeconfig should point to the kcp instance." extra_params extra_help
parse_flags $@

configure_compute_workspace

SP_WORKSPACE_NAME=${APPSTUDIO_WORKSPACE:-"redhat-appstudio"}
CLUSTER_SECRET_NAME_PREFIX="redhat-appstudio"
echo "Configuring service provider workspace for AppStudio '${SP_WORKSPACE_NAME}'"
configure_service_provider_workspace

SP_WORKSPACE_NAME=${HACBS_WORKSPACE:-"redhat-hacbs"}
CLUSTER_SECRET_NAME_PREFIX="redhat-hacbs"
echo "Configuring service provider workspace for HACBS '${SP_WORKSPACE_NAME}'"
configure_service_provider_workspace

echo
echo "Triggering hard refresh of all Applications:"
for APP in $(kubectl get apps -n openshift-gitops -o name --kubeconfig ${CLUSTER_KUBECONFIG}); do
  kubectl patch ${APP} -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}' --kubeconfig ${CLUSTER_KUBECONFIG}
done

echo "Triggering replace sync of all Applications:"
for APP in $(kubectl get apps -n openshift-gitops -o name --kubeconfig ${CLUSTER_KUBECONFIG}); do
  kubectl patch ${APP} -n openshift-gitops --type merge -p='{"operation": {"sync":{"syncOptions": ["Replace=true"]}}}' --kubeconfig ${CLUSTER_KUBECONFIG}
done
