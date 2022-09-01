#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

function extra_params() {
  case "$1" in
    -m|--mode)
      shift
      MODE=$1
      shift
      ;;
    -sk|--skip-kcp)
      shift
      SKIP_KCP=${1:-"true"}
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
  echo "-m,  --mode                   The mode to be used when applying AppStudio component manifests (default is 'upstream')"
  echo "-sk, --skip-kcp               If set to true, then it doesn't configure kcp workspaces nor any ArgoCD cluster."
}

source ${ROOT}/hack/flags.sh "The bootstrap.sh script installs and configures ArgoCD in Openshift cluster and configures service provider workspaces." extra_params extra_help
parse_flags $@

if [ "$(oc auth can-i '*' '*' --all-namespaces --kubeconfig ${CLUSTER_KUBECONFIG})" != "yes" ]; then
  echo
  echo "[ERROR] User '$(oc whoami --kubeconfig ${CLUSTER_KUBECONFIG})' does not have the required 'cluster-admin' role." 1>&2
  echo "Log into the cluster with a user with the required privileges (e.g. kubeadmin) and retry."
  exit 1
fi

echo
echo "Installing the OpenShift GitOps operator subscription:"
kubectl apply -f $ROOT/openshift-gitops/subscription-openshift-gitops.yaml --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo -n "Waiting for 'appproject/default' and namespace 'openshift-gitops' to exist: "
while ! kubectl get appproject/default -n openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG} &> /dev/null ; do
  echo -n .
  sleep 1
done
echo "OK"

echo
echo -n "Waiting for OpenShift GitOps Route: "
while ! kubectl get route/openshift-gitops-server -n openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG} &> /dev/null ; do
  echo -n .
  sleep 1
done
echo "OK"

echo
echo "Patching OpenShift GitOps ArgoCD CR"

# Switch the Route to use re-encryption
kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec": {"server": {"route": {"enabled": true, "tls": {"termination": "reencrypt"}}}}}' --type=merge  --kubeconfig ${CLUSTER_KUBECONFIG}

# Allow any authenticated users to be admin on the Argo CD instance
# - Once we have a proper access policy in place, this should be updated to be consistent with that policy.
kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec":{"rbac":{"policy":"g, system:authenticated, role:admin"}}}' --type=merge --kubeconfig ${CLUSTER_KUBECONFIG}

# Mark Pending PVC as Healthy, workaround for WaitForFirstConsumer StorageClasses.
# If the attachment will fail then it will be visible on the pod anyway.
kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  resourceCustomizations: |
    PersistentVolumeClaim:
      health.lua: |
        hs = {}
        if obj.status ~= nil then
          if obj.status.phase ~= nil then
            if obj.status.phase == "Pending" then
              hs.status = "Healthy"
              hs.message = obj.status.phase
              return hs
            end
            if obj.status.phase == "Bound" then
              hs.status = "Healthy"
              hs.message = obj.status.phase
              return hs
            end
          end
        end
        hs.status = "Progressing"
        return hs
' --type=merge --kubeconfig ${CLUSTER_KUBECONFIG}

# Exclude tenancy.kcp.dev API as ArgoCD won't probably have enough permissions for all kinds in the group (and we doesn't need to sync it anyway).
kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  resourceExclusions: |
    - apiGroups:
      - tekton.dev
      clusters:
      - "*"
      kinds:
      - TaskRun
      - PipelineRun
    - apiGroups:
      - tenancy.kcp.dev
      clusters:
      - "*"
      kinds:
      - "*"
' --type=merge --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Add Role/RoleBindings for OpenShift GitOps:"
kubectl apply --kustomize $ROOT/openshift-gitops/cluster-rbac --kubeconfig ${CLUSTER_KUBECONFIG}

ARGO_CD_ROUTE=$(kubectl get --kubeconfig ${CLUSTER_KUBECONFIG} \
                 -n openshift-gitops \
                 -o template \
                 --template={{.spec.host}} \
                 route/openshift-gitops-server \
               )
ARGO_CD_URL="https://$ARGO_CD_ROUTE"

echo
echo "========================================================================="
echo
echo "Argo CD URL is: $ARGO_CD_URL"
echo
echo "(NOTE: It may take a few moments for the route to become available)"
echo
echo -n "Waiting for the route: "
while ! curl --fail --insecure --output /dev/null --silent "$ARGO_CD_URL"; do
  echo -n .
  sleep 3
done
echo "OK"
echo
echo "Login/password uses your OpenShift credentials ('Login with OpenShift' button)"
echo
echo "========================================================================="
echo


configure_kcp() {
  if [[ "${SKIP_KCP}" == "true" ]]
  then
    echo "!!! WARNING: Skipping kcp configuration - it won't configure any workspace in '${1}' nor ArgoCD cluster for it. !!!"
  else
    if [[ ${2} == "true" ]]
    then
      kubectl config use ${1}
    fi
    source ${ROOT}/hack/configure-kcp.sh -kn ${1}
  fi
}

echo
echo "Using mode: ${MODE:-Upstream}"
case $MODE in
    ""|"upstream")
        configure_kcp kcp-unstable "true"
        configure_kcp kcp-stable "true"
        kubectl apply -f $ROOT/argo-cd-apps/app-of-apps/all-applications.yaml --kubeconfig ${CLUSTER_KUBECONFIG}
        ;;
    "dev")
        configure_kcp dev
        kubectl apply -f $ROOT/argo-cd-apps/app-of-apps/all-applications.yaml --kubeconfig ${CLUSTER_KUBECONFIG}

        if [ -z "${MY_GIT_REPO_URL}" ]; then
            MY_GIT_REPO_URL=$(git --git-dir=${ROOT}/.git --work-tree=${ROOT}  ls-remote --get-url| sed 's|^git@github.com:|https://github.com/|')
        fi
        if [ -z "${MY_GIT_BRANCH}" ]; then
            MY_GIT_BRANCH=$(git  --git-dir=${ROOT}/.git --work-tree=${ROOT} rev-parse --abbrev-ref HEAD)
        fi

        echo "Redirecting the root app-of-apps to use the git repo '${MY_GIT_REPO_URL}' and branch '${MY_GIT_BRANCH}' and updating the path to development:"
        $ROOT/hack/util-update-app-of-apps.sh ${MY_GIT_REPO_URL} development ${MY_GIT_BRANCH}
        echo

        echo "Resetting the default repos in the development directory to be the current git repo:"
        echo "These changes need to be pushed to your fork to be seen by argocd"
        $ROOT/hack/util-set-development-repos.sh ${MY_GIT_REPO_URL} development ${MY_GIT_BRANCH}
        ;;
    "preview")
        $ROOT/hack/preview.sh ;;
esac
