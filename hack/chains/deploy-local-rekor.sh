#!/bin/bash
#
# A wrapper for deploying rekor into a local cluster.
# Uses the deploy-local-rekor.tpl file, which has a placeholder for
# {domain} which is replaced by getting the domain from the openshift
# cluster. 

# Make sure we have our template file
if [[ ! -f $PWD/deploy-local-rekor.tpl ]];then
  echo "Template file 'deploy-local-rekor.tpl' not found in this directory "; exit 1 
fi

# Test if we have the argocd cli command:
if ! command -v argocd &> /dev/null; then
  echo "The argocd CLI is not installed. Please install and try again."
  echo "See https://argo-cd.readthedocs.io/en/stable/getting_started/#2-download-argo-cd-cli for details"
  exit 1
fi

# Test if we are logged into our cluster as an account with a 'cluster-admin' role.
if [ "$(oc auth can-i '*' '*' --all-namespaces 2> /dev/null)" != "yes" ]; then
  echo
  echo "[ERROR] User '$(oc whoami)' does not have the required 'cluster-admin' role." 1>&2
  echo "Log into the cluster with a user with the required privileges (e.g. kubeadmin) and retry."
  exit 1
fi

# Get our domain from the ingresses config of the cluster
domain=$( kubectl get ingresses.config.openshift.io cluster -o json|jq -r ".spec.domain" )
if [[ $1 == "--verbose" ]]; then
  echo Domain: $domain
fi

# Set the template input file
input_file="$PWD/deploy-local-rekor.tpl"

output=$( sed -e "s/\${domain}/$domain/" $input_file )

# If we want to see what's being written to the deployment file
if [[ $1 == "--verbose" ]]; then
  echo "Displaying rekor-server configuration yaml"
  echo ""
  echo "--------"
  echo "$output" | yq e -P - 
  echo "--------"
fi

# Ensure we're actually logged in before trying to execute argocd command

# Find the list of argocd apps to test logged in status

ARGOAPPS=$(argocd app list -o name --grpc-web 2> /dev/null)

if [[ $? != 0 ]]; then
  # Assume the user is not logged in.
  # Try to be helpful.

  ARGO_HOST=$(
    kubectl get route/openshift-gitops-server -n openshift-gitops \
      -o jsonpath='{.spec.host}' )

  LOGIN_CMD="argocd login $ARGO_HOST --sso --grpc-web"

  echo
  echo "*** Please log in with argocd and try again ***"
  echo
  echo "Running login command:"
  echo "  $LOGIN_CMD"
  echo
  $LOGIN_CMD
  echo
  exit 1
fi

echo "$output" | argocd app create --grpc-web --upsert -f - 

# Let the user know we've created the rekor app, set expectations, and 
# test if it's up and
# running.
echo """
Created application "rekor". Syncing may take a few moments.
Waiting to verify application is available.
"""

# Try accessing the log via the URL
# curl https://rekor-server.$domain/api/v1/log
while ! curl --fail --insecure --output /dev/null --silent "https://rekor-server.$domain/api/v1/log"; do
  echo -n .
  sleep 3
done

echo "Done"
