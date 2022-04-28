#!/bin/bash
#
# A helper script to demonstrate a pipeline that performs a release if, and only if, the
# Enterprise Contract is met.
#
# Pre-requisites:
#
#   1. Create a Secret named "release-demo" with access to push to the dst-image-ref, and
#      link it to the pipeline ServiceAccount (so skopeo can copy the image there):
#      oc create secret docker-registry release-demo quay.io --docker-username=$USER --docker-password=top-secret
#      oc secrets link pipeline release-demo --for=pull,mount
#
#   2. Create a Secret named "cosign-public-key" with the "cosign.pub" attribute to house the
#      public key to be used for verifying the signature of the image and its attestation:
#      ./copy-public-sig-key.sh
#
#
# Usage:
#   release-pipeline-with-ec-demo.sh <src-image-ref> <dst-image-ref>
# Where:
#   <src-image-ref> string pointing to the image to be released, e.g.
#                   quay.io/spam/bacon:latest
#   <dst-image-ref> string pointing to the location where the image
#                   image will be released to, e.g.
#                   quay.io/spam/bacon:yummy
# Environment Variables:
#   TASK_BUNDLE        image reference to the tekton bundle containing the
#                      verify-enterprise-contract task
#   BUILD_PIPELINE_RUN pipeline run to examine, otherwise auto-guessed
#
set -euo pipefail

SRC_IMAGE_REF="$1"
DST_IMAGE_REF="$2"

NAMESPACE="$(oc get sa default -o jsonpath='{.metadata.namespace}')"
SIG_KEY="k8s://$NAMESPACE/cosign-public-key"

# The image used by the verify-enterprise-contract task. This image is usually
# built by the 'hack/build-and-push.sh' script from the
# https://github.com/redhat-appstudio/build-definitions repository.
DEFAULT_TASK_BUNDLE='quay.io/redhat-appstudio/appstudio-tasks:f647b6ced45f59f1cddbd9b8ae9b560173d1cc1b-2'
TASK_BUNDLE="${TASK_BUNDLE:-${DEFAULT_TASK_BUNDLE}}"

# Finds the first PipelineRun that has a TaskRun with the result named 'HACBS_TEST_OUTPUT'
FALLBACK_PIPELINE_RUN=$(kubectl get taskruns -o go-template='{{ $found := false }}{{ range $tr := .items }}{{ range $tr.status.taskResults }}{{ if and (eq .name "HACBS_TEST_OUTPUT") (not $found) }}{{ index $tr.metadata.labels "tekton.dev/pipelineRun" }}{{ $found = true }}{{ end }}{{ end }}{{ end }}')

source $(dirname $0)/_helpers.sh


title "Simple Release Pipeline"

echo -n "
---
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: simple-release
spec:
  params:
  - name: SRC_IMAGE_REF
    type: string
    description: Image reference to verify
  - name: DST_IMAGE_REF
    type: string
    description: Reference to copy the image to
  - name: PUBLIC_KEY
    type: string
    description: >-
      Public key used to verify signatures. Must be a valid k8s cosign
      reference, e.g. k8s://my-space/my-secret where my-secret contains
      the expected cosign.pub attribute.
  tasks:
  - name: ec
    taskRef:
      name: verify-enterprise-contract
      bundle: ${TASK_BUNDLE}
    params:
    - name:  IMAGE_REF
      value: \$(params.SRC_IMAGE_REF)
    - name: PUBLIC_KEY
      value: \$(params.PUBLIC_KEY)
    - name: PIPELINERUN_NAME
      value: ${BUILD_PIPELINE_RUN:-$FALLBACK_PIPELINE_RUN}
    # These are here to facilitate alternate versions of the demo
    # - name: COSIGN_EXPERIMENTAL
    #   value: "0"
    # - name: POLICY_REPO
    #   value: https://github.com/hacbs-contract/ec-policies.git
    # - name: POLICY_REPO_REF
    #   value: main
    # - name: STRICT_POLICY
    #   value: "0"
    # - name: REKOR_HOST
    #   value: https://<rekor-server-ingress-route>
    # # Set SSL_CERT_DIR to the value below when using rekor-local
    # - name: SSL_CERT_DIR
    #   value: /var/run/secrets/kubernetes.io/serviceaccount

  - name: release
    taskRef:
      name: skopeo-copy
      kind: ClusterTask
    params:
    - name: srcImageURL
      value: docker://\$(params.SRC_IMAGE_REF)
    - name: destImageURL
      value: docker://\$(params.DST_IMAGE_REF)
    runAfter:
    - ec
    workspaces:
    - name: images-url
      workspace: images-url

  workspaces:
  - name: images-url

" | oc apply -f - > /dev/null

oc get pipeline simple-release -o yaml | yq e '.spec' -

title "Enterprise Contract Policy"

oc get configmap ec-policy >/dev/null 2>&1 || echo -n '
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: ec-policy
data:
  # HINT: Add "test" to the list to ignore tests
  policy.json: |
      {"non_blocking_checks":["not_useful"]}
' | oc create -f - > /dev/null

oc get configmap ec-policy -o json | jq '.data."policy.json" | fromjson'

title "Verify Push Secret"

# Check if required secret exists
oc get secret release-demo > /dev/null
echo "The 'pipeline' ServiceAccount has access to these registries:"
oc get secret -o json \
    $(oc get sa pipeline -o json | jq '.secrets[].name' -r | xargs) | \
    jq '.items[] | .data | .".dockercfg" // .".dockerconfigjson" // ""? |
        @base64d | fromjson | .auths // . | keys[] | " * "+.' -r | sort -u


title "Run the Simple Release Pipeline"

# Check if required secret exists
oc get secret cosign-public-key > /dev/null
show-then-run "tkn pipeline start simple-release --param SRC_IMAGE_REF=${SRC_IMAGE_REF} --param DST_IMAGE_REF=${DST_IMAGE_REF} --param PUBLIC_KEY=${SIG_KEY} --workspace name=images-url,emptyDir= --showlog"
