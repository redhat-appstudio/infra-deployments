#!/bin/bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source $SCRIPTDIR/_helpers.sh
set -ue

echo "To watch the chains controller logs:"
echo "  kubectl logs -f -l app=tekton-chains-controller -n tekton-chains"
pause

title "Set project"
# Todo: Make it work in any project
oc project tekton-chains

title "Ensure we have the demo kaniko build task"
kubectl apply -f https://raw.githubusercontent.com/tektoncd/chains/main/examples/kaniko/kaniko.yaml

# Tweak it to avoid TLS failures
# Fixme: Make it work without --skip-tls-verify
kubectl patch task/kaniko-chains --type=json \
  --patch='[{"op": "add", "path": "/spec/steps/1/args/5", "value":"--skip-tls-verify=true"}]'

IMAGE_REGISTRY=$( oc registry info )
DOCKERCFG=$( oc get sa tekton-chains-controller -o json | jq -r '.secrets[1].name' )

title "Run a build task and watch it"
tkn task start kaniko-chains \
  --param IMAGE=$IMAGE_REGISTRY/tekton-chains/kaniko-chains \
  --use-param-defaults \
  --workspace name=source,emptyDir= \
  --workspace name=dockerconfig,secret=$DOCKERCFG \
  --showlog
