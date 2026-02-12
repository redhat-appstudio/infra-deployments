#!/bin/bash -e
set -o pipefail

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

# Print help message
function print_help() {
  echo "Usage: $0 MODE [--obo] [--eaas] [-h|--help]"
  echo "  MODE             upstream/preview (default: upstream)"
  echo "  --obo        (only in preview mode) Install Observability operator and Prometheus instance for federation"
  echo "  --eaas       (only in preview mode) Install environment as a service components"
  echo
  echo "Example usage: \`$0 --obo --eaas"
}

OBO=false
EAAS=false

while [[ $# -gt 0 ]]; do
  key=$1
  case $key in
    --obo)
      OBO=true
      shift
      ;;
    --eaas)
      EAAS=true
      shift
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      shift
      ;;
  esac
done

if [ -f $ROOT/hack/preview.env ]; then
    source $ROOT/hack/preview.env
fi

if [ -z "$MY_GIT_FORK_REMOTE" ]; then
    echo "Set MY_GIT_FORK_REMOTE environment to name of your fork remote"
    exit 1
fi

MY_GIT_REPO_URL=$(git ls-remote --get-url $MY_GIT_FORK_REMOTE | sed 's|^git@github.com:|https://github.com/|')
MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
trap "git checkout $MY_GIT_BRANCH" EXIT


if echo "$MY_GIT_REPO_URL" | grep -q redhat-appstudio/infra-deployments; then
    echo "Use your fork repository for preview"
    exit 1
fi

# Do not allow to use default github org
if [ -z "$MY_GITHUB_ORG" ] || [ "$MY_GITHUB_ORG" == "redhat-appstudio-appdata" ]; then
    echo "Set MY_GITHUB_ORG environment variable"
    exit 1
fi

if ! git diff --exit-code --quiet; then
    echo "Changes in working Git working tree, commit them or stash them"
    exit 1
fi

# Create preview branch for preview configuration
PREVIEW_BRANCH=preview-${MY_GIT_BRANCH}${TEST_BRANCH_ID+-$TEST_BRANCH_ID}
if git rev-parse --verify $PREVIEW_BRANCH &> /dev/null; then
    git branch -D $PREVIEW_BRANCH
fi
git checkout -b $PREVIEW_BRANCH

# patch argoCD applications to point to your fork
update_patch_file () {
  local file=${1:?}

  yq -i ".[0].value = \"$MY_GIT_REPO_URL\"" "$file"
  yq -i ".[1].value = \"$PREVIEW_BRANCH\""  "$file"
}
update_patch_file "${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/application-patch.yaml"
update_patch_file "${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/application-set-patch.yaml"
update_patch_file "${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/application-set-multisrc-src-1-patch.yaml"

if $OBO ; then
  echo "Adding Observability operator and Prometheus for federation"
  yq -i '.resources += ["monitoringstack/"]' $ROOT/components/monitoring/prometheus/development/kustomization.yaml
fi

if $EAAS; then
  echo "Enabling EaaS cluster role"
  yq -i '.components += ["../../../k-components/assign-eaas-role-to-local-cluster"]' \
    $ROOT/argo-cd-apps/base/local-cluster-secret/all-in-one/kustomization.yaml
fi

# After changes introduced in https://github.com/redhat-appstudio/infra-deployments/pull/4415/files the nodes need to be labeled
nodes=$(kubectl get nodes -o name)
node_count=$(echo "$nodes" | wc -l)

for node in $nodes; do
    echo "labeling $node..."
    if kubectl label $node konflux-ci.dev/workload=konflux-tenants --overwrite; then
        echo "successfully labeled $node"
    else
        echo "failed to label $node"
    fi
done

echo "verifying labels..."
labeled_count=$(kubectl get nodes --show-labels | grep -c "konflux-ci.dev/workload=konflux-tenants")

if [ "$node_count" -eq "$labeled_count" ]; then
    echo "all nodes labeled successfully."
else
    echo "label verification failed. Labeled $labeled_count out of $node_count nodes."
    exit 1
fi

# delete argoCD applications which are not in DEPLOY_ONLY env var if it's set
if [ -n "$DEPLOY_ONLY" ]; then
  APPLICATIONS=$(\
    oc kustomize argo-cd-apps/overlays/development |\
    yq e --no-doc 'select(.kind == "ApplicationSet") | .metadata.name'
  )
  DELETED=$(yq e --no-doc .metadata.name $ROOT/argo-cd-apps/overlays/development/delete-applications.yaml)
  for APP in $APPLICATIONS; do
    if ! grep -q "\b$APP\b" <<< $DEPLOY_ONLY && ! grep -q "\b$APP\b" <<< $DELETED; then
      echo Disabling $APP based on DEPLOY_ONLY variable
      echo '---' >> $ROOT/argo-cd-apps/overlays/development/delete-applications.yaml
      yq e -n ".apiVersion=\"argoproj.io/v1alpha1\"
                 | .kind=\"ApplicationSet\"
                 | .metadata.name = \"$APP\"
                 | .\$patch = \"delete\"" >> $ROOT/argo-cd-apps/overlays/development/delete-applications.yaml
    fi
  done
