#!/bin/bash
#
# Create a secret in your current namespace containing the chains
# signing secret's public key.
#
# This means it can be accessible for use in tasks without requiring
# permission to the full (protected) signing secret which includes
# the private key.
#
# It's assumed you have access to the tekton-chains signing-secret
# when you run this.
#
set -euo pipefail
cosign public-key --key k8s://tekton-chains/signing-secrets > cosign.pub
oc create secret generic cosign-public-key --from-file=cosign.pub --dry-run=client -o yaml | oc apply -f-
