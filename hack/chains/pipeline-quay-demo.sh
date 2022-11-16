#!/bin/bash

source $(dirname $0)/_helpers.sh
set -e

title "Setting project"
# Todo: Make it work in any project
oc project tekton-chains

QUAY_IMAGE=$1
QUAY_SECRET_NAME=$2

if [[ -z "$QUAY_IMAGE" ]] || [[ -z "$QUAY_SECRET_NAME" ]]; then
  echo "Example usage:"
  echo "  $0 quay.io/sbaird/chains-demo sbaird-chains-demo-pull-secret-name"
  exit 1
fi

if ! kubectl get secret/$QUAY_SECRET_NAME -o name; then
  echo "Can't find $QUAY_SECRET_NAME!"
  echo "Please ensure it exists and try again!"
  echo "For example:"
  echo "  kubectl create -f sbaird-chains-demo-secret.yml"
  exit 1
fi

title "Quay image"
echo $QUAY_IMAGE

title "Image push secret name"
echo $QUAY_SECRET_NAME

title "Suggested config for this demo:"
$SCRIPTDIR/config.sh default --dry-run

title "Current config:"
$SCRIPTDIR/config.sh

title "To watch the chains controller logs:"
echo "  kubectl logs -f -l app=tekton-chains-controller -n tekton-chains | sed G"

pause

title "Creating and configuring pipeline for demo"
kubectl apply -f $SCRIPTDIR/pipeline-quay-demo.yaml

title "Adding secret to tekton-chains service account"
# Chains also needs the quay secret so it can push signatures
# and (in future maybe) attestations
kubectl patch sa pipeline -n tekton-chains \
  -p "{\"imagePullSecrets\": [{\"name\": \"$QUAY_SECRET_NAME\"}]}"

title "Starting pipeline run"
tkn pipeline start \
  --param IMAGE="$QUAY_IMAGE" \
  --param PUSH_SECRET_NAME="$QUAY_SECRET_NAME" \
  --showlog \
  -w name=source,pvc,claimName="ci-builds" \
  chains-demo-pipeline

title "Expecting image to appear here"
# The url isn't quite right, but it redirects to the correct one
echo "https://$QUAY_IMAGE"

# Wait a few seconds to let the registry finish processing (I guess?)
sleep 3

# Use this script to verify the image and show cosign output.
# (It appears cosign does require a quay.io login here, so pass
# in the secret name to make sure we have one. Not sure why that is
# since the repo is public.)
title "Verify image with cosign"
$SCRIPTDIR/cosign-verify-image.sh $QUAY_IMAGE $QUAY_SECRET_NAME

pause

title "Verify taskrun with rekor-cli"
# Should find the most recent taskrun
$SCRIPTDIR/rekor-verify-taskrun.sh
