#!/bin/bash
#
# Use this to switch Argo CD's automatic syncing and self healing
# off and on again. Useful if you want to modify the configuration
# for tekton chains.
#

if [[ $1 != off ]] && [[ $1 != on ]]; then
  echo "Usage:"
  echo " $0 off|on"
  exit 1
fi

# Find the list of argo cd apps
ARGO_APPS=$( argocd app list -o name )

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

# Actually I only care about these two apps for chains config.
# Comment this out if you want to disable syncing for everything.
ARGO_APPS="all-components-staging build enterprise-contract rekor"

# Now apply the change
for app in $ARGO_APPS; do

  if [[ $1 == off ]]; then
    echo "* Disabling automated syncing for '$app'"
    argocd app set $app --sync-policy none

  elif [[ $1 == on ]]; then
    echo "* Enabling automated syncing for '$app'"
    argocd app set $app --sync-policy automated --self-heal

  fi

done
