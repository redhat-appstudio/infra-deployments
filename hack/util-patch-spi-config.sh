#!/bin/bash

# Expects up to 3 parameters.
# 1. is the vault host (defaults to https://vault-spi-vault.apps.<cluster URL>)
# 2. is the base URL of SPI (defaults to https://spi-oauth-spi-system.apps.<cluster URL>)
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

patchConfig() {
  PATCH_FILE=$1
  echo 'Patching VAULTHOST and BASEURL for '"$PATCH_FILE"
  if [ -z ${2} ]; then
      APPS_BASE_URL=$(oc get ingress.config cluster -o jsonpath='{.spec.domain}')
      VAULT_HOST="https://vault-spi-vault.${APPS_BASE_URL}"
  else
      VAULT_HOST=${2}
  fi

  if [ -z ${3} ]; then
      if [ -z $APPS_BASE_URL ]; then
         APPS_BASE_URL=$(oc get ingress.config cluster -o jsonpath='{.spec.domain}')
      fi
      SPI_BASE_URL="https://spi-oauth-spi-system.${APPS_BASE_URL}"
  else
      SPI_BASE_URL=${3}
  fi


  TMP_FILE=$(mktemp)

  cat "$PATCH_FILE" | jq --arg VAULTHOST "${VAULT_HOST}" --arg BASEURL "${SPI_BASE_URL}" "${JQ_SCRIPT}" > "$TMP_FILE"
  cp "$TMP_FILE" "$PATCH_FILE"

  rm "$TMP_FILE"


}




ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

patchConfig "$ROOT/components/spi/overlays/development/config-patch.json" ${1} ${2}
patchConfig "$ROOT/components/remote-secret-controller/overlays/development/config-patch.json" ${1} ${2}