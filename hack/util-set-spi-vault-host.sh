#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
PATCH_FILE="$ROOT/components/spi/config-vaulthost-patch.json"
VAULT_HOST=${1:-vault.spi-vault.svc.cluster.local:8200}

TMP_FILE=$(mktemp)

cat $PATCH_FILE | jq --arg VAULT_HOST "${VAULT_HOST}" '.value = $VAULT_HOST' > "$TMP_FILE"
mv "$TMP_FILE" "$PATCH_FILE"
