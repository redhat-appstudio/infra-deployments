#!/bin/bash
PROJECT=$(oc config view --minify -o 'jsonpath={..namespace}')

cat > tmp-pipeline.yaml <<ENV-INLINE-PL-DECL
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: cleanup  
spec: 
  tasks: 
    - name: cleanup 
      taskRef:
        kind: ClusterTask
        name: cleanup-build-directories 
      workspaces:
        - name: source 
          workspace: workspace
  workspaces:
    - name: workspace
ENV-INLINE-PL-DECL

cat > tmp-cleanup.yaml <<ENV-INLINE-PR-DECL
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata: 
  name: cleanup
spec: 
  pipelineRef:
    name: cleanup
  workspaces:
    - name: workspace
      persistentVolumeClaim:
        claimName: app-studio-default-workspace
      subPath: "." 
ENV-INLINE-PR-DECL

oc apply -f tmp-pipeline.yaml   
BUILD_TAG=$(date +"%Y-%m-%d-%H%M%S")
PRNAME=cleanup-$BUILD_TAG
yq -M e ".metadata.name=\"$PRNAME\"" tmp-cleanup.yaml   |  oc apply -f -
  
rm -rf tmp-cleanup.yaml tmp-pipeline.yaml 
tkn pipelinerun logs $PRNAME -f
tkn pipelinerun delete $PRNAME -f
tkn pipeline delete cleanup -f


 