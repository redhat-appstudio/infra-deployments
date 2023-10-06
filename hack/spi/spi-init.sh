#!/bin/bash

# !!! Note that this script should not be used for production purposes !!!
###########

source $(dirname "$0")/utils.sh

SPI_DATA_PATH_PREFIX=${SPI_DATA_PATH_PREFIX:-spi}
SPI_POLICY_NAME=${SPI_DATA_PATH_PREFIX//\//-}
SPI_APP_ROLE_FILE="$(realpath .tmp)/approle_secret.yaml"

function k8sAuth() {
	if ! vaultExec "vault auth list | grep -q kubernetes"; then
		echo "setup kubernetes authentication ..."
		vaultExec "vault auth enable kubernetes"
	fi
	vaultExec "vault write auth/kubernetes/role/spi-controller-manager \
        bound_service_account_names=spi-controller-manager \
        bound_service_account_namespaces=spi-system \
        policies=${SPI_POLICY_NAME}"
	vaultExec "vault write auth/kubernetes/role/spi-oauth \
          bound_service_account_names=spi-oauth-sa \
          bound_service_account_namespaces=spi-system \
          policies=${SPI_POLICY_NAME}"
	# shellcheck disable=SC2016
	vaultExec 'vault write auth/kubernetes/config \
        kubernetes_host=https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT'
}

function approleAuth() {
	if ! vaultExec "vault auth list | grep -q approle"; then
		echo "setup approle authentication ..."
		vaultExec "vault auth enable approle"
	fi

	mkdir -p .tmp

	echo '' > ${SPI_APP_ROLE_FILE}
	approleSet spi-operator ${SPI_APP_ROLE_FILE}
	approleSet spi-oauth ${SPI_APP_ROLE_FILE}

	echo "secret yaml with Vault credentials prepared"
}

function auth() {
	k8sAuth
	approleAuth
}

function approleAuthSPI() {
	login
	audit
	auth
}

if ! timeout 100s bash -c "while ! oc get applications.argoproj.io -n openshift-gitops -o name | grep -q spi-in-cluster-local; do printf '.'; sleep 5; done"; then
	printf "Application spi-in-cluster-local not found (timeout)\n"
	oc get apps -n openshift-gitops -o name
	exit 1
else
	if [ "$(oc get applications.argoproj.io spi-in-cluster-local -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; then
		echo "Initializing SPI"
		approleAuthSPI
		oc apply -f $SPI_APP_ROLE_FILE -n spi-system
		echo "SPI initialization was completed"
	else
		echo "SPI initialization was skipped"
	fi
fi
