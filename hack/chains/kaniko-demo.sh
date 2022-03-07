#!/bin/bash

source $(dirname $0)/_helpers.sh
set -ue

title "Suggested config for this demo:"
$SCRIPTDIR/config.sh default --dry-run

title "Current config:"
$SCRIPTDIR/config.sh

title "To watch the chains controller logs:"
echo "  kubectl logs -f -l app=tekton-chains-controller -n tekton-chains | sed G"

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

title "Wait a few seconds for chains finalizers to complete"
sleep 10

# This will use cosign to verify the new build
$SCRIPTDIR/kaniko-cosign-verify.sh

pause

# This will use rekor-cli to verify the new build
$SCRIPTDIR/rekor-verify-taskrun.sh
