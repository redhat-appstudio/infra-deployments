#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
PATCH_FILE="$ROOT/components/spi/config-vaulthost-patch.json"

if [ -z ${1} ]; then
    CLUSTER_URL_HOST=$(oc whoami --show-console|sed 's|https://console-openshift-console.apps.||')
    VAULT_HOST="https://vault-spi-vault.apps.${CLUSTER_URL_HOST}"
else
    VAULT_HOST=${1}
fi

TMP_FILE=$(mktemp)

cat $PATCH_FILE | jq --arg VAULT_HOST "${VAULT_HOST}" '.value = $VAULT_HOST' > "$TMP_FILE"
mv "$TMP_FILE" "$PATCH_FILE"
