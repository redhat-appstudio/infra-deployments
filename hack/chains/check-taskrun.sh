#!/bin/bash

##
## Fixme: This works with the default chains config which is
## to use 'tekton' for storage and 'tekton' for the format. I can't
## make the verify-blob work with the storage configured to use 'oci'
## but I'll keep this script to help figure it out.
##

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Use a specific taskrun if provided, otherwise use the latest
TASKRUN_NAME=${1:-$( tkn taskrun describe --last -o name )}
TASKRUN_NAME=taskrun/$( echo $TASKRUN_NAME | sed 's#.*/##' )

QUIET_OPT=$2
SIG_KEY=$COSIGN_SIG_KEY

# Preserve sanity while hacking
set -ue

if [[ $QUIET_OPT == "--quiet" ]]; then
  ECHO=:
  QUIET=1
else
  ECHO=echo
  QUIET=
fi

# Helper for jsonpath
get-jsonpath() {
  kubectl get $TASKRUN_NAME -o jsonpath={.$1}
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

if [[ -z $SIG_KEY ]]; then
  # Requires that you're authenticated with an account that can access
  # the signing-secret, i.e. kubeadmin but not developer
  SIG_KEY=k8s://tekton-chains/signing-secrets

  # If you have the public key locally because you created it
  # (Presumably real public keys can be published somewhere in future)
  #SIG_KEY=$SCRIPTDIR/../../cosign.pub
fi

title() {
  $ECHO
  $ECHO "🔗 ---- $* ----"
}

# Show details about this taskrun
title Taskrun name
$ECHO $TASKRUN_NAME

title Signature
$ECHO $SIGNATURE

title Payload
#[[ -z $QUIET ]] && echo "$PAYLOAD" | jq
[[ -z $QUIET ]] && echo "$PAYLOAD" | yq e -P -

# Keep going if the verify fails
set +e

# Now use cosign to verify the signed payload
title Verification
cosign verify-blob --key $SIG_KEY --signature $SIG_FILE $PAYLOAD_FILE
COSIGN_EXIT_CODE=$?

# Clean up
rm $SIG_FILE $PAYLOAD_FILE

# Use the exit code from cosign
exit $COSIGN_EXIT_CODE
