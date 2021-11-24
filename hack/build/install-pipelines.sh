#!/bin/bash
PROJECT=$(oc config view --minify -o 'jsonpath={..namespace}')

PATCH_NS="$(printf '.items[].metadata.namespace="%q"' $PROJECT)" 

declare -a totrim=(
        "metadata.managedFields"  
        "metadata.creationTimestamp"  
        "metadata.generation"   
        "metadata.annotations"  
        "metadata.resourceVersion"  
        "metadata.uid"   
        "spec.volumeName"   
         ) 

echo installing default pipelines in $PROJECT

oc get pipelines -n build-templates -o yaml > pipelines.yaml 
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
 
yq eval 'del(.items[].metadata.labels["app.kubernetes.io/instance"])' pipelines.yaml -i 
yq eval 'del(.items[].metadata.labels["app.kubernetes.io/instance"])' pvc.yaml -i 
yq eval 'del(.items[].metadata.labels | select(length==0))' pipelines.yaml -i 

## Delete unneeded fields 

for i in "${totrim[@]}"
do 
    PATCH_FIELD="$(printf 'del(.items[].%q)' $i)" 
    #echo "$PATCH_FIELD"  
    yq eval "$PATCH_FIELD"  pipelines.yaml -i  
    yq eval "$PATCH_FIELD"  pvc.yaml -i 
done 

declare -a patchs=(
        "pipelines.yaml"   
        "pvc.yaml"    
         ) 
for i in "${patchs[@]}"
do 
echo $i
yq  e "$PATCH_NS" $i  -i  
oc apply -f  $i
#for debug comment out
rm $i 
done     

 