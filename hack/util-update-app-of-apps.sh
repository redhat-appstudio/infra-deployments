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
MANIFEST=$ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml
GITURL=$1
OVERLAYDIR=argo-cd-apps/overlays/$2

PATCHREPO="$(printf '.spec.source.repoURL="%q"' $GITURL)" 
PATCHOVERLAY="$(printf '.spec.source.path="%q"' $OVERLAYDIR)"  

# the overlay content can be updated selectively per user in their fork
# to replace the specific component they are evolving  

echo
echo "Setting the application repo to $GITURL overlay to $OVERLAYDIR"  
yq  e "$PATCHOVERLAY" $MANIFEST | yq  e "$PATCHREPO" - | kubectl apply -f -
 