#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [ -f $ROOT/hack/preview.env ]; then
    source $ROOT/hack/preview.env
fi

if [ -z "$MY_GIT_FORK_REMOTE" ]; then
    echo "Set MY_GIT_FORK_REMOTE environment to name of your fork remote"
    exit 1
fi

MY_GIT_REPO_URL=$(git ls-remote --get-url $MY_GIT_FORK_REMOTE | sed 's|^git@github.com:|https://github.com/|')
MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)


if echo "$MY_GIT_REPO_URL" | grep -q redhat-appstudio/infra-deployments; then
    echo "Use your fork repository for preview"
    exit 1
fi

if ! git diff --exit-code --quiet; then
    echo "Changes in working Git working tree, commit them or stash them"
    exit 1
fi

# Create preview branch for preview configuration
PREVIEW_BRANCH=preview-${MY_GIT_BRANCH}${TEST_BRANCH_ID+-$TEST_BRANCH_ID}
if git rev-parse --verify $PREVIEW_BRANCH; then
    git branch -D $PREVIEW_BRANCH
fi
git checkout -b $PREVIEW_BRANCH

# reset the default repos in the development directory to be the current git repo
# this needs to be pushed to your fork to be seen by argocd
$ROOT/hack/util-set-development-repos.sh $MY_GIT_REPO_URL development $PREVIEW_BRANCH

# set the API server which SPI uses to authenticate users to empty string (by default) so that multi-cluster
# setup is not needed
yq -i e ".0.value.0.value=\"$SPI_API_SERVER\"" $ROOT/components/spi/oauth-service-deployment-patch.json

# set backend route for quality dashboard for current cluster
$ROOT/hack/util-set-quality-dashboard-backend-route.sh

if [ -n "$MY_GITHUB_ORG" ]; then
    $ROOT/hack/util-set-github-org $MY_GITHUB_ORG
fi

domain=$(kubectl get ingresses.config.openshift.io cluster --template={{.spec.domain}})

# set SPI
TMP_FILE=$(mktemp)
SPI_BASE_URL="https://spi-oauth-route-spi-system.${domain}"
yq e ".sharedSecret=\"${SHARED_SECRET:-$(openssl rand -hex 20)}\"" $ROOT/components/spi/config.yaml | \
    yq e ".serviceProviders[0].type=\"${SPI_TYPE:-GitHub}\"" - | \
    yq e ".serviceProviders[0].clientId=\"${SPI_CLIENT_ID:-app-client-id}\"" - | \
    yq e ".serviceProviders[0].clientSecret=\"${SPI_CLIENT_SECRET:-app-secret}\"" - | \
    yq e ".baseUrl=\"$SPI_BASE_URL\"" - > $TMP_FILE
oc create -n spi-system secret generic oauth-config --from-file=config.yaml=$TMP_FILE --dry-run=client -o yaml | oc apply -f -
echo "SPI configurared, set Authorization callback URL to $SPI_BASE_URL"
rm $TMP_FILE

if [ -n "$DOCKER_IO_AUTH" ]; then
    AUTH=$(mktemp)
    # Set global pull secret
    oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > $AUTH
    oc registry login --registry=docker.io --auth-basic=$DOCKER_IO_AUTH --to=$AUTH
    oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=$AUTH
    # Set current namespace pipeline serviceaccount which is used by buildah
    oc create secret docker-registry docker-io-pull --from-file=.dockerconfigjson=$AUTH -o yaml --dry-run=client | oc apply -f-
    oc secrets link pipeline docker-io-pull
    rm $AUTH
fi

rekor_server="rekor.$domain"
sed -i "s/rekor-server.enterprise-contract-service.svc/$rekor_server/" $ROOT/argo-cd-apps/base/enterprise-contract.yaml
yq -i e ".data |= .\"transparency.url\"=\"https://$rekor_server\"" $ROOT/components/build/tekton-chains/chains-config.yaml

