#!/bin/bash
# PipelineRun builds images for application.
# This script removes images from Openshift internal image registry
# to have ability re-triger PipelineRuns without skip build image tasks.  

usage () {
	echo "Usage:   $0 [--only-images ONLY_IMAGES --n NAMESPACE]"
	echo "Example: ./cleanup-images-and-pipelineruns.sh --only-images true --n test"
}

while [[ "$#" -gt 0 ]]; do
  case $1 in
    '--only-images') ONLY_IMAGES="$2"; shift 1;;
    '--n') NAMESPACE="$2"; shift 1;;
	'--help'|'-h') usage; exit;;
  esac
  shift 1
done

deleteImageStreams() {
    echo "[INFO] Delete image streams..."
    ISs=($(oc get is --no-headers=true -o jsonpath="{.items[*]['metadata.name']}" -n "${NAMESPACE}"))
    for is in "${ISs[@]}"; do
        # echo "[INFO] Found image stream ${is}. Let's delete it."
        oc delete is "${is}" -n "${NAMESPACE}"
    done
}

deletePipelineRuns() {
    echo "[INFO] Delete pipelineruns..."
    pipelineruns=($(oc get pipelineruns --no-headers=true -o jsonpath="{.items[*]['metadata.name']}" -n "${NAMESPACE}"))
    for pipelinerun in "${pipelineruns[@]}"; do
        oc delete pipelinerun "${pipelinerun}" -n "${NAMESPACE}"
    done
}

deleteImageStreams
if [ "${ONLY_IMAGES}" != "true" ]; then
    deletePipelineRuns
fi
