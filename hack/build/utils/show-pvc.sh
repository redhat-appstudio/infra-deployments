#!/bin/bash
PROJECT=$(oc config view --minify -o 'jsonpath={..namespace}')
  
read -r -d '' BASE_TASKRUN <<'BASE_TASKRUN' 
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata: 
  name: REPLACE
spec: 
  taskRef:
    name: appstudio-utils
    kind: ClusterTask  
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
        claimName: app-studio-default-workspace
      subPath: "."
WORKSPACE

function run_task() { 
  BUILD_TAG=$(date +"%Y-%m-%d-%H%M%S")
  PRNAME=showdir-$BUILD_TAG 
  COMBINED=$(printf "%s\n    %s\n" "$1" "$2")

echo "Show Workspace:"
echo "$2"
echo "---"

  echo "$COMBINED" | \
  yq -M e ".metadata.name=\"$PRNAME\"" - |  oc apply -f - 
  tkn taskrun logs $PRNAME -f
  tkn taskrun delete $PRNAME -f 
}
 
run_task "$BASE_TASKRUN" "$WORKSPACE" 