[ -n "${BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/build-service\")) |=.newName=\"${BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build/build-service/kustomization.yaml
[ -n "${BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/build-service\")) |=.newTag=\"${BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build/build-service/kustomization.yaml
[[ -n "${BUILD_SERVICE_PR_OWNER}" && "${BUILD_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/build-service*\")) |= \"https://github.com/${BUILD_SERVICE_PR_OWNER}/build-service/config/default?ref=${BUILD_SERVICE_PR_SHA}\"" $ROOT/components/build/build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newName=\"${JVM_BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newTag=\"${JVM_BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_PR_OWNER}" && "${JVM_BUILD_SERVICE_PR_SHA}" ]] && sed -i -e "s|\(https://github.com/\)redhat-appstudio\(/jvm-build-service/.*?ref=\)\(.*\)|\1${JVM_BUILD_SERVICE_PR_OWNER}\2${JVM_BUILD_SERVICE_PR_SHA}|" -e "s|\(https://raw.githubusercontent.com/\)redhat-appstudio\(/jvm-build-service/\)[^/]*\(/.*\)|\1${JVM_BUILD_SERVICE_PR_OWNER}\2${JVM_BUILD_SERVICE_PR_SHA}\3|" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newName=\"${JVM_BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newTag=\"${JVM_BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_CACHE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-cache\")) |=.newName=\"${JVM_BUILD_SERVICE_CACHE_IMAGE_REPO}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_CACHE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-cache\")) |=.newTag=\"${JVM_BUILD_SERVICE_CACHE_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_SIDECAR_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"run-maven-component-build\")) |=.newName=\"${JVM_BUILD_SERVICE_SIDECAR_IMAGE_REPO}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_SIDECAR_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"run-maven-component-build\")) |=.newTag=\"${JVM_BUILD_SERVICE_SIDECAR_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_SIDECAR_IMAGE_REPO}" && ${JVM_BUILD_SERVICE_SIDECAR_IMAGE_TAG} ]] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"JVM_BUILD_SERVICE_SIDECAR_IMAGE\")) |=.value=\"${JVM_BUILD_SERVICE_SIDECAR_IMAGE_REPO}:${JVM_BUILD_SERVICE_SIDECAR_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/operator-images.yaml
[[ -n "${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE_REPO}" && ${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE_TAG} ]] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE\")) |=.value=\"${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE_REPO}:${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/operator-images.yaml
[ -n "${JVM_DELETE_TASKRUN_PODS}" ] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"JVM_DELETE_TASKRUN_PODS\")) |=.value=\"${JVM_DELETE_TASKRUN_PODS}\"" $ROOT/components/build/jvm-build-service/operator-images.yaml
[ -n "${DEFAULT_BUILD_BUNDLE}" ] && yq -i e "(.configMapGenerator[].literals[] | select(. == \"default_build_bundle*\")) |= \"default_build_bundle=${DEFAULT_BUILD_BUNDLE}\"" $ROOT/components/build/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_CACHE_IMAGE_REPO}" && ${JVM_BUILD_SERVICE_CACHE_IMAGE_TAG} ]] && yq -i e "(.data.\"image.cache\") |=.=\"${JVM_BUILD_SERVICE_CACHE_IMAGE_REPO}:${JVM_BUILD_SERVICE_CACHE_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/system-config.yaml
[[ -n "${JVM_BUILD_SERVICE_JDK8_BUILDER_IMAGE_REPO}" && ${JVM_BUILD_SERVICE_JDK8_BUILDER_IMAGE_TAG} ]] && yq -i e "(.data.\"builder-image.jdk8.image\") |=.=\"${JVM_BUILD_SERVICE_JDK8_BUILDER_IMAGE_REPO}:${JVM_BUILD_SERVICE_JDK8_BUILDER_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/system-config.yaml
[[ -n "${JVM_BUILD_SERVICE_JDK11_BUILDER_IMAGE_REPO}" && ${JVM_BUILD_SERVICE_JDK11_BUILDER_IMAGE_TAG} ]] && yq -i e "(.data.\"builder-image.jdk11.image\") |=.=\"${JVM_BUILD_SERVICE_JDK11_BUILDER_IMAGE_REPO}:${JVM_BUILD_SERVICE_JDK11_BUILDER_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/system-config.yaml
[[ -n "${JVM_BUILD_SERVICE_JDK17_BUILDER_IMAGE_REPO}" && ${JVM_BUILD_SERVICE_JDK17_BUILDER_IMAGE_TAG} ]] && yq -i e "(.data.\"builder-image.jdk17.image\") |=.=\"${JVM_BUILD_SERVICE_JDK17_BUILDER_IMAGE_REPO}:${JVM_BUILD_SERVICE_JDK17_BUILDER_IMAGE_TAG}\"" $ROOT/components/build/jvm-build-service/system-config.yaml


[ -n "${HAS_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newName=\"${HAS_IMAGE_REPO}\"" $ROOT/components/has/kustomization.yaml
[ -n "${HAS_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newTag=\"${HAS_IMAGE_TAG}\"" $ROOT/components/has/kustomization.yaml
[[ -n "${HAS_PR_OWNER}" && "${HAS_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/application-service*\")) |= \"https://github.com/${HAS_PR_OWNER}/application-service/config/default?ref=${HAS_PR_SHA}\"" $ROOT/components/has/kustomization.yaml
[ -n "${HAS_DEFAULT_IMAGE_REPOSITORY}" ] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name ==\"IMAGE_REPOSITORY\").value) |= \"${HAS_DEFAULT_IMAGE_REPOSITORY}\"" $ROOT/components/has/manager_resources_patch.yaml

[ -n "${INTEGRATION_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/integration-service\")) |=.newName=\"${INTEGRATION_IMAGE_REPO}\"" $ROOT/components/integration/kustomization.yaml
[ -n "${INTEGRATION_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/integration-service\")) |=.newTag=\"${INTEGRATION_IMAGE_TAG}\"" $ROOT/components/integration/kustomization.yaml
[ -n "${INTEGRATION_RESOURCES}" ] && yq -i e "(.resources[] | select(.==\"https://github.com/redhat-appstudio/integration-service/config/default?ref=*\")) |=.=\"${INTEGRATION_RESOURCES}\"" $ROOT/components/integration/kustomization.yaml

[ -n "${RELEASE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/release-service\")) |=.newName=\"${RELEASE_IMAGE_REPO}\"" $ROOT/components/release/kustomization.yaml
[ -n "${RELEASE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/release-service\")) |=.newTag=\"${RELEASE_IMAGE_TAG}\"" $ROOT/components/release/kustomization.yaml
[ -n "${RELEASE_RESOURCES}" ] && yq -i e "(.resources[] | select(.==\"https://github.com/redhat-appstudio/release-service/config/default?ref=*\")) |=.=\"${RELEASE_RESOURCES}\"" $ROOT/components/release/kustomization.yaml
[ -n "${SPI_OPERATOR_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/service-provider-integration-operator\")) |=.newName=\"${SPI_OPERATOR_IMAGE_REPO}\"" $ROOT/components/spi/kustomization.yaml
[ -n "${SPI_OPERATOR_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/service-provider-integration-operator\")) |=.newTag=\"${SPI_OPERATOR_IMAGE_TAG}\"" $ROOT/components/spi/kustomization.yaml

if ! git diff --exit-code --quiet; then
    git commit -a -m "Preview mode, do not merge into main"
    git push -f --set-upstream $MY_GIT_FORK_REMOTE $PREVIEW_BRANCH
fi

git checkout $MY_GIT_BRANCH

#set the local cluster to point to the current git repo and branch and update the path to development
$ROOT/hack/util-update-app-of-apps.sh $MY_GIT_REPO_URL development $PREVIEW_BRANCH

while [ "$(oc get applications.argoproj.io all-components-staging -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; do
  sleep 5
done

APPS=$(kubectl get apps -n openshift-gitops -o name)

if echo $APPS | grep -q spi; then
  if [ "`oc get applications.argoproj.io spi -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}'`" != "Healthy Synced" ]; then
    echo Initializing SPI
    curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/${E2E_TESTS_COMMIT_SHA:-main}/scripts/spi-e2e-setup.sh | bash -s
  fi
fi

# update Pipelines as code secrets and github app
if [ -n "${GITHUB_APP_ID}" ] && [ -n "${GITHUB_APP_PRIVATE_KEY}" ]; then
  $ROOT/hack/build/setup-pac-app.sh
fi

# trigger refresh of apps
for APP in $APPS; do
  kubectl patch $APP -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}'
done

# wait for the refresh
while [ -n "$(oc get applications.argoproj.io -n openshift-gitops -o jsonpath='{range .items[*]}{@.metadata.annotations.argocd\.argoproj\.io/refresh}{end}')" ]; do
  sleep 5
done

INTERVAL=10
while :; do
  STATE=$(kubectl get apps -n openshift-gitops --no-headers)
  NOT_DONE=$(echo "$STATE" | grep -v "Synced[[:blank:]]*Healthy")
  echo "$NOT_DONE"
  if [ -z "$NOT_DONE" ]; then
     echo All Applications are synced and Healthy
     exit 0
  else
     UNKNOWN=$(echo "$NOT_DONE" | grep Unknown | grep -v Progressing | cut -f1 -d ' ')
     if [ -n "$UNKNOWN" ]; then
       for app in $UNKNOWN; do
         ERROR=$(oc get -n openshift-gitops applications.argoproj.io $app -o jsonpath='{.status.conditions}')
         if echo "$ERROR" | grep -q 'context deadline exceeded'; then
           echo Refreshing $app
           kubectl patch applications.argoproj.io $app -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "soft"}}}'
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
