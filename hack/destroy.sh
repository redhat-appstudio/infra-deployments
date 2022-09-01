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
  echo "-m,  --mode                   The mode to be used when cleaning up kcp workspaces (default is 'upstream')"
  echo "-sk, --skip-kcp               If set to true, then it doesn't clean kcp workspaces."
}

source ${ROOT}/hack/flags.sh "The destroy.sh script deletes kcp workspaces and removes ArgoCD from Openshift cluster." extra_params extra_help
parse_flags $@

# Remove resources in the reverse order from bootstrapping

clean_kcp() {
  if [[ "${SKIP_KCP}" == "true" ]]
  then
    echo "!!! WARNING: Skipping kcp cleanup - it won't delete any workspace in '${1}'. !!!"
  else
    if [[ ${2} == "true" ]]
    then
      kubectl config use ${1}
    fi
    echo
    echo "Using the '${ROOT_WORKSPACE}' workspace as the root"
    KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}
    echo

    if [[ ${ROOT_WORKSPACE} == "root" ]]
    then
      COMPUTE_WORKSPACE=${COMPUTE_WORKSPACE:-"redhat-appstudio-internal-compute"}
    else
      COMPUTE_WORKSPACE=${COMPUTE_WORKSPACE:-"compute"}
    fi
    echo "Removing '${COMPUTE_WORKSPACE}' workspace:"
    KUBECONFIG=${KCP_KUBECONFIG} kubectl delete workspace ${COMPUTE_WORKSPACE}
    echo

    APPSTUDIO_WORSKPACE=${APPSTUDIO_WORSKPACE:-"redhat-appstudio"}
    echo "Removing '${COMPUTE_WORKSPACE}' workspace:"
    KUBECONFIG=${KCP_KUBECONFIG} kubectl delete workspace ${APPSTUDIO_WORSKPACE}
    echo
  fi
}

echo
echo "Using mode: ${MODE:-Upstream}"
case $MODE in
    ""|"upstream")
        clean_kcp kcp-unstable "true"
        clean_kcp kcp-stable "true"
        ;;
    *)
        clean_kcp dev
        ;;
esac


ARGO_CD_ROUTE=$(kubectl get --kubeconfig ${CLUSTER_KUBECONFIG} \
                 -n openshift-gitops \
                 -o template \
                 --template={{.spec.host}} \
                 route/openshift-gitops-server \
               )
ARGO_CD_URL="https://${ARGO_CD_ROUTE}"

echo
echo "Removing applications, you can see progress ${ARGO_CD_ROUTE}"
echo "If there is running Sync then cancel it manually"
echo "Remove Argo CD Applications:"
kubectl delete -f ${ROOT}/argo-cd-apps/app-of-apps/all-applications.yaml --kubeconfig ${CLUSTER_KUBECONFIG}
kubectl wait --for=delete -f ${ROOT}/argo-cd-apps/app-of-apps/all-applications.yaml --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Remove RBAC for OpenShift GitOps:"
kubectl delete -k ${ROOT}/openshift-gitops/cluster-rbac --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Remove the OpenShift GitOps instance:"
kubectl delete gitopsservice cluster -n openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG}

echo 
echo "Remove the OpenShift GitOps operator subscription:"
kubectl delete -f ${ROOT}/openshift-gitops/subscription-openshift-gitops.yaml --kubeconfig ${CLUSTER_KUBECONFIG}

echo 
echo "Removing operators and operands:"
oc delete clusterserviceversions.operators.coreos.com --all -n openshift-operators --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Wait until ArgoCD instance is gone:"
oc wait --for=delete argocd openshift-gitops -n openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Remove openshift-gitops namespace:"
oc delete namespace openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG}
oc wait --for=delete namespace openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Remove openshift-gitops namespace:"
oc delete namespace openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG}
oc wait --for=delete namespace openshift-gitops --kubeconfig ${CLUSTER_KUBECONFIG}

echo
echo "Remove all generated kcp namespaces:"
oc delete namespace -l internal.workload.kcp.dev/cluster --kubeconfig ${CLUSTER_KUBECONFIG}
oc wait --for=delete namespace -l internal.workload.kcp.dev/cluster --kubeconfig ${CLUSTER_KUBECONFIG}
oc delete namespace -l workload.kcp.io/sync-target --kubeconfig ${CLUSTER_KUBECONFIG}
oc wait --for=delete namespace -l workload.kcp.io/sync-target --kubeconfig ${CLUSTER_KUBECONFIG}

echo 
echo "Complete."
