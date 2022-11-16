#!/bin/bash -ue

DEFAULT_URL=quay.io/sbaird/chains-demo
IMAGE_URL=${1:-$DEFAULT_URL}
DEFAULT_REKOR_SERVER=https://rekor.sigstore.dev
REKOR_SERVER=${2:-$DEFAULT_REKOR_SERVER}

if [[ $IMAGE_URL =~ ^(sha256:)?[0-9a-fA-F]{64}$ ]]; then
  # Assume the param is already an image digest
  IMAGE_DIGEST="$IMAGE_URL"
else
  # Assume the param is an image url and look up the digest with skopeo
  IMAGE_DIGEST=$( skopeo inspect --no-tags docker://$IMAGE_URL | jq -r .Digest )
fi

# Use the digest to do a rekor search
UUIDS=$( rekor-cli search --sha "$IMAGE_DIGEST" --rekor_server $REKOR_SERVER 2>/dev/null )

# There might be more than one result so loop over them
for uuid in $UUIDS; do
  # Fetch the rekor data
  REKOR_DATA=$( rekor-cli get --uuid $uuid --rekor_server $REKOR_SERVER --format json )

  # Pull out some useful fields
  LOG_INDEX=$( echo "$REKOR_DATA" | jq -r .LogIndex )
  BODY=$( echo "$REKOR_DATA" | jq -r .Body )
  ATTESTATION=$( echo "$REKOR_DATA" | jq -r .Attestation )

  # Show useful output
  echo "# Image digest: $IMAGE_DIGEST"
  echo "# rekor-cli get --uuid $uuid --rekor_server $REKOR_SERVER"
  echo "# https://rekor.sigstore.dev/api/v1/log/entries?logIndex=$LOG_INDEX"

  echo "# Body:"
  echo "$BODY" | yq e -P -
  echo

  echo "# Attestation:"
  echo "$ATTESTATION" | base64 -d | yq e -P -
  echo
done
