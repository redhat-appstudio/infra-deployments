#!/bin/bash

# !!! Note that this script should not be used for production purposes !!!

source $(dirname "$0")/utils.sh

set -e

mkdir -p $HOME/.tmp
touch $HOME/.tmp/keys-file

VAULT_KUBE_CONFIG=${VAULT_KUBE_CONFIG:-${KUBECONFIG:-$HOME/.kube/config}}
VAULT_NAMESPACE=${VAULT_NAMESPACE:-spi-vault}
SECRET_NAME=spi-vault-keys
VAULT_PODNAME=${VAULT_PODNAME:-vault-0}
KEYS_FILE=${KEYS_FILE:-"$(echo $HOME/.tmp)/keys-file"}
ROOT_TOKEN=""
ROOT_TOKEN_NAME=vault-root-token

SPI_DATA_PATH_PREFIX=${SPI_DATA_PATH_PREFIX:-spi}
SPI_POLICY_NAME=${SPI_DATA_PATH_PREFIX//\//-}

function init() {
	INIT_STATE=$(isInitialized)
	if [ "$INIT_STATE" == "false" ]; then
		echo '' >${KEYS_FILE}
		vaultExec "vault operator init" >"${KEYS_FILE}"
		echo "Keys written at ${KEYS_FILE}"
	elif [ "$INIT_STATE" == "true" ]; then
		echo "Vault already initialized"
	else
		echo "$INIT_STATE"
		exit 1
	fi
}

function isInitialized() {
	STATUS=$(vaultExec "vault status -format=yaml 2>&1")
	INITIALIZED=$(echo "$STATUS" | grep "initialized")
	if [ -z "${INITIALIZED}" ]; then
		echo "failed to obtain initialization status; vault may be in an irrecoverable error state"
		echo "vault status output: ${STATUS}"
	fi
	echo "${INITIALIZED}" | awk '{split($0,a,": "); print a[2]}'
}

function isSealed() {
	SEALED=$(vaultExec "vault status -format=yaml | grep sealed")
	echo "${SEALED}" | awk '{split($0,a,": "); print a[2]}'
}

function secret() {
	if [ ! -s "${KEYS_FILE}" ]; then
		return
	fi

	if oc --kubeconfig=${VAULT_KUBE_CONFIG} get secret ${SECRET_NAME} -n ${VAULT_NAMESPACE} 2>/dev/null; then
		echo "Secret ${SECRET_NAME} already exists. Deleting ..."
		oc --kubeconfig=${VAULT_KUBE_CONFIG} delete secret ${SECRET_NAME} -n ${VAULT_NAMESPACE}
	fi

	COMMAND="oc --kubeconfig=${VAULT_KUBE_CONFIG} create secret generic ${SECRET_NAME} -n ${VAULT_NAMESPACE}"
	KEYI=1
	# shellcheck disable=SC2013
	for KEY in $(grep "Unseal Key" "${KEYS_FILE}" | awk '{split($0,a,": "); print a[2]}'); do
		COMMAND="${COMMAND} --from-literal=key${KEYI}=${KEY}"
		((KEYI++))
	done

	${COMMAND}
}

function unseal() {
	KEYI=1
	until [ "$(isSealed)" == "false" ]; do
		echo "unsealing ..."
		KEY=$(oc --kubeconfig=${VAULT_KUBE_CONFIG} get secret ${SECRET_NAME} -n ${VAULT_NAMESPACE} --template="{{.data.key${KEYI}}}" | base64 --decode)
		if [ -z "${KEY}" ]; then
			echo "failed to unseal"
			exit 1
		fi
		vaultExec "vault operator unseal ${KEY}"
		((KEYI++))
	done
	echo "unsealed"
}

function ensureRootToken() {
	if [ -s "${KEYS_FILE}" ]; then
		ROOT_TOKEN=$(grep "Root Token" "${KEYS_FILE}" | awk '{split($0,a,": "); print a[2]}')
	else
		generateRootToken
	fi

	# save ROOT_TOKEN to be used in the `spi-init` and `remote-secret-init` scripts
	oc --kubeconfig=${VAULT_KUBE_CONFIG} create secret generic ${ROOT_TOKEN_NAME} \
		--from-literal=root_token=${ROOT_TOKEN} -n ${VAULT_NAMESPACE}
}

function generateRootToken() {
	echo "generating root token ..."

	vaultExec "vault operator generate-root -cancel" >/dev/null
	INIT=$(vaultExec "vault operator generate-root -init -format=yaml")
	NONCE=$(echo "${INIT}" | grep "nonce:" | awk '{split($0,a,": "); print a[2]}')
	OTP=$(echo "${INIT}" | grep "otp:" | awk '{split($0,a,": "); print a[2]}')

	KEYI=1
	COMPLETE="false"
	until [ "${COMPLETE}" == "true" ]; do
		KEY=$(oc --kubeconfig=${VAULT_KUBE_CONFIG} get secret ${SECRET_NAME} -n ${VAULT_NAMESPACE} --template="{{.data.key${KEYI}}}" | base64 --decode)
		if [ -z "${KEY}" ]; then
			echo "failed to generate token"
			exit 1
		fi
		GENERATE_OUTPUT=$(vaultExec "echo ${KEY} | vault operator generate-root -nonce=${NONCE} -format=yaml -")
		COMPLETE=$(echo "${GENERATE_OUTPUT}" | grep "complete:" | awk '{split($0,a,": "); print a[2]}')
		if [ "${COMPLETE}" == "true" ]; then
			ENCODED_TOKEN=$(echo "${GENERATE_OUTPUT}" | grep "encoded_token" | awk '{split($0,a,": "); print a[2]}')
			ROOT_TOKEN=$(vaultExec "vault operator generate-root \
        -decode=${ENCODED_TOKEN} \
        -otp=${OTP} -format=yaml" |
				awk '{split($0,a,": "); print a[2]}')
		fi
		((KEYI++))
	done
}

function applyPolicy() {
	POLICY_FILE=/tmp/spi_policy.hcl
	vaultExec "echo 'path \"${SPI_DATA_PATH_PREFIX}/*\" { capabilities = [\"read\", \"create\", \"list\", \"delete\", \"update\"] }' > ${POLICY_FILE}"
	vaultExec "vault policy write ${SPI_POLICY_NAME} ${POLICY_FILE}"
	vaultExec "rm ${POLICY_FILE}"
}

function spiSecretEngine() {
	if ! vaultExec "vault secrets list | grep -q ${SPI_DATA_PATH_PREFIX}"; then
		echo "creating SPI secret engine ..."
		vaultExec "vault secrets enable -path=${SPI_DATA_PATH_PREFIX} kv-v2"
	fi
}

function initVault() {
	until [ "$(oc --kubeconfig=${VAULT_KUBE_CONFIG} get pod ${VAULT_PODNAME} -n ${VAULT_NAMESPACE} -o jsonpath='{.status.phase}')" == "Running" ]; do
		sleep 5
		echo "Waiting for Vault pod to be running."
	done

	sleep 5

	init
	secret
	unseal
	ensureRootToken
	login
	audit
	spiSecretEngine
	applyPolicy
}

if ! timeout 100s bash -c "while ! oc get applications.argoproj.io -n openshift-gitops -o name | grep -q spi-vault-in-cluster-local; do printf '.'; sleep 5; done"; then
	printf "Application spi-vault-in-cluster-local not found (timeout)\n"
	oc get apps -n openshift-gitops -o name
	exit 1
else
	if [ "$(oc get applications.argoproj.io spi-vault-in-cluster-local -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; then
		echo "Initializing vault"
		initVault
		echo "Vault initialization was completed"
	else
		echo "Vault initialization was skipped"
	fi
fi
