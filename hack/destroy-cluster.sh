#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
TOOLCHAIN_E2E_TEMP_DIR=/tmp/toolchain-e2e

# Remove resources in the reverse order from bootstrapping

# Todo: Is there anything related to tekton-chains that needs removing here?

echo
echo "Remove Argo CD Applications:"
kubectl delete -f $ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml
#kubectl -k $ROOT/argo-cd-apps/overlays/staging

echo
echo "Remove RBAC for OpenShift GitOps:"
kubectl delete -k $ROOT/openshift-gitops/cluster-rbac

echo 
echo "Remove the OpenShift GitOps operator subscription:"
kubectl delete -f $ROOT/openshift-gitops/subscription-openshift-gitops.yaml

echo 
echo "Removing GitOps operator and operands:"

while : ; do
  kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"metadata":{"finalizers":null}}' --type=merge
  OPERATORS=$(oc get clusterserviceversions.operators.coreos.com -o name)
  for OPERATOR in $OPERATORS; do
      if echo $OPERATOR | grep -q openshift-gitops-operator; then
          kubectl delete $OPERATOR
      fi
  done
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
