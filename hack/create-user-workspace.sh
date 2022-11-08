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
HOME_WORKSPACE=${HOME_WORKSPACE:-"~"}
APPSTUDIO_WORKSPACE=${APPSTUDIO_WORKSPACE:-"redhat-appstudio"}
HACBS_WORKSPACE=${HACBS_WORKSPACE:-"redhat-hacbs"}
PIPELINE_SERVICE_WORKSPACE=${PIPELINE_SERVICE_WORKSPACE:-"redhat-pipeline-service-compute"}

if [[ -n ${KCP_KUBECONFIG} ]]
then
  export KUBECONFIG=${KCP_KUBECONFIG}
fi

if [ "${ROOT_WORKSPACE}" == "~" ]; then
  ROOT_WORKSPACE=$(kubectl ws '~' --short)
fi

echo "Accessing the home workspace:"
kubectl ws $HOME_WORKSPACE

APPSTUDIO_SP_WORKSPACE=${APPSTUDIO_SP_WORKSPACE:-${ROOT_WORKSPACE}:${APPSTUDIO_WORKSPACE}}
HACBS_SP_WORKSPACE=${HACBS_SP_WORKSPACE:-${ROOT_WORKSPACE}:${HACBS_WORKSPACE}}
PIPELINE_SERVICE_SP_WORKSPACE=${PIPELINE_SERVICE_SP_WORKSPACE:-${ROOT_WORKSPACE}:${PIPELINE_SERVICE_WORKSPACE}}

CWT="${APPSTUDIO_SP_WORKSPACE}:appstudio"
if [[ ${SERVICE_NAME} == "hacbs" ]]
then
  CWT="${HACBS_SP_WORKSPACE}:hacbs"
fi

USER_WORKSPACE=${USER_WORKSPACE:-"${SERVICE_NAME}"}
echo "Creating & accessing ${SERVICE_NAME} workspace '${USER_WORKSPACE}':"
kubectl ws create ${USER_WORKSPACE}  --ignore-existing --type ${CWT} --enter

kubectl kustomize ${ROOT}/apibindings/${SERVICE_NAME}/ | sed "s|\${APPSTUDIO_SP_WORKSPACE}|${APPSTUDIO_SP_WORKSPACE}|g;s|\${HACBS_SP_WORKSPACE}|${HACBS_SP_WORKSPACE}|g;s|\${PIPELINE_SERVICE_SP_WORKSPACE}|${PIPELINE_SERVICE_SP_WORKSPACE}|g" | \
  kubectl apply -f -

echo
echo "Patching APIBindings to accept all permission claims:"
API_BINDINGS=$(kubectl get apibindings.apis.kcp.dev -l provided-by=infra-deployments -o name)
for API_BINDING in ${API_BINDINGS}
do
  EXPORT_PERMISSION_CLAIMS=$(kubectl get ${API_BINDING} -o jsonpath='{.status.exportPermissionClaims}')
  ACCEPTED_CLAIMS=
  for EXPORT_CLAIM in $(echo "${EXPORT_PERMISSION_CLAIMS}" | jq -c '.[]')
  do
    ACCEPTED_CLAIMS=${ACCEPTED_CLAIMS}$(echo "${EXPORT_CLAIM}" | jq '. += {"state": "Accepted"}' | jq -c)","
  done
  kubectl patch ${API_BINDING} --type='json' -p="[{'op': 'replace', 'path': '/spec/permissionClaims', 'value': [${ACCEPTED_CLAIMS}]}]"
done

echo
echo "The ${USER_WORKSPACE} user workspace is created: $(kubectl ws . --short)"
