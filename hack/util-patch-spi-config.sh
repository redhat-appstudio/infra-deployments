#!/bin/bash

# Expects up to 3 parameters.
# 1. is the vault host (defaults to https://spi-vault-spi-system.apps.<cluster URL>)
# 2. is the base URL of SPI (defaults to https://spi-oauth-route-spi-system.apps.<cluster URL>)
# 3. is either true or false and defaults to true. When true, Vault is configured to accepts TLS connections with untrusted certificates.

JQ_SCRIPT=$(cat << "EOF"
map(
    if (.op == "replace" and .path == "/data/VAULTHOST") then
        {"op": .op, "path": .path, "value": $VAULTHOST }
    elif (.op == "replace" and .path == "/data/BASEURL") then
        {"op": .op, "path": .path, "value": $BASEURL }
    else
        .
    end
)
EOF
)

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
PATCH_FILE="$ROOT/components/spi/config-patch.json"

if [ -z ${1} ]; then
    APPS_BASE_URL=$(oc get ingress.config cluster -o jsonpath='{.spec.domain}')
    VAULT_HOST="https://spi-vault-spi-system.${APPS_BASE_URL}"
else
    VAULT_HOST=${1}
fi

if [ -z ${2} ]; then
    if [ -z $APPS_BASE_URL ]; then
       APPS_BASE_URL=$(oc get ingress.config cluster -o jsonpath='{.spec.domain}')
    fi
    SPI_BASE_URL="https://spi-oauth-route-spi-system.${APPS_BASE_URL}"
else
    SPI_BASE_URL=${2}
fi

if [ -z ${3} ]; then
    VAULT_INSECURE_TLS="true"
else
    VAULT_INSECURE_TLS=${3}
fi

TMP_FILE=$(mktemp)

cat $PATCH_FILE | jq --arg VAULTHOST "${VAULT_HOST}" --arg BASEURL "${SPI_BASE_URL}" "${JQ_SCRIPT}" > "$TMP_FILE"
cp "$TMP_FILE" "$PATCH_FILE"

if [ "$VAULT_INSECURE_TLS" == "true" ]; then
    cat "$PATCH_FILE" | jq '. += [{"op": "add", "path": "/data/VAULTINSECURETLS", "value": "true"}]' > "$TMP_FILE"
    cp "$TMP_FILE" "$PATCH_FILE"
fi

rm "$TMP_FILE"
