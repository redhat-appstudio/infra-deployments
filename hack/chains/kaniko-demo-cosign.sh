#!/bin/bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source $SCRIPTDIR/_helpers.sh
set -ue

# Use a specific taskrun if provided, otherwise use the latest
TASKRUN_NAME=${1:-$( tkn taskrun describe --last -o name )}
TASKRUN_NAME=taskrun/$( echo $TASKRUN_NAME | sed 's#.*/##' )

# Let's not hard code the image url or the registry
IMAGE_URL=$( oc get $TASKRUN_NAME -o json | jq -r '.status.taskResults[1].value' )
IMAGE_REGISTRY=$( echo $IMAGE_URL | cut -d/ -f1 )
#IMAGE_REGISTRY=$( oc registry info )

SIG_KEY="k8s://tekton-chains/signing-secrets"

# Make sure we have a docker credential since cosign will need it
# (Todo: Probably shouldn't assume kubeadmin user here)
oc whoami -t | docker login -u kubeadmin --password-stdin $IMAGE_REGISTRY

title "Inspecting $TASKRUN_NAME metadata"
echo "(Beware long lines are truncated)"
# Just want to show the chains related fields
# Truncating to avoid a whole screen full of base64 encoded data
oc get $TASKRUN_NAME -o yaml | yq e .metadata -C - | cut -c -150
pause

title "Cosign verify the image"
# Save the output data to a file so we can look at it later
show-then-run "cosign verify --key $SIG_KEY $IMAGE_URL --output-file /tmp/verify.out"
pause

# The output files are json, but let's show as yaml for readability

title "Inspect the cosign verify output"
yq e . -P /tmp/verify.out
pause

title "Cosign verify the image's attestation"
show-then-run "cosign verify-attestation --key $SIG_KEY $IMAGE_URL --output-file /tmp/verify-att.out"
pause

# If you build the same build more than once there will be multiple
# attestations and hence multiple lines in this file, which makes it
# invalid json. For the sake of the demo we'll be lazy and ignore
# all but the last line.
#
title "Inspect the cosign verify attestation output"
tail -1 /tmp/verify-att.out | yq e . -P -
pause

title "Inspect the payload from that attestation output"
tail -1 /tmp/verify-att.out | yq e .payload - | base64 -d | yq e . -P -
