#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

# Remove resources in the reverse order from bootstrapping

echo
echo "Remove Argo CD Applications:"
kustomize build  $ROOT/argo-cd-apps/overlays/staging | kubectl delete -f -

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
  kubectl delete namespace --timeout=30s openshift-gitops

  kubectl get namespace/openshift-gitops
  RC=$?
  if [ "${RC}" != 0 ]; then
    break
  fi

  echo $RC
done

echo 
echo "Complete."