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
#                      verify-enterprise-contract-v2 task
#
set -euo pipefail

SRC_IMAGE_REF="$1"
DST_IMAGE_REF="$2"

NAMESPACE="$(oc get sa default -o jsonpath='{.metadata.namespace}')"
SIG_KEY="k8s://$NAMESPACE/cosign-public-key"

# The image used by the verify-enterprise-contract-v2 task. This image is usually
# built by the 'hack/build-and-push.sh' script from the
# https://github.com/redhat-appstudio/build-definitions repository.
DEFAULT_TASK_BUNDLE='quay.io/redhat-appstudio/appstudio-tasks:50f736328e1e426af7c7751cae2326be7857685b-3'
TASK_BUNDLE="${TASK_BUNDLE:-${DEFAULT_TASK_BUNDLE}}"

REKOR_HOST="$($(dirname $0)/config.sh get | yq -e '."transparency.url" // ""')"
if [[ -z $REKOR_HOST ]]; then
    REKOR_HOST="https://rekor.sigstore.dev"
fi

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
  - name: IMAGES
    type: string
    description: >-
        Spec of ApplicationSnapshot containing the images to verify.
        For demo purposes, it should contain a single image reference.
  - name: SRC_IMAGE_REF
    type: string
    description: >-
        Reference to copy the image from. Outside of a demo environment,
        this should be extracted from the ApplicationSnapshot to ensure
        the image being promoted is the same one being verified.
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
      name: verify-enterprise-contract-v2
      kind: Task
      bundle: ${TASK_BUNDLE}
    params:
    - name:  IMAGES
      value: \$(params.IMAGES)
    - name: PUBLIC_KEY
      value: \$(params.PUBLIC_KEY)
    - name: REKOR_HOST
      value: ${REKOR_HOST}
    - name: SSL_CERT_DIR
      value: /var/run/secrets/kubernetes.io/serviceaccount
    # These are here to facilitate alternate versions of the demo
    # - name: STRICT
    #   value: \"false\"
    # - name: POLICY_CONFIGURATION
    #   value: \"ec-policy\"

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

oc get EnterpriseContractPolicy ec-policy >/dev/null 2>&1 || echo -n '
---
kind: EnterpriseContractPolicy
apiVersion: appstudio.redhat.com/v1alpha1
metadata:
  name: ec-policy
spec:
  description: Red Hat enterprise requirements
  exceptions:
    nonBlocking:
    - not_useful
    - test
    - tasks
  sources:
  - git:
      repository: https://github.com/hacbs-contract/ec-policies
      revision: main
' | oc apply -f - > /dev/null

oc get EnterpriseContractPolicy ec-policy -o yaml | yq e '.spec' -

title "ApplicationSnapshot"

echo -n "
---
apiVersion: appstudio.redhat.com/v1alpha1
kind: ApplicationSnapshot
metadata:
  name: demo
spec:
  application: demo-app
  components:
    - containerImage: ${SRC_IMAGE_REF}
      name: demo-component
" | oc apply -f - > /dev/null
oc get ApplicationSnapshot demo -o yaml | yq e '.spec' -

# The spec section of the ApplicationSnaptshot is what is expected by the
# verify-enterprise-contract-v2 task.
IMAGES="$(oc get ApplicationSnapshot demo -o json | jq '.spec | tostring' -r)"

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
show-then-run "tkn pipeline start simple-release --param IMAGES=${IMAGES} --param SRC_IMAGE_REF=${SRC_IMAGE_REF} --param DST_IMAGE_REF=${DST_IMAGE_REF} --param PUBLIC_KEY=${SIG_KEY} --workspace name=images-url,emptyDir= --showlog"
