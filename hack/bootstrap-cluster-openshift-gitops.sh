#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..


echo 
echo "Installing the OpenShift GitOps operator subscription"
kubectl apply -f $ROOT/openshift-gitops/subscription-openshift-gitops.yaml

echo
echo "Waiting for default project to exist:"
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

# Add RBAC for OpenShift GitOps
kustomize build $ROOT/openshift-gitops/cluster-rbac | kubectl apply -f -


# Add Argo CD Applications
kustomize build  $ROOT/argo-cd-apps/overlays/staging | kubectl apply -f -

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