fi

# Deploy Kueue if the OCP version >=4.16
OCP_MINOR=$(oc get clusterversion version -o jsonpath='{.status.desired.version}' | cut -d. -f2)
echo "Detected OCP minor version: ${OCP_MINOR}"
if [[ "$OCP_MINOR" -lt 16 ]]; then
  # Check if kueue is already in the delete-applications.yaml to prevent duplicates
  DELETE_FILE="$ROOT/argo-cd-apps/overlays/development/delete-applications.yaml"
  if ! grep -q "name: kueue" "$DELETE_FILE"; then
    echo "Adding kueue to delete-applications.yaml (OCP version < 4.16)"
    echo '---' >> "$DELETE_FILE"
    yq e -n ".apiVersion=\"argoproj.io/v1alpha1\"
              | .kind=\"ApplicationSet\"
              | .metadata.name = \"kueue\"
              | .\$patch = \"delete\"" >> "$DELETE_FILE"
  else
    echo "kueue already exists in delete-applications.yaml, skipping duplicate addition"
  fi

  # Remove kueue from policies kustomization if present
  yq -i 'del(.resources[] | select(test("^kueue/?$")))' "$ROOT/components/policies/development/kustomization.yaml"
fi

$ROOT/hack/util-set-github-org $MY_GITHUB_ORG

domain=$(oc get ingresses.config.openshift.io cluster --template={{.spec.domain}})

rekor_server="rekor.$domain"
sed -i.bak "s/rekor-server.enterprise-contract-service.svc/$rekor_server/" $ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml && rm $ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml.bak

