#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/
GITURL=$1
BRANCH=$2
if [ -z "$BRANCH" ]
then
      echo No Branch specified, setting all overlays targetRevisions to main 
      BRANCH=main 
else  
      echo Setting all overlays targetRevisions to $BRANCH 
fi

PATCH="$(printf '.spec.url="%q"' $GITURL)" 
yq  e "$PATCH" "${ROOT}/pipelines-as-code-overlay.yaml"  -i

