#!/bin/bash -e
set -o pipefail

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

# Print help message
function print_help() {
  echo "Usage: $0 MODE [--toolchain] [--keycloak] [--obo] [-h|--help]"
  echo "  MODE             upstream/preview (default: upstream)"
  echo "  --toolchain  (only in preview mode) Install toolchain operators"
  echo "  --keycloak   (only in preview mode) Configure the toolchain operator to use keycloak deployed on the cluster"
  echo "  --obo        (only in preview mode) Install Observability operator and Prometheus instance for federation"
  echo
  echo "Example usage: \`$0 --toolchain --keycloak --obo"
}
TOOLCHAIN=false
KEYCLOAK=false
OBO=false

while [[ $# -gt 0 ]]; do
  key=$1
  case $key in
    --toolchain)
      TOOLCHAIN=true
      shift
      ;;
    --keycloak)
      KEYCLOAK=true
      shift
      ;;
    --obo)
      OBO=true
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



if $TOOLCHAIN ; then
  echo "Deploying toolchain"
  "$ROOT/hack/sandbox-development-mode.sh"

  if $KEYCLOAK; then
    echo "Patching toolchain config to use keylcoak installed on the cluster"

    BASE_URL=$(oc get ingresses.config.openshift.io/cluster -o jsonpath={.spec.domain})
    RHSSO_URL="https://keycloak-dev-sso.$BASE_URL"

    oc patch ToolchainConfig/config -n toolchain-host-operator --type=merge --patch-file=/dev/stdin << EOF
spec:
  host:
    registrationService:
      auth:
        authClientConfigRaw: '{
                  "realm": "redhat-external",
                  "auth-server-url": "$RHSSO_URL/auth",
                  "ssl-required": "none",
                  "resource": "cloud-services",
                  "clientId": "cloud-services",
                  "public-client": true
                }'
        authClientLibraryURL: $RHSSO_URL/auth/js/keycloak.js
        authClientPublicKeysURL: $RHSSO_URL/auth/realms/redhat-external/protocol/openid-connect/certs
EOF
  fi
fi

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

if $OBO ; then
  echo "Adding Observability operator and Prometheus for federation"
  yq -i '.resources += ["monitoringstack/"]' $ROOT/components/monitoring/prometheus/development/kustomization.yaml
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

# set the API server which SPI uses to authenticate users to empty string (by default) so that multi-cluster
# setup is not needed
yq -i e ".0.value=\"$SPI_API_SERVER\"" $ROOT/components/spi/overlays/development/oauth-service-config-patch.json
# patch the SPI configuration with the Vault host configuration to provided VAULT_HOST variable or to current cluster
# and the base URL set to the SPI_BASE_URL variable or the URL of the  route to the SPI OAuth service in the current cluster
# This script also sets up the Vault client to accept insecure TLS connections so that the custom vault host doesn't have
# to serve requests using a trusted TLS certificate.
$ROOT/hack/util-patch-spi-config.sh
# configure the secrets and providers in SPI
TMP_FILE=$(mktemp)
yq e ".serviceProviders[0].type=\"GitHub\"" $ROOT/components/spi/base/config.yaml | \
    yq e ".serviceProviders[0].clientId=\"${SPI_GITHUB_CLIENT_ID:-app-client-id}\"" - | \
    yq e ".serviceProviders[0].clientSecret=\"${SPI_GITHUB_CLIENT_SECRET:-app-secret}\"" - | \
    yq e ".serviceProviders[1].type=\"Quay\"" - | \
    yq e ".serviceProviders[1].clientId=\"${SPI_QUAY_CLIENT_ID:-app-client-id}\"" - | \
    yq e ".serviceProviders[1].clientSecret=\"${SPI_QUAY_CLIENT_SECRET:-app-secret}\"" - > $TMP_FILE
oc create namespace spi-system --dry-run=client -o yaml | oc apply -f -
oc create -n spi-system secret generic shared-configuration-file --from-file=config.yaml=$TMP_FILE --dry-run=client -o yaml | oc apply -f -
echo "SPI configured"
rm $TMP_FILE

$ROOT/hack/util-set-github-org $MY_GITHUB_ORG

domain=$(oc get ingresses.config.openshift.io cluster --template={{.spec.domain}})

rekor_server="rekor.$domain"
sed -i.bak "s/rekor-server.enterprise-contract-service.svc/$rekor_server/" $ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml && rm $ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml.bak

