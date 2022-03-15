#!/bin/bash

MODE=$1

ROOT="$(realpath -mq ${BASH_SOURCE[0]}/../..)"

if [ "$(oc auth can-i '*' '*' --all-namespaces)" != "yes" ]; then
  echo
  echo "[ERROR] User '$(oc whoami)' does not have the required 'cluster-admin' role." 1>&2
  echo "Log into the cluster with a user with the required privileges (e.g. kubeadmin) and retry."
  exit 1
fi

echo 
echo "Installing the OpenShift GitOps operator subscription:"
kubectl apply -f $ROOT/openshift-gitops/subscription-openshift-gitops.yaml

echo
echo -n "Waiting for default project (and namespace) to exist: "
while ! kubectl get appproject/default -n openshift-gitops &> /dev/null ; do
  echo -n .
  sleep 1
done
echo "OK"

echo
echo -n "Waiting for OpenShift GitOps Route: "
while ! kubectl get route/openshift-gitops-server -n openshift-gitops &> /dev/null ; do
  echo -n .
  sleep 1
done
echo "OK"

echo
echo "Patching OpenShift GitOps ArgoCD CR"

# Switch the Route to use re-encryption
kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec": {"server": {"route": {"enabled": true, "tls": {"termination": "reencrypt"}}}}}' --type=merge

# Allow any authenticated users to be admin on the Argo CD instance
# - Once we have a proper access policy in place, this should be updated to be consistent with that policy.
kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec":{"rbac":{"policy":"g, system:authenticated, role:admin"}}}' --type=merge

echo 
echo "Add Role/RoleBindings for OpenShift GitOps:"
kubectl apply --kustomize $ROOT/openshift-gitops/cluster-rbac

echo "Setting secrets for Tekton Results"
if ! kubectl get namespace tekton-pipelines &>/dev/null; then
  kubectl create namespace tekton-pipelines
fi

OPENSSLDIR=`openssl version -d | cut -f2 -d'"'`
 
if ! kubectl get secret -n tekton-pipelines tekton-results-tls &>/dev/null; then
  ROUTE=$(oc whoami --show-console | sed 's|https://console-openshift-console|api-tekton-pipelines|')
  openssl req -x509 \
    -newkey rsa:4096 \
    -keyout key.pem \
    -out cert.pem \
    -days 3650 \
    -nodes \
    -subj "/CN=tekton-results-api-service.tekton-pipelines.svc.cluster.local" \
    -reqexts SAN \
    -extensions SAN \
    -config <(cat ${OPENSSLDIR:-/etc/pki/tls}/openssl.cnf \
        <(printf "\n[SAN]\nsubjectAltName=DNS:tekton-results-api-service.tekton-pipelines.svc.cluster.local, DNS:$ROUTE"))
  kubectl create secret tls -n tekton-pipelines tekton-results-tls --cert=cert.pem --key=key.pem
  rm cert.pem key.pem
fi
if ! kubectl get secret -n tekton-pipelines tekton-results-postgres &>/dev/null; then
  kubectl create secret generic tekton-results-postgres \
    --namespace="tekton-pipelines" \
    --from-literal=POSTGRES_USER=results \
    --from-literal=POSTGRES_PASSWORD=$(openssl rand -base64 20)
fi

echo
echo "Setting secrets for GitOps"
if ! kubectl get namespace gitops &>/dev/null; then
  kubectl create namespace gitops
fi
if ! kubectl get secret -n gitops gitops-postgresql-staging &>/dev/null; then
  kubectl create secret generic gitops-postgresql-staging \
    --namespace=gitops \
    --from-literal=postgresql-password=$(openssl rand -base64 20)
fi

echo
echo "Setting Cluster Mode: ${MODE:-Upstream}"
case $MODE in
    ""|"upstream")
        kubectl apply -f $ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml ;;
    "development")
        $ROOT/hack/development-mode.sh ;;
    "preview")
        $ROOT/hack/preview.sh ;;
esac

ARGO_CD_ROUTE=$(kubectl get \
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
