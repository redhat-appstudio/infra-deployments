#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

kubectl create namespace cluster-argocd

#kubectl apply -f $ROOT/../my-sealed-secret-key.yaml

# Install Argo CD
kustomize build $ROOT/argo-cd/ | kubectl apply -f -
#kubectl apply -f $ROOT/../../argo-cd-secret.yaml

echo
echo "Waiting for default project to exist:"

while : ; do
  kubectl get appproject/default -n cluster-argocd && break
  sleep 1
done

# Add Argo CD Applications
kustomize build  $ROOT/argo-cd-apps/overlays/staging | kubectl apply -f -


