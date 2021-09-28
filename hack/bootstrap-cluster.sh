#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

kubectl create namespace cluster-argocd

#kubectl apply -f $ROOT/../my-sealed-secret-key.yaml

# Install Argo CD
kustomize build $ROOT/argo-cd/ | kubectl apply -f -


echo
echo "Waiting for default project to exist:"

while : ; do
  kubectl get appproject/default -n cluster-argocd && break
  sleep 1
done

echo "Waiting for Argo CD Route"
while : ; do
  kubectl get routes -n cluster-argocd | grep "argocd-server" && break
  sleep 1
done


echo "Waiting for Argo CD Admin Secret"
while : ; do
  oc get secret argocd-initial-admin-secret -n cluster-argocd -o jsonpath='{.data.password}' && break
  sleep 1
done

# Add Argo CD Applications
kustomize build  $ROOT/argo-cd-apps/overlays/staging | kubectl apply -f -


ADMIN_SECRET=`oc get secret argocd-initial-admin-secret -n cluster-argocd -o jsonpath='{.data.password}' | base64 -d`
echo
echo "========================================================================="
echo
echo "Argo CD Route is:"
kubectl get routes -n cluster-argocd	
echo
echo "(It may take a few moments for the route to become available)"
echo
echo
echo "Argo CD admin login is: admin"
echo "Argo CD admin password is: $ADMIN_SECRET" 
echo



