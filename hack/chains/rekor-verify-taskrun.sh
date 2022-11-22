#!/bin/bash

source $(dirname $0)/_helpers.sh
set -ue

# Use a specific taskrun if provided, otherwise use the latest
TASKRUN_NAME=${1:-$( tkn taskrun describe --last -o name )}
TASKRUN_NAME=taskrun/$( trim-name $TASKRUN_NAME )

# Let's not hard code the image url or the registry
IMAGE_DIGEST=$( kubectl get $TASKRUN_NAME -o jsonpath='{.status.taskResults[?(@.name == "IMAGE_DIGEST")].value}' )
IMAGE_URL=$( kubectl get $TASKRUN_NAME -o jsonpath='{.status.taskResults[?(@.name == "IMAGE_URL")].value}' )
IMAGE_REGISTRY=$( echo $IMAGE_URL | cut -d/ -f1 )

TRANSPARENCY_URL=$(
  kubectl get $TASKRUN_NAME -o jsonpath='{.metadata.annotations.chains\.tekton\.dev/transparency}' )

# In the future we might use our own rekor servers, so let's not hard code that
REKOR_SERVER=$( echo $TRANSPARENCY_URL | cut -d/ -f1-3 )

if [[ $IMAGE_URL != null ]]; then
  title "Image url"
  # This link might not work but never mind
  echo https://$IMAGE_URL

  title "Lookup the transparency log entry for the image itself"
  # ...which is different to the transparency log entry for the taskrun
  $SCRIPTDIR/rekor-image-lookup.sh $IMAGE_DIGEST $REKOR_SERVER
  # This should work also
  #$SCRIPTDIR/rekor-image-lookup.sh $IMAGE_DIGEST $REKOR_SERVER
fi

# Extract the log index from the url
LOG_INDEX=$( echo $TRANSPARENCY_URL | cut -d= -f2 )

# Todo: We're reading the transparency url from the taskrun annotations.
# Is there another way to get it? How else can we link the image in the
# registry to its rekor entry?

title "Transparency url for $TASKRUN_NAME found in the annotations"
echo $TRANSPARENCY_URL
pause

title "Take a look at it"
curl-json $TRANSPARENCY_URL | yq-pretty
pause

# Todo: Should probably use rekor-cli here instead, e.g.:
#   rekor-cli get --log-index $LOG_INDEX --format json | jq ...
# The keys and format are slightly different.
#
BODY_DATA=$( curl-json $TRANSPARENCY_URL | jq -r 'values[].body' )
ATTESTATION_DATA=$( curl-json $TRANSPARENCY_URL | jq -r 'values[].attestation' )

title "Extract the rekor body"
echo "$BODY_DATA" | base64 -d | yq-pretty
pause

if [[ $ATTESTATION_DATA = '{}' ]]; then
  title "No attestation found"
else
  title "Extract the rekor attestation"
  # It really is base64 encoded twice here
  echo $ATTESTATION_DATA | jq -r .data | base64 -d | base64 -d | yq-pretty
  pause
fi

title "Using the rekor-cli"
show-then-run "rekor-cli get --log-index $LOG_INDEX --rekor_server $REKOR_SERVER"
pause

title "There's also a --format json option:"
rekor-cli get --log-index $LOG_INDEX --rekor_server $REKOR_SERVER --format json | yq-pretty
pause

title "Try a rekor-cli verify"
show-then-run "rekor-cli verify --log-index $LOG_INDEX --rekor_server $REKOR_SERVER"
