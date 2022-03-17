#!/bin/bash -ue

DEFAULT_URL=quay.io/sbaird/chains-demo
IMAGE_URL=${1:-$DEFAULT_URL}

# In reality we might know the digest already, but let's look it up
IMAGE_DIGEST=$( skopeo inspect --no-tags docker://$IMAGE_URL | jq -r .Digest )

# Use the digest to do a rekor search
UUIDS=$( rekor-cli search --sha "$IMAGE_DIGEST" 2>/dev/null )

# There might be more than one result so loop over them
for uuid in $UUIDS; do
  # Fetch the rekor data
  REKOR_DATA=$( rekor-cli get --uuid $uuid --format json )

  # Pull out some useful fields
  LOG_INDEX=$( echo "$REKOR_DATA" | jq -r .LogIndex )
  BODY=$( echo "$REKOR_DATA" | jq -r .Body )
  ATTESTATION=$( echo "$REKOR_DATA" | jq -r .Attestation )

  # Show useful output
  echo "# Image digest: $IMAGE_DIGEST"
  echo "# rekor-cli get --uuid $uuid"
  echo "# https://rekor.sigstore.dev/api/v1/log/entries?logIndex=$LOG_INDEX"

  echo "# Body:"
  echo "$BODY" | yq e -P -
  echo

  echo "# Attestation:"
  echo "$ATTESTATION" | base64 -d | yq e -P -
  echo
done