[ -n "${BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/build-service\")) |=.newName=\"${BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build-service/base/kustomization.yaml
[ -n "${BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/build-service\")) |=.newTag=\"${BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build-service/base/kustomization.yaml
[ -n "${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}" ] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"IMAGE_TAG_ON_PR_EXPIRATION\") | .value) |= \"${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}\"" $ROOT/components/build-service/development/image-expiration-patch.yaml
[[ -n "${BUILD_SERVICE_PR_OWNER}" && "${BUILD_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/build-service*\")) |= \"https://github.com/${BUILD_SERVICE_PR_OWNER}/build-service/config/default?ref=${BUILD_SERVICE_PR_SHA}\"" $ROOT/components/build-service/base/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newName=\"${JVM_BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/jvm-build-service/base/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newTag=\"${JVM_BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/jvm-build-service/base/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_PR_OWNER}" && "${JVM_BUILD_SERVICE_PR_SHA}" ]] && sed -i -e "s|\(https://github.com/\)redhat-appstudio\(/jvm-build-service/.*?ref=\)\(.*\)|\1${JVM_BUILD_SERVICE_PR_OWNER}\2${JVM_BUILD_SERVICE_PR_SHA}|" -e "s|\(https://raw.githubusercontent.com/\)redhat-appstudio\(/jvm-build-service/\)[^/]*\(/.*\)|\1${JVM_BUILD_SERVICE_PR_OWNER}\2${JVM_BUILD_SERVICE_PR_SHA}\3|" $ROOT/components/jvm-build-service/base/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_CACHE_IMAGE}" && ${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE} ]] && yq -i e "select(.[].path == \"/spec/template/spec/containers/0/env\") | .[].value |= . + [{\"name\" : \"JVM_BUILD_SERVICE_CACHE_IMAGE\", \"value\": \"${JVM_BUILD_SERVICE_CACHE_IMAGE}\"}, {\"name\": \"JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE\", \"value\": \"${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE}\"}] | (.[].value[] | select(.name == \"IMAGE_TAG\")) |= .value = \"\"" $ROOT/components/jvm-build-service/base/operator_env_patch.yaml

[ -n "${HAS_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newName=\"${HAS_IMAGE_REPO}\"" $ROOT/components/has/base/kustomization.yaml
[ -n "${HAS_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newTag=\"${HAS_IMAGE_TAG}\"" $ROOT/components/has/base/kustomization.yaml
[[ -n "${HAS_PR_OWNER}" && "${HAS_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/application-service*\")) |= \"https://github.com/${HAS_PR_OWNER}/application-service/config/default?ref=${HAS_PR_SHA}\"" $ROOT/components/has/base/kustomization.yaml

[ -n "${INTEGRATION_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/integration-service\")) |=.newName=\"${INTEGRATION_SERVICE_IMAGE_REPO}\"" $ROOT/components/integration/kustomization.yaml
[ -n "${INTEGRATION_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/integration-service\")) |=.newTag=\"${INTEGRATION_SERVICE_IMAGE_TAG}\"" $ROOT/components/integration/kustomization.yaml
[[ -n "${INTEGRATION_SERVICE_PR_OWNER}" && "${INTEGRATION_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/integration-service*\")) |= \"https://github.com/${INTEGRATION_SERVICE_PR_OWNER}/integration-service/config/default?ref=${INTEGRATION_SERVICE_PR_SHA}\"" $ROOT/components/integration/kustomization.yaml

[ -n "${RELEASE_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/release-service\")) |=.newName=\"${RELEASE_SERVICE_IMAGE_REPO}\"" $ROOT/components/release/kustomization.yaml
[ -n "${RELEASE_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/release-service\")) |=.newTag=\"${RELEASE_SERVICE_IMAGE_TAG}\"" $ROOT/components/release/kustomization.yaml
[[ -n "${RELEASE_SERVICE_PR_OWNER}" && "${RELEASE_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/release-service*\")) |= \"https://github.com/${RELEASE_SERVICE_PR_OWNER}/release-service/config/default?ref=${RELEASE_SERVICE_PR_SHA}\"" $ROOT/components/release/kustomization.yaml
[ -n "${SPI_OPERATOR_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/service-provider-integration-operator\")) |=.newName=\"${SPI_OPERATOR_IMAGE_REPO}\"" $ROOT/components/spi/overlays/development/kustomization.yaml
[ -n "${SPI_OPERATOR_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/service-provider-integration-operator\")) |=.newTag=\"${SPI_OPERATOR_IMAGE_TAG}\"" $ROOT/components/spi/overlays/development/kustomization.yaml
[ -n "${SPI_OAUTH_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/service-provider-integration-oauth\")) |=.newName=\"${SPI_OAUTH_IMAGE_REPO}\"" $ROOT/components/spi/overlays/development/kustomization.yaml
[ -n "${SPI_OAUTH_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/service-provider-integration-oauth\")) |=.newTag=\"${SPI_OAUTH_IMAGE_TAG}\"" $ROOT/components/spi/overlays/development/kustomization.yaml

[ -n "${IMAGE_CONTROLLER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/image-controller\")) |=.newName=\"${IMAGE_CONTROLLER_IMAGE_REPO}\"" $ROOT/components/image-controller/base/kustomization.yaml
[ -n "${IMAGE_CONTROLLER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/image-controller\")) |=.newTag=\"${IMAGE_CONTROLLER_IMAGE_TAG}\"" $ROOT/components/image-controller/base/kustomization.yaml
[[ -n "${IMAGE_CONTROLLER_PR_OWNER}" && "${IMAGE_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/image-controller*\")) |= \"https://github.com/${IMAGE_CONTROLLER_PR_OWNER}/image-controller/config/default?ref=${IMAGE_CONTROLLER_PR_SHA}\"" $ROOT/components/image-controller/base/kustomization.yaml

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

# Init Vault
$ROOT/hack/spi/vault-init.sh

# Init SPI
$ROOT/hack/spi/spi-init.sh

# Init Remote Secret
$ROOT/hack/spi/remote-secret-init.sh

# Configure Pipelines as Code and required credentials
$ROOT/hack/build/setup-pac-integration.sh

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


if $KEYCLOAK && $TOOLCHAIN ; then
  echo "Restarting toolchain registration service to pick up keycloak's certs."
  oc rollout restart StatefulSet/keycloak -n dev-sso
  oc wait --for=condition=Ready pod/keycloak-0 -n dev-sso --timeout=5m

  oc delete deployment/registration-service -n toolchain-host-operator
  # Wait for the new deployment to be available
  timeout --foreground 5m bash  <<- "EOF"
		while [[ "$(oc get deployment/registration-service -n toolchain-host-operator -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')" != "True" ]]; do
			echo "Waiting for registration-service to be available again"
			sleep 2
		done
		EOF
  if [ $? -ne 0 ]; then
	  echo "Timed out waiting for registration-service to be available"
	  exit 1
  fi
fi
