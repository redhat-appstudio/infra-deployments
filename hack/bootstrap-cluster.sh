#!/bin/bash -e

MODE=$1

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

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
' --type=merge

echo 
echo "Add Role/RoleBindings for OpenShift GitOps:"
kubectl apply --kustomize $ROOT/openshift-gitops/cluster-rbac

echo "Setting secrets for Tekton Results"
kubectl create namespace tekton-pipelines -o yaml --dry-run=client | oc apply -f-

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
kubectl create namespace gitops -o yaml --dry-run=client | oc apply -f-
if ! kubectl get secret -n gitops gitops-postgresql-staging &>/dev/null; then
  kubectl create secret generic gitops-postgresql-staging \
    --namespace=gitops \
    --from-literal=postgresql-password=$(openssl rand -base64 20)
fi

echo
echo "Setting secrets for Quality Dashboard"
kubectl create namespace quality-dashboard -o yaml --dry-run=client | oc apply -f-
if ! kubectl get secret -n quality-dashboard quality-dashboard-secrets &>/dev/null; then
  kubectl create secret generic quality-dashboard-secrets \
    --namespace=quality-dashboard \
    --from-literal=rds-endpoint=REPLACE_WITH_RDS_ENDPOINT \
    --from-literal=storage-user=postgres \
    --from-literal=storage-password=REPLACE_DB_PASSWORD \
    --from-literal=storage-database=quality \
    --from-literal=github-token=REPLACE_GITHUB_TOKEN \
    --from-literal=jira-token=REPLACE_JIRA_TOKEN
fi

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
echo "Setting Cluster Mode: ${MODE:-Upstream}"
case $MODE in
    ""|"upstream")
        kubectl apply -f $ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml
        # Check if we have a tekton-chains namespace, and if so, remove any explicit transparency.url setting
        # which might be left from running this script with the 'preview' flag to enable the cluster local
        # rekor instance. By default, chains will use the publicly accessible sandbox instance hosted by sigstore
        # of rekor
        # If, in the future, we're boostrapping to use a Red Hat internal rekor instance instead of the public,
        # default sandbox instance hosted by sigstore we would want to reconsider this approach.
        if kubectl get namespace tekton-chains &> /dev/null; then
          # Remove our transparency.url, if present, to ensure we're not using the cluster local rekor
          # which is only available in 'preview' mode.
          kubectl patch configmap/chains-config -n tekton-chains --type=json --patch '[{"op":"remove","path":"/data/transparency.url"}]'
          kubectl delete pod -n tekton-chains -l app=tekton-chains-controller
        fi
        ;;
    "preview")
        $ROOT/hack/preview.sh ;;
esac
