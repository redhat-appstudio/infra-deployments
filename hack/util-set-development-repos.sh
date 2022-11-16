
#!/bin/bash

# Redirect the root app-of-apps to the users local git repo (usually a fork)
# if that repo is a simple clone this replacement is a noop
# if that repo is a fork, this repo will updated to the forked repo

# This allows any component to be replaced via gitops via a kustomize base development directory
# That directory needs to be modified in the users fork to replace replacing any components in their cluster
# This will minimize the chance of error in pull requests to upstream which may accidentally include
# references to the forked repo.
# note, if accidental merges are accepted in the development directory, they will not affect staging. 

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
MANIFEST=$ROOT/argo-cd-apps/app-of-apps/all-applications.yaml
GITURL=$1
OVERLAYDIR=$ROOT/argo-cd-apps/overlays/$2
BRANCH=$3 
if [ -z "$BRANCH" ]
then
      echo No Branch specified, setting all overlays targetRevisions to kcp
      BRANCH=kcp
else  
      echo Setting all overlays targetRevisions to $BRANCH 
fi
echo
echo In dev mode, verify that argo-cd-apps/overlays/development includes a kustomization that points to this repo

yq e "select(.kind == \"Application\") |= with(.spec.source; .repoURL = \"$GITURL\" | .targetRevision = \"$BRANCH\")" $OVERLAYDIR/repo-overlay.yaml -i
yq e "select(.kind == \"ApplicationSet\") |= with(.spec.template.spec.source; .repoURL = \"$GITURL\" | .targetRevision = \"$BRANCH\")" $OVERLAYDIR/repo-overlay.yaml -i

echo
echo The list of components which will be patched is
yq e --no-doc '.metadata.name' $OVERLAYDIR/repo-overlay.yaml
echo
if [ -n "$DEPLOY_ONLY" ]; then
    for APP in $(yq e -N '.metadata.name' $OVERLAYDIR/repo-overlay.yaml); do
        if ! grep "\b$APP\b" <<< $DEPLOY_ONLY; then
           echo Disabling $APP based on DEPLOY_ONLY variable
           cat >> $OVERLAYDIR/delete-applications.yaml <<EOF
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: $APP
\$patch: delete
EOF
        fi
    done
fi
