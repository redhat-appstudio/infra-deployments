#!/bin/bash

SPI_DATA_PATH_PREFIX=${SPI_DATA_PATH_PREFIX:-spi}
SPI_POLICY_NAME=${SPI_DATA_PATH_PREFIX//\//-}
VAULT_KUBE_CONFIG=${VAULT_KUBE_CONFIG:-${KUBECONFIG:-$HOME/.kube/config}}
VAULT_NAMESPACE=${VAULT_NAMESPACE:-spi-vault}
VAULT_PODNAME=${VAULT_PODNAME:-vault-0}
ROOT_TOKEN_NAME=vault-root-token

function login() {
	ROOT_TOKEN=$(oc --kubeconfig=${VAULT_KUBE_CONFIG} get secret ${ROOT_TOKEN_NAME} -n ${VAULT_NAMESPACE} -o jsonpath="{.data.root_token}" | base64 --decode)
	vaultExec "vault login ${ROOT_TOKEN} > /dev/null"
}

function audit() {
	if ! vaultExec "vault audit list | grep -q file"; then
		echo "enabling audit log ..."
		vaultExec "vault audit enable file file_path=stdout"
	fi
}

function vaultExec() {
	COMMAND=${1}
	oc --kubeconfig=${VAULT_KUBE_CONFIG} exec ${VAULT_PODNAME} -n ${VAULT_NAMESPACE} -- sh -c "${COMMAND}" 2>/dev/null
}

function approleSet() {
	vaultExec "vault write auth/approle/role/${1} token_policies=${SPI_POLICY_NAME}"
	ROLE_ID=$(vaultExec "vault read auth/approle/role/${1}/role-id --format=json" | jq -r '.data.role_id')
	SECRET_ID=$(vaultExec "vault write -force auth/approle/role/${1}/secret-id --format=json" | jq -r '.data.secret_id')
	APP_ROLE_FILE=${2}
	echo "---" >>${APP_ROLE_FILE}
	oc --kubeconfig=${VAULT_KUBE_CONFIG} create secret generic vault-approle-${1} \
		--from-literal=role_id=${ROLE_ID} --from-literal=secret_id=${SECRET_ID} \
		--dry-run=client -o yaml >>${APP_ROLE_FILE}
}

function restart() {
	echo "restarting vault pod '${VAULT_PODNAME}' ..."
	oc --kubeconfig=${VAULT_KUBE_CONFIG} delete pod ${VAULT_PODNAME} -n ${VAULT_NAMESPACE} >/dev/null
}
