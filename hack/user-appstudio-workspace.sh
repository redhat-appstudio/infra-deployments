#!/bin/bash

set -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [[ -z ${APPSTUDIO_SP_WORKSPACE} ]]
then
  ROOT_WORKSPACE=${ROOT_WORKSPACE:-"root"}
  APPSTUDIO_WORKSPACE=${APPSTUDIO_WORKSPACE:-"redhat-appstudio"}
  if [ "${ROOT_WORKSPACE}" == "~" ]; then
    CURRENT_WS=$(KUBECONFIG=${KCP_KUBECONFIG} kubectl ws . | cut -f2 -d'"')
    APPSTUDIO_SP_WORKSPACE=${CURRENT_WS}:${APPSTUDIO_WORKSPACE}
  else
    APPSTUDIO_SP_WORKSPACE=${ROOT_WORKSPACE}:${APPSTUDIO_WORKSPACE}
  fi
fi

USER_APPSTUDIO_WORKSPACE=${USER_APPSTUDIO_WORKSPACE:-"appstudio"}
echo "Accessing the home workspace:"
kubectl ws
echo "Creating & accessing AppStudio workspace '${USER_APPSTUDIO_WORKSPACE}':"
kubectl ws create ${USER_APPSTUDIO_WORKSPACE}  --ignore-existing --type root:universal
kubectl ws ${USER_APPSTUDIO_WORKSPACE}

for API_BINDING in ${ROOT}/apibindings/appstudio/*
do
  APPSTUDIO_SP_WORKSPACE=${APPSTUDIO_SP_WORKSPACE} envsubst < ${API_BINDING} | kubectl apply -f -
done
