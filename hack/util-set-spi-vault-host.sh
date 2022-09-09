#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
PATCH_FILE="$ROOT/components/spi/config-patch.json"

if [ -z ${1} ]; then
    CLUSTER_URL_HOST=$(kubectl get ingresses.config.openshift.io cluster --template={{.spec.domain}})
    VAULT_HOST="https://spi-vault-spi-system.${CLUSTER_URL_HOST}"
else
    VAULT_HOST=${1}
fi

TMP_FILE=$(mktemp)

cat $PATCH_FILE | jq --arg VAULT_HOST "${VAULT_HOST}" '.[0].value = $VAULT_HOST' > "$TMP_FILE"
mv "$TMP_FILE" "$PATCH_FILE"

# because we can't be sure that target testing cluster has valid signed cert, we allow insecure tls connection to Vault
yq e -i '.patches += {"target": {"kind": "Deployment","name": "controller-manager|oauth-service"}, "path": "insecuretls-patch.json"}' $ROOT/components/spi/kustomization.yaml
