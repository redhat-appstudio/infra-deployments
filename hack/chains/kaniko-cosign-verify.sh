#!/bin/bash

source $(dirname $0)/_helpers.sh
set -ue

# Use a specific taskrun if provided, otherwise use the latest
TASKRUN_NAME=${1:-$( tkn taskrun describe --last -o name )}
TASKRUN_NAME=taskrun/$( trim-name $TASKRUN_NAME )

# Let's not hard code the image url or the registry
IMAGE_DIGEST=$( kubectl get $TASKRUN_NAME -o jsonpath='{.status.taskResults[?(@.name == "IMAGE_DIGEST")].value}' )
IMAGE_URL=$( kubectl get $TASKRUN_NAME -o jsonpath='{.status.taskResults[?(@.name == "IMAGE_URL")].value}' )
IMAGE_REGISTRY=$( echo $IMAGE_URL | cut -d/ -f1 )
#IMAGE_REGISTRY=$( oc registry info )
INTERNAL_REGISTRY=$([[ $IMAGE_REGISTRY == 'image-registry.openshift-image-registry.svc:5000' ]] && echo true || echo false)
# From https://github.com/redhat-appstudio/build-definitions/ two images are created ${GIT_SHA}-1 and ${GIT_SHA}-2
# the ${GIT_SHA}-1 contains the tasks we need here, there is a race condition here, the image might not have been
# built for the HEAD ref used here, in that case wait few minutes for it to be built
APPSTUDIO_TASKS_IMGREF=quay.io/redhat-appstudio/appstudio-tasks:$(git ls-remote --heads https://github.com/redhat-appstudio/build-definitions.git refs/heads/main|cut -f 1)-1

if ! $INTERNAL_REGISTRY; then
  title "Make sure we're logged in to the registry"
  # Make sure we have a docker credential since cosign will need it
  # (Todo: Probably shouldn't assume kubeadmin user here)
  oc whoami -t | docker login -u kubeadmin --password-stdin $IMAGE_REGISTRY
fi

title "Inspect $TASKRUN_NAME annotations"
# Just want to show the chains related fields
oc get $TASKRUN_NAME -o yaml | yq-pretty .metadata.annotations
pause

title "Image url from task result"
echo "$IMAGE_URL"


title "Image digest from task result"
echo "$IMAGE_DIGEST"
echo

if $INTERNAL_REGISTRY; then
  title "Cosign verify the image - using Tekton task"
  COSIGN_VERIFY_TASK_NAME="cosign-verify-$(openssl rand --hex 10)"
  oc create -f <(echo "
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  name: $COSIGN_VERIFY_TASK_NAME
spec:
  taskRef:
    name: cosign-verify
    bundle: $APPSTUDIO_TASKS_IMGREF
  params:
    - name: PUBLIC_KEY
      value: |
$(oc get secret -n tekton-chains signing-secrets -o=go-template='{{index .data "cosign.pub"|base64decode}}' | sed 's/^/        /g')
    - name: IMAGE
      value: $IMAGE_URL
  workspaces:
    - name: sslcertdir
      secret:
        secretName: chains-ca-cert
")
  tkn taskrun logs --follow "$COSIGN_VERIFY_TASK_NAME"
  tkn taskrun describe "$COSIGN_VERIFY_TASK_NAME"
  tkn taskrun describe "$COSIGN_VERIFY_TASK_NAME" -o=go-template='{{range .status.taskResults}}{{if eq .name "VERIFY_JSON"}}{{.value}}{{end}}{{end}}' > /tmp/verify.out
else
  title "Cosign verify the image"
  # Save the output data to a file so we can look at it later
  # (Actually we could just pipe it to jq because the text goes to stderr I think..?)
  show-then-run "cosign verify --key $SIG_KEY $IMAGE_URL --output-file /tmp/verify.out"
fi
yq-pretty < /tmp/verify.out
pause

if $INTERNAL_REGISTRY; then
  title "Cosign verify the image's attestation - using Tekton task"
  COSIGN_VERIFY_ATTN_TASK_NAME="cosign-verify-attestation-$(openssl rand --hex 10)"
  oc create -f <(echo "
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  name: $COSIGN_VERIFY_ATTN_TASK_NAME
spec:
  taskRef:
    name: cosign-verify-attestation
    bundle: $APPSTUDIO_TASKS_IMGREF
  params:
    - name: PUBLIC_KEY
      value: |
$(oc get secret signing-secrets -o=go-template='{{index .data "cosign.pub"|base64decode}}' | sed 's/^/        /g')
    - name: IMAGE
      value: $IMAGE_URL
  workspaces:
    - name: sslcertdir
      secret:
        secretName: chains-ca-cert
")
  tkn taskrun logs --follow "$COSIGN_VERIFY_ATTN_TASK_NAME"
  tkn taskrun describe "$COSIGN_VERIFY_ATTN_TASK_NAME"
  tkn taskrun describe "$COSIGN_VERIFY_ATTN_TASK_NAME" -o=go-template='{{range .status.taskResults}}{{if eq .name "VERIFY_ATTESTATION_JSON"}}{{.value}}{{end}}{{end}}' > /tmp/verify-att.out
else
  title "Cosign verify the image's attestation"
  show-then-run "cosign verify-attestation --key $SIG_KEY $IMAGE_URL --output-file /tmp/verify-att.out"
fi
# There can be multiple attestations for some reason and hence multiple lines in
# this file, which makes it invalid json. For the sake of the demo we'll ignore
# all but the last line.
tail -1 /tmp/verify-att.out | yq-pretty
pause

title "Inspect the payload from that attestation output"
tail -1 /tmp/verify-att.out | yq e .payload - | base64 -d | yq-pretty
