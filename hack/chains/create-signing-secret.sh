#!/bin/bash
###
### NB: There is a hook that runs this after Gitops sync so you
### should not generally need to run this script manually.
### See components/build/tekton-chains/chains-secrets-config.yaml
###

# Show a message and exit gracefully if cosign isn't installed
! which cosign >/dev/null 2>&1 &&\
  echo "Please install cosign as per https://docs.sigstore.dev/cosign/installation/" && exit

# Once the key-pair has been set it's marked as immutable so it can't be updated.
# Try to handle that nicely. The object is expected to always exist so check the data.
SIG_KEY_DATA=$(kubectl get secret signing-secrets -n tekton-chains -o jsonpath='{.data}')
[[ -n $SIG_KEY_DATA ]] && echo "Signing secret exists." && exit

# To make this run conveniently without user input let's create a random password
RANDOM_PASS=$( head -c 12 /dev/urandom | base64 )

# Generates the key pair secret directly in the cluster. Also creates public key file cosign.pub
env COSIGN_PASSWORD=$RANDOM_PASS cosign generate-key-pair k8s://tekton-chains/signing-secrets

# Uncomment if you want to show the secret
#kubectl get secret signing-secrets -n tekton-chains -o yaml | yq e . -
