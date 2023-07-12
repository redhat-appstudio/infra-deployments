#!/bin/bash

# !!! Note that this script should not be used for production purposes !!!

source $(dirname "$0")/utils.sh

REMOTE_SECRET_APP_ROLE_FILE="$(realpath .tmp)/approle_remote_secret.yaml"

function auth() {
	if ! vaultExec "vault auth list | grep -q approle"; then
		echo "setup approle authentication ..."
		vaultExec "vault auth enable approle"
	fi

	mkdir -p .tmp

	echo '' > ${REMOTE_SECRET_APP_ROLE_FILE}
	approleSet remote-secret-operator ${REMOTE_SECRET_APP_ROLE_FILE}

	echo "secret yaml with Vault credentials prepared"
}

function approleAuthRemoteSecret() {
	login
	audit
	auth
}

if ! timeout 300s bash -c "while ! oc get applications.argoproj.io -n openshift-gitops -o name | grep -q remote-secret-controller-in-cluster-local; do printf '.'; sleep 5; done"; then
	printf "Application remote-secret-controller-in-cluster-local not found (timeout)\n"
	oc get apps -n openshift-gitops -o name
	exit 1
else
	if [ "$(oc get applications.argoproj.io remote-secret-controller-in-cluster-local -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; then
		echo "Initializing remote secret controller"
		approleAuthRemoteSecret
		restart
		oc apply -f $REMOTE_SECRET_APP_ROLE_FILE -n remotesecret
		echo "Remote secret controller initialization was completed"
	else
		echo "Remote secret controller initialization was skipped"
	fi
fi
