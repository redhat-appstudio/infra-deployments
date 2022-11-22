#!/bin/bash
#
# A wrapper for cosign to help verify a signed image
#

# Required
IMAGE_URL=$1

# If these are set we'll try to extract a secret and use it to login
REGISTRY_SECRET=$2
NAMESPACE=$3

# Set this to use a different sig key. See below for the defaults.
SIG_KEY=$COSIGN_SIG_KEY

# Set this to --verbose for lots of cosign debug output
VERBOSE=$COSIGN_VERBOSE

set -eu

if [[ -z $IMAGE_URL ]]; then
  echo "Image url is required."
  exit 1
fi

if [[ -z $SIG_KEY ]]; then
  # Requires that you're authenticated with an account that can access
  # the signing-secret in the cluster, i.e. kubeadmin but not developer
  SIG_KEY=k8s://tekton-chains/signing-secrets

  # If you have the public key locally because you created it
  # (Presumably real public keys will published somewhere in future)
  #SIG_KEY=$(git rev-parse --show-toplevel)/cosign.pub
fi

if [[ -n $REGISTRY_SECRET ]]; then
  #
  # Extract credentials for a docker login
  # Question: Could podman be used instead?
  #
  if [[ -n $NAMESPACE ]]; then
    REGISTRY_SECRET="$REGISTRY_SECRET -n $NAMESPACE"
  fi

  USER_PASS=$(
    kubectl get secret $REGISTRY_SECRET -o json |
      jq -r '.data[".dockerconfigjson"]' | base64 -d |
        jq -r '.auths["quay.io"].auth' | base64 -d )

  REGISTRY=$( echo $IMAGE_URL | cut -d/ -f1 )
  REG_USER=$( echo "$USER_PASS" | cut -d: -f1 )
  REG_PASS=$( echo "$USER_PASS" | cut -d: -f2 )

  docker login -u $REG_USER -p $REG_PASS $REGISTRY
fi

# Now verify
set -x
COSIGN_EXPERIMENTAL=1 cosign verify $VERBOSE --key $SIG_KEY $IMAGE_URL -o json | jq
