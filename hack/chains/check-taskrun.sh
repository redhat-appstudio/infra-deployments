#!/bin/bash
#
# This script is only for use with chains configured with
# 'tekton' taskrun storage, which means the chains data is
# stored in the taskrun annotations.
#
# It's probably not useful for 'oci' taskrun storage.
#

source $(dirname $0)/_helpers.sh
set -ue

# Use a specific taskrun if provided, otherwise use the latest
TASKRUN_NAME=${1:-$( tkn taskrun describe --last -o name )}
TASKRUN_NAME=taskrun/$( trim-name $TASKRUN_NAME )

title Taskrun name
say $TASKRUN_NAME

# Helper for jsonpath
get-jsonpath() {
  kubectl get $TASKRUN_NAME -o jsonpath={.$1}
}

# Helper for reading chains values
get-chainsval() {
  get-jsonpath metadata.annotations.chains\\.tekton\\.dev/$1
}

# Helper for reading a task result
get-taskresult() {
  kubectl get $TASKRUN_NAME \
    -o jsonpath="{.status.taskResults[?(@.name == \"$1\")].value}"
}

# Helper for reading a resources result
get-resourcesresult() {
  kubectl get $TASKRUN_NAME \
    -o jsonpath="{.status.resourcesResult[?(@.key == \"$1\")].value}"
}

# Fetch task run signature and payload
TASKRUN_UID=$( get-jsonpath metadata.uid )
SIGNATURE=$( get-chainsval signature-taskrun-$TASKRUN_UID )
RAW_PAYLOAD=$( get-chainsval payload-taskrun-$TASKRUN_UID | base64 --decode )
PAYLOAD=$RAW_PAYLOAD

# For a task that builds an image, image digest should be available
# in the task results
IMAGE_DIGEST=$( get-taskresult IMAGE_DIGEST )

# Another place we might find it..?
# (The taskrun created by simple-demo.sh produces a digest here.)
[[ -z $IMAGE_DIGEST ]] && IMAGE_DIGEST=$( get-resourcesresult digest )

if [[ -n $IMAGE_DIGEST ]]; then
  SHORT_IMAGE_DIGEST=$( echo "$IMAGE_DIGEST" | cut -d: -f2 | head -c 12 )

  # For tekton storage, we should then be able to grab these
  IMAGE_SIGNATURE=$( get-chainsval signature-$SHORT_IMAGE_DIGEST )
  IMAGE_PAYLOAD=$( get-chainsval payload-$SHORT_IMAGE_DIGEST | base64 --decode)

  title "Image digest found in taskrun"
  say "Image digest: $IMAGE_DIGEST"

  if [[ -n $IMAGE_SIGNATURE ]] || [[ -n $IMAGE_PAYLOAD ]]; then
    say "Image signature: $IMAGE_SIGNATURE"
    say "Image signed payload:"
    [[ -z $QUIET ]] && echo "$IMAGE_PAYLOAD" | yq-pretty
  fi

  # Todo: How to verify these also?
fi

# Try to detect and then handle different formats
# (It seems like it would be better if the format was available
# explicitly but afaict it is not.)
# In the longer term, we won't care about tekton format, so let's
# not worry too much.

# If the signature is 96 chars then we might be using tekton format
if [[ ${#SIGNATURE} == 96 ]]; then
  title "Assuming tekton format"
  # The signature is just the signature, continue

else
  title "Assuming in-toto format for taskrun"

  # The signature value is actually encoded json with a payload and
  # signature list inside it
  SIG_DATA=$( echo $SIGNATURE | base64 --decode )
  SIGNATURE=$( echo $SIG_DATA | jq -r '.signatures[0].sig' )

  # Looks like the same payload can be found in both places...
  # Not sure if feature or bug
  OTHER_PAYLOAD=$( echo $SIG_DATA | jq -r .payload | base64 --decode )

  if [[ "$PAYLOAD" != "$OTHER_PAYLOAD" ]]; then
    # Seems like we'll never get here
    echo "The two payloads are unexpectedly different!"
    exit 1
  fi

  # Data given to signature verification via verify-blob needs to be in DSSE protocol format
  # See https://github.com/secure-systems-lab/dsse/blob/master/protocol.md
  PAYLOAD_TYPE=$( echo $SIG_DATA | jq -r .payloadType )
  PAYLOAD="DSSEv1 ${#PAYLOAD_TYPE} ${PAYLOAD_TYPE} ${#PAYLOAD} $PAYLOAD"

fi

# Cosign needs files on disk to do a verify-blob afaict
SIG_FILE=$( mktemp )
PAYLOAD_FILE=$( mktemp )
function cleanup() {
  rm -f "$SIG_FILE" "$PAYLOAD_FILE"
}
trap cleanup EXIT
echo -n "$PAYLOAD" > $PAYLOAD_FILE
echo -n "$SIGNATURE" > $SIG_FILE

title Taskrun signature
say $SIGNATURE

pause

title Taskrun payload
[[ -z $QUIET ]] && echo "$RAW_PAYLOAD" | yq-pretty

pause

# Now use cosign to verify the signed payload
title Taskrun verification
show-then-run cosign verify-blob --key $SIG_KEY --signature $SIG_FILE $PAYLOAD_FILE

# For debugging...
title "To view taskrun"
say " env EDITOR=view kubectl edit $TASKRUN_NAME"

# Clean up
rm $SIG_FILE $PAYLOAD_FILE
