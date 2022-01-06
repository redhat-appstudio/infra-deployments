#!/bin/bash
PROJECT=$(oc config view --minify -o 'jsonpath={..namespace}')

PATCH_NS="$(printf '.metadata.namespace="%q"' $PROJECT)" 

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
   
yq  e "$PATCH_NS" pvc.yaml | oc apply -f -
rm pvc.yaml 