[ -n "${BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/build-service\")) |=.newName=\"${BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build-service/development/kustomization.yaml
[ -n "${BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/build-service\")) |=.newTag=\"${BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build-service/development/kustomization.yaml
[ -n "${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}" ] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"IMAGE_TAG_ON_PR_EXPIRATION\") | .value) |= \"${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}\"" $ROOT/components/build-service/development/image-expiration-patch.yaml
[[ -n "${BUILD_SERVICE_PR_OWNER}" && "${BUILD_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/build-service*\")) |= \"https://github.com/${BUILD_SERVICE_PR_OWNER}/build-service/config/default?ref=${BUILD_SERVICE_PR_SHA}\"" $ROOT/components/build-service/development/kustomization.yaml

[ -n "${HAS_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newName=\"${HAS_IMAGE_REPO}\"" $ROOT/components/has/base/kustomization.yaml
[ -n "${HAS_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newTag=\"${HAS_IMAGE_TAG}\"" $ROOT/components/has/base/kustomization.yaml
[[ -n "${HAS_PR_OWNER}" && "${HAS_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/application-service*\")) |= \"https://github.com/${HAS_PR_OWNER}/application-service/config/default?ref=${HAS_PR_SHA}\"" $ROOT/components/has/base/kustomization.yaml

[ -n "${INTEGRATION_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/integration-service\")) |=.newName=\"${INTEGRATION_SERVICE_IMAGE_REPO}\"" $ROOT/components/integration/development/kustomization.yaml
[ -n "${INTEGRATION_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/integration-service\")) |=.newTag=\"${INTEGRATION_SERVICE_IMAGE_TAG}\"" $ROOT/components/integration/development/kustomization.yaml
[[ -n "${INTEGRATION_SERVICE_PR_OWNER}" && "${INTEGRATION_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/integration-service*\")) |= (sub(\"\?ref=.+\", \"?ref=${INTEGRATION_SERVICE_PR_SHA}\" ) | sub(\"github.com/redhat-appstudio\", \"github.com/${INTEGRATION_SERVICE_PR_OWNER}\"))" $ROOT/components/integration/development/kustomization.yaml

[ -n "${RELEASE_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/release-service\")) |=.newName=\"${RELEASE_SERVICE_IMAGE_REPO}\"" $ROOT/components/release/development/kustomization.yaml
[ -n "${RELEASE_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/release-service\")) |=.newTag=\"${RELEASE_SERVICE_IMAGE_TAG}\"" $ROOT/components/release/development/kustomization.yaml
[[ -n "${RELEASE_SERVICE_PR_OWNER}" && "${RELEASE_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/release-service*\")) |= \"https://github.com/${RELEASE_SERVICE_PR_OWNER}/release-service/config/default?ref=${RELEASE_SERVICE_PR_SHA}\"" $ROOT/components/release/development/kustomization.yaml

[ -n "${MINTMAKER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker\")) |=.newName=\"${MINTMAKER_IMAGE_REPO}\"" $ROOT/components/mintmaker/development/kustomization.yaml
[ -n "${MINTMAKER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker\")) |=.newTag=\"${MINTMAKER_IMAGE_TAG}\"" $ROOT/components/mintmaker/development/kustomization.yaml
[[ -n "${MINTMAKER_PR_OWNER}" && "${MINTMAKER_PR_SHA}" ]] && yq -i "(.resources[] | select(contains(\"konflux-ci/mintmaker\"))) |= (sub(\"konflux-ci/mintmaker\", \"${MINTMAKER_PR_OWNER}/mintmaker\") | sub(\"ref=.*\", \"ref=${MINTMAKER_PR_SHA}\"))" $ROOT/components/mintmaker/development/kustomization.yaml

[ -n "${MINTMAKER_RENOVATE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker-renovate-image\")) |=.newName=\"${MINTMAKER_RENOVATE_IMAGE_REPO}\"" $ROOT/components/mintmaker/development/kustomization.yaml
[ -n "${MINTMAKER_RENOVATE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker-renovate-image\")) |=.newTag=\"${MINTMAKER_RENOVATE_IMAGE_TAG}\"" $ROOT/components/mintmaker/development/kustomization.yaml

[ -n "${IMAGE_CONTROLLER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/image-controller\")) |=.newName=\"${IMAGE_CONTROLLER_IMAGE_REPO}\"" $ROOT/components/image-controller/development/kustomization.yaml
[ -n "${IMAGE_CONTROLLER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/image-controller\")) |=.newTag=\"${IMAGE_CONTROLLER_IMAGE_TAG}\"" $ROOT/components/image-controller/development/kustomization.yaml
[[ -n "${IMAGE_CONTROLLER_PR_OWNER}" && "${IMAGE_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/image-controller*\")) |= \"https://github.com/${IMAGE_CONTROLLER_PR_OWNER}/image-controller/config/default?ref=${IMAGE_CONTROLLER_PR_SHA}\"" $ROOT/components/image-controller/development/kustomization.yaml

[ -n "${MULTI_ARCH_CONTROLLER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"multi-platform-controller\")) |=.newName=\"${MULTI_ARCH_CONTROLLER_IMAGE_REPO}\"" $ROOT/components/multi-platform-controller/base/kustomization.yaml
[ -n "${MULTI_ARCH_CONTROLLER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"multi-platform-controller\")) |=.newTag=\"${MULTI_ARCH_CONTROLLER_IMAGE_TAG}\"" $ROOT/components/multi-platform-controller/base/kustomization.yaml
[[ -n "${MULTI_ARCH_CONTROLLER_PR_OWNER}" && "${MULTI_ARCH_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/multi-platform-controller*\")) |= (sub(\"\?ref=.+\", \"?ref=${MULTI_ARCH_CONTROLLER_PR_SHA}\" ) | sub(\"github.com/konflux-ci\", \"github.com/${MULTI_ARCH_CONTROLLER_PR_OWNER}\"))" $ROOT/components/multi-platform-controller/base/kustomization.yaml

[ -n "${EAAS_HYPERSHIFT_BASE_DOMAIN}" ] && yq -i e "(.[] | select(.value.name==\"baseDomain\")).value.value |= \"${EAAS_HYPERSHIFT_BASE_DOMAIN}\"" $ROOT/components/cluster-as-a-service/development/add-hypershift-params.yaml
[ -n "${EAAS_HYPERSHIFT_CLI_ROLE_ARN}" ] && yq -i e "(.[] | select(.value.name==\"hypershiftRoleArn\")).value.value |= \"${EAAS_HYPERSHIFT_CLI_ROLE_ARN}\"" $ROOT/components/cluster-as-a-service/development/add-hypershift-params.yaml

# Configure smee.io channel URL for webhook forwarding (used for Forgejo/Codeberg testing)
[ -n "${SMEE_CHANNEL}" ] && yq -i e ".[].value = \"${SMEE_CHANNEL}\"" $ROOT/components/smee-client/development/sever-url-patch.yaml

[[ -n "${PIPELINE_PR_OWNER}" && "${PIPELINE_PR_SHA}" ]] && yq -i e ".resources[] |= sub(\"ref=[^ ]*\"; \"ref=${PIPELINE_PR_SHA}\") | .resources[] |= sub(\"openshift-pipelines\"; \"${PIPELINE_PR_OWNER}\")" $ROOT/components/pipeline-service/development/kustomization.yaml

if ! git diff --exit-code --quiet; then
    git commit -a -m "Preview mode, do not merge into main"
    git push -f --set-upstream $MY_GIT_FORK_REMOTE $PREVIEW_BRANCH
fi

# Create the root Application
oc apply -k $ROOT/argo-cd-apps/app-of-app-sets/development

while [ "$(oc get applications.argoproj.io all-application-sets -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; do
  echo Waiting for sync of all-application-sets argoCD app
  sleep 5
done

APPS=$(oc get apps -n openshift-gitops -o name)
# trigger refresh of apps
for APP in $APPS; do
  oc patch $APP -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}' &
done
wait

# wait for the refresh
while [ -n "$(oc get applications.argoproj.io -n openshift-gitops -o jsonpath='{range .items[*]}{@.metadata.annotations.argocd\.argoproj\.io/refresh}{end}')" ]; do
  sleep 5
done

INTERVAL=10
while :; do
  STATE=$(oc get apps -n openshift-gitops --no-headers)
  NOT_DONE=$(echo "$STATE" | grep -v "Synced[[:blank:]]*Healthy" || true)
  echo "$NOT_DONE"
  if [ -z "$NOT_DONE" ]; then
     echo All Applications are synced and Healthy
     break
  else
     UNKNOWN=$(echo "$NOT_DONE" | grep Unknown | grep -v Progressing | cut -f1 -d ' ') || :
     if [ -n "$UNKNOWN" ]; then
       for app in $UNKNOWN; do
         ERROR=$(oc get -n openshift-gitops applications.argoproj.io $app -o jsonpath='{.status.conditions}')
         if echo "$ERROR" | grep -q 'context deadline exceeded'; then
           echo Refreshing $app
           oc patch applications.argoproj.io $app -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "soft"}}}'
           while [ -n "$(oc get applications.argoproj.io -n openshift-gitops $app -o jsonpath='{.metadata.annotations.argocd\.argoproj\.io/refresh}')" ]; do
             sleep 5
           done
           echo Refresh of $app done
           continue 2
         fi
         echo $app failed with:
         if [ -n "$ERROR" ]; then
           echo "$ERROR"
         else
           oc get -n openshift-gitops applications.argoproj.io $app -o yaml
         fi
       done
       exit 1
     fi
     echo Waiting $INTERVAL seconds for application sync
     sleep $INTERVAL
  fi
done

# Wait for all tekton components to be installed
# The status of a tektonconfig CR should be "type: Ready, status: True" once the install is completed
# More info: https://tekton.dev/docs/operator/tektonconfig/#tekton-config
while :; do
  STATE=$(oc get tektonconfig config -o json | jq -r '.status.conditions[] | select(.type == "Ready")')
  [ "$(jq -r '.status' <<< "$STATE")" == "True" ] && echo All required tekton resources are installed and ready && break
  echo Some tekton resources are not ready yet:
  jq -r '.message' <<< "$STATE"
  # start temporary work around for https://issues.redhat.com/browse/SRVKP-3245
  MSG=$(jq -r '.message' <<< "$STATE")
  if echo "$MSG" | grep -q 'Components not in ready state: OpenShiftPipelinesAsCode: reconcile again and proceed'; then
    if [[ "$(oc get deployment/pipelines-as-code-controller -n openshift-pipelines -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')" != "True" ]]; then
      echo "pipelines-as-code-controller still not available"
      continue
    fi
    if [[ "$(oc get deployment/pipelines-as-code-watcher -n openshift-pipelines -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')" != "True" ]]; then
      echo "pipelines-as-code-watcher still not available"
      continue
    fi
    if [[ "$(oc get deployment/pipelines-as-code-webhook -n openshift-pipelines -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')" != "True" ]]; then
      echo "pipelines-as-code-webhook still not available"
      continue
    fi
    echo "BYPASSING tektonconfig CHECK BECAUSE OF https://issues.redhat.com/browse/SRVKP-3245 FOR OpenShiftPipelinesAsCode"
    break
  fi
  # end temporary work around for https://issues.redhat.com/browse/SRVKP-3245
  sleep $INTERVAL
done

# Sometimes Tekton CRDs need a few mins to be ready
retry=0
while true; do
  if [ "$retry" -eq 5 ]; then
    printf "Error: Tekton CRDs are not yet available on the cluster.\n" >&2
    exit 1
  fi
  tekton_crds=$(oc api-resources --api-group="tekton.dev" --no-headers)
  if [[ $tekton_crds =~ pipelines && $tekton_crds =~ tasks && $tekton_crds =~ pipelineruns && $tekton_crds =~ taskruns ]]; then
    echo "Tekton CRDs are ready"
    break
  fi
  sleep $INTERVAL
  retry=$((retry + 1))
done

# Configure Pipelines as Code and required credentials
$ROOT/hack/build/setup-pac-integration.sh

