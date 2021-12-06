#!/bin/bash
PROJECT=$(oc config view --minify -o 'jsonpath={..namespace}')

PATCH_NS="$(printf '.metadata.namespace="%q"' $PROJECT)" 


PIPELINE_NAME=$1 
if [ -z "$PIPELINE_NAME" ]
then
      echo Missing parameter Pipeline Name
      exit -1 
fi

oc get pipelines $PIPELINE_NAME -n build-templates -o yaml 2> err  > pipelines.yaml 
ERR=$? 
if (( $ERR != 0 )); then
  echo No Pipeline named $PIPELINE_NAME found in build-templates
  oc get pipelines $PIPELINE_NAME -o yaml 2> err  > pipelines.yaml 
  ERR=$? 
  if (( $ERR != 0 )); then
    echo No Pipeline named $PIPELINE_NAME found in current project, exiting
    rm -f pipelines.yaml err 
    exit -1
  fi 
  rm -f pipelines.yaml err  
  echo "Warning using Pipeline $PIPELINE_NAME already found in user project"
  echo "This pipeline was not installed via the gitops method in App Studio Build"
  echo "Ensure your pipeline is in a git repo to ensure you won't lose it."
  exit 0
fi 

RV=$(yq e '.metadata.resourceVersion' pipelines.yaml )  
OLD_RV=$(oc get pipelines $PIPELINE_NAME -o yaml  2> err  | yq eval '.metadata.annotations.build-templates/revision' -)
rm err 
if [ "$RV" == "$OLD_RV" ]
then
      echo "Pipeline $PIPELINE_NAME is installed"
      rm -f pipelines.yaml 
      exit 0
fi 
echo "Installing $PIPELINE_NAME in $PROJECT"

#oc get pvc -n build-templates -o yaml > pvc.yaml
# until the read access to pvc in build-templates is updated, include required pvc
cat > pvc.yaml <<ENV-INLINE_PVC-DECL
apiVersion: v1
items:
  - apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      finalizers:
        - kubernetes.io/pvc-protection 
      name: app-studio-default-workspace
      namespace: $PROJECT 
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
      volumeMode: Filesystem
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
ENV-INLINE_PVC-DECL
 
yq eval 'del(.metadata.labels["app.kubernetes.io/instance"])' pipelines.yaml -i 
yq eval 'del(.metadata.labels["app.kubernetes.io/instance"])' pvc.yaml -i 
yq eval 'del(.metadata.labels | select(length==0))' pipelines.yaml -i 

## Delete unneeded fields 
declare -a totrim=(
        "metadata.managedFields"  
        "metadata.creationTimestamp"  
        "metadata.generation"   
        "metadata.annotations"  
        "metadata.resourceVersion"  
        "metadata.uid"   
        "spec.volumeName"   
         ) 
for i in "${totrim[@]}"
do 
    PATCH_FIELD="$(printf 'del(.%q)' $i)" 
    #echo "$PATCH_FIELD"  
    yq eval "$PATCH_FIELD"  pipelines.yaml -i  
    yq eval "$PATCH_FIELD"  pvc.yaml -i 
done
REV="$(printf '.metadata.annotations={"build-templates/revision": "%q"}' $RV)" 
yq eval "$REV" pipelines.yaml -i


declare -a patchs=(
        "pipelines.yaml"   
        "pvc.yaml"    
         ) 
for i in "${patchs[@]}"
do 
#echo $i
yq  e "$PATCH_NS" $i  -i  
oc apply -f  $i
#for debug comment out
rm $i 
done     

 