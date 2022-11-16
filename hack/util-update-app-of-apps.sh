#!/bin/bash

# Redirect the root app-of-apps to the users local git repo (usually a fork)
# if that repo is a simple clone this replacement is a noop
# if that repo is a fork, this repo will updated to the forked repo

# This allows any component to be replaced via gitops via a kustomize development directory
# That directory needs to be modified in the users fork when replacing any components in their cluster
# This will minimize the chance of error in pull requests to upstream which may accidentally include
# references to the forked repo.
# note, if accidental merges are accepted in the development directory, they will not affect staging. 

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
MANIFEST=$ROOT/argo-cd-apps/app-of-apps/all-applications.yaml
GITURL=$1
OVERLAYDIR=argo-cd-apps/overlays/$2

BRANCH=$3 
if [ -z "$BRANCH" ]
then
      echo No Branch specified, setting app-of-apps to kcp
      BRANCH=kcp
else  
      echo Setting app-of-apps targetRevision to $BRANCH  
fi

PATCHREPO="$(printf '.spec.source.repoURL="%q"' $GITURL)" 
PATCHOVERLAY="$(printf '.spec.source.path="%q"' $OVERLAYDIR)"  
PATCHBRANCH="$(printf '.spec.source.targetRevision="%q"' $BRANCH)"  

# the overlay content can be updated selectively per user in their fork
# to replace the specific component they are evolving  

KUBECONFIG_PARAM=""
if [[ -n ${CLUSTER_KUBECONFIG} ]]
then
  KUBECONFIG_PARAM="--kubeconfig ${CLUSTER_KUBECONFIG}"
fi
echo

echo "Setting the application repo to $GITURL, branch $BRANCH in overlay $OVERLAYDIR"
kubectl create -f $MANIFEST --dry-run=client -o json ${KUBECONFIG_PARAM}  | \
 jq "$PATCHOVERLAY" | \
 jq "$PATCHREPO" | \
 jq "$PATCHBRANCH" | \
 kubectl apply ${KUBECONFIG_PARAM} -f -
 