#!/bin/bash

set -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [[ -z ${1} ]]
then
  echo "You have to provide the name of the service you want to initiate in the user workspace:"
  echo "     create-user-workspace.sh <appstudio|hacbs>"
  exit 1
fi

SERVICE_NAME=${1}

ROOT_WORKSPACE=${ROOT_WORKSPACE:-"root"}
APPSTUDIO_WORKSPACE=${APPSTUDIO_WORKSPACE:-"redhat-appstudio"}
HACBS_WORKSPACE=${HACBS_WORKSPACE:-"redhat-hacbs"}

if [[ -n ${KCP_KUBECONFIG} ]]
then
  export KUBECONFIG=${KCP_KUBECONFIG}
fi

echo "Accessing the home workspace:"
kubectl ws '~'

if [ "${ROOT_WORKSPACE}" == "~" ]; then
  ROOT_WORKSPACE=$(kubectl ws . | cut -f2 -d'"')
fi

APPSTUDIO_SP_WORKSPACE=${APPSTUDIO_SP_WORKSPACE:-${ROOT_WORKSPACE}:${APPSTUDIO_WORKSPACE}}
HACBS_SP_WORKSPACE=${HACBS_SP_WORKSPACE:-${ROOT_WORKSPACE}:${HACBS_WORKSPACE}}

USER_APPSTUDIO_WORKSPACE=${USER_APPSTUDIO_WORKSPACE:-"${SERVICE_NAME}"}
echo "Creating & accessing AppStudio workspace '${USER_APPSTUDIO_WORKSPACE}':"
kubectl ws create ${USER_APPSTUDIO_WORKSPACE}  --ignore-existing --type root:universal --enter

kubectl kustomize ${ROOT}/apibindings/${SERVICE_NAME}/ | sed "s|\${APPSTUDIO_SP_WORKSPACE}|${APPSTUDIO_SP_WORKSPACE}|g;s|\${HACBS_SP_WORKSPACE}|${HACBS_SP_WORKSPACE}|g" | \
  kubectl apply -f -

echo
echo "The ${SERVICE_NAME} user workspace is created: $(kubectl ws . | cut -f2 -d'"')"
