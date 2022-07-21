#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
PATCH_FILE="$ROOT/components/spi/oauth-service-deployment-patch.json"
API_SERVER=$1

TMP_FILE=$(mktemp)

cat $PATCH_FILE | jq --arg API_SERVER "${API_SERVER}" '.[0].value.value = $API_SERVER' > "$TMP_FILE"

mv "$TMP_FILE" "$PATCH_FILE"
