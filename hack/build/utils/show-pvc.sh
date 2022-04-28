#!/bin/bash

usage () {
	echo "Usage:   $0 [--n NAMESPACE --pvc-claim appstudio]."
	echo "Example: ./show-pvc.sh --n test --pvc-claim appstudio "
}

while [[ "$#" -gt 0 ]]; do
  case $1 in
    '--pvc-claim') PVC_CLAIM="$2"; shift 1;;
    '--n') PROJECT="$2"; shift 1;;
	  '--help'|'-h') usage; exit;;
  esac
  shift 1
done

if [ -z "${PVC_CLAIM}" ]; then
  PVC_CLAIM="app-studio-default-workspace"
fi

if [ -z "${PROJECT}" ]; then
  PROJECT=$(oc config view --minify -o 'jsonpath={..namespace}')
fi

oc get pvc "${PVC_CLAIM}" -n "${PROJECT}" || exit 1

read -r -d '' BASE_TASKRUN <<'BASE_TASKRUN' 
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata: 
  name: REPLACE
spec: 
  taskRef:
    name: utils-task
    bundle: quay.io/redhat-appstudio/appstudio-tasks:utils-task-v0.1.5 
  params:
    - name: SCRIPT  
      value: |
        #!/usr/bin/env bash
        echo  "Show Directory"   
        echo  
        echo "ls /workspace"
        ls /workspace 
        echo  
        echo "du -a /workspace"
        du -a /workspace
        echo
  workspaces:
BASE_TASKRUN

read -r -d '' WORKSPACE <<'WORKSPACE' 
    - name: source
      persistentVolumeClaim:
        claimName: "${PVC_CLAIM}"
      subPath: "."
WORKSPACE

function run_task() { 
  BUILD_TAG=$(date +"%Y-%m-%d-%H%M%S")
  PRNAME=showdir-$BUILD_TAG
  SOURCE=$(echo "${2}" | sed -r "s/\\$\\{PVC_CLAIM\\}/${PVC_CLAIM}/g")
  COMBINED=$(printf "%s\n    %s\n" "$1" "${SOURCE}")

  echo "Show Workspace:"
  echo "$2"
  echo "---"

  echo "$COMBINED" | \
  yq -M e ".metadata.name=\"$PRNAME\"" - |  oc apply -n "${PROJECT}" -f -
  tkn taskrun logs $PRNAME -n "${PROJECT}" -f
  tkn taskrun delete $PRNAME -n "${PROJECT}" -f
}

run_task "$BASE_TASKRUN" "$WORKSPACE"
