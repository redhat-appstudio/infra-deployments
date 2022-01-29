#!/bin/bash

TASKRUN_NAME=$1

# Preserve sanity while hacking
set -ue

if [[ -z $TASKRUN_NAME ]]; then
  # Use the most recently created taskrun
  # (Fixme: Would be better to exclude running tasks)
  TASKRUN_NAME=$(
    kubectl get taskrun -o name --sort-by=.metadata.creationTimestamp |
      tail -1 | cut -d/ -f2 )
fi

# Helper for jsonpath
get-jsonpath() {
  kubectl get taskrun/$TASKRUN_NAME -o jsonpath={.$1}
}

# Helper for reading chains values
get-chainsval() {
  get-jsonpath metadata.annotations.chains\\.tekton\\.dev/$1
}

# Fetch signature and payload
TASKRUN_UID=$( get-jsonpath metadata.uid )
SIGNATURE=$( get-chainsval signature-taskrun-$TASKRUN_UID )
PAYLOAD=$( get-chainsval payload-taskrun-$TASKRUN_UID | base64 --decode )

# Cosign wants files on disk afaict
SIG_FILE=$( mktemp )
PAYLOAD_FILE=$( mktemp )
echo -n "$PAYLOAD" > $PAYLOAD_FILE
echo -n "$SIGNATURE" > $SIG_FILE

# Requires that you're authenticated with an account that can access
# the signing-secret, i.e. kubeadmin but not developer
SIG_KEY=k8s://tekton-chains/signing-secrets

# If you have the public key locally because you created it
# (Presumably real public keys can be published somewhere in future)
#SIG_KEY=./cosign.pub

title() {
  echo
  echo "🔗 ---- $* ----"
}

# Show details about this taskrun
title Taskrun name
echo $TASKRUN_NAME

title Signature
echo $SIGNATURE

title Payload
#echo "$PAYLOAD" | jq
echo "$PAYLOAD" | yq e -P -

# Keep going if the verify fails
set +e

# Now use cosign to verify the signed payload
title Verification
cosign verify-blob --key $SIG_KEY --signature $SIG_FILE $PAYLOAD_FILE

# Clean up
rm $SIG_FILE $PAYLOAD_FILE
