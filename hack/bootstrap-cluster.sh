#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..


echo 
echo "Installing the OpenShift GitOps operator subscription:"
kubectl apply -f $ROOT/openshift-gitops/subscription-openshift-gitops.yaml

echo
echo "Waiting for default project (and namespace) to exist:"
while : ; do
  kubectl get appproject/default -n openshift-gitops && break
  sleep 1
done

echo
echo "Waiting for OpenShift GitOps Route:"
while : ; do
  kubectl get route/openshift-gitops-server -n openshift-gitops && break
  sleep 1
done

echo 
echo "Add Role/RoleBindings for OpenShift GitOps:"
kustomize build $ROOT/openshift-gitops/cluster-rbac | kubectl apply -f -

echo
echo "Add parent Argo CD Application:"
kubectl apply -f $ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml

echo
echo "========================================================================="
echo
echo "Argo CD Route is:"
kubectl get route/openshift-gitops-server -n openshift-gitops
echo
echo "(NOTE: It may take a few moments for the route to become available)"
echo
echo "Login/password uses your OpenShift credentials ('Login with OpenShift' button)"
echo



