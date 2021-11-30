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
        name: openshift-client
      params:
        - name: SCRIPT 
          value: |
            #!/usr/bin/env bash 
            echo "Pre-Cleanup PVC Root Contents"
            ls -al
            rm -rf .*
            rm -rf ./*
            rm -rf *build*
            echo "Post-Cleanup PVC Root Contents"
            ls -al 
      workspaces:
        - name: manifest-dir 
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
yq -M e ".metadata.name=\"cleanup-$BUILD_TAG\"" tmp-cleanup.yaml   |  oc apply -f -

oc apply -f tmp-cleanup.yaml  
rm -rf tmp-cleanup.yaml tmp-pipeline.yaml 

 