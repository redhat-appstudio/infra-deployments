#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
TOOLCHAIN_E2E_TEMP_DIR=/tmp/toolchain-e2e

# Remove resources in the reverse order from bootstrapping

echo
echo "Remove Argo CD Applications:"
kubectl delete -f $ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml
#kustomize build  $ROOT/argo-cd-apps/overlays/staging | kubectl delete -f -

echo
echo "Remove RBAC for OpenShift GitOps:"
kustomize build $ROOT/openshift-gitops/cluster-rbac | kubectl delete -f -

echo 
echo "Remove the OpenShift GitOps operator subscription:"
kubectl delete -f $ROOT/openshift-gitops/subscription-openshift-gitops.yaml

echo 
echo "Removing GitOps operator and operands:"

while : ; do
  kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"metadata":{"finalizers":null}}' --type=merge
  kubectl delete csv/openshift-gitops-operator.v1.3.0 -n openshift-operators
  kubectl delete csv/openshift-gitops-operator.v1.3.0 -n openshift-gitops
  kubectl delete csv/openshift-gitops-operator.v1.3.1 -n openshift-operators
  kubectl delete csv/openshift-gitops-operator.v1.3.1 -n openshift-gitops
  kubectl delete namespace --timeout=30s openshift-gitops

  kubectl get namespace/openshift-gitops
  RC=$?
  if [ "${RC}" != 0 ]; then
    break
  fi

  echo $RC
done

echo
echo "Remove Toolchain (Sandbox) Operators with the user data:"
rm -rf ${TOOLCHAIN_E2E_TEMP_DIR} 2>/dev/null || true
git clone --depth=1 https://github.com/codeready-toolchain/toolchain-e2e.git ${TOOLCHAIN_E2E_TEMP_DIR}
make -C ${TOOLCHAIN_E2E_TEMP_DIR} appstudio-cleanup

echo 
echo "Complete."