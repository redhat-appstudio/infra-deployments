#!/bin/bash -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

TOOLCHAIN=$1
KEYCLOAK=$2

if [ -n "$TOOLCHAIN" ]; then
  echo "Deploying toolchain"
  "$ROOT/hack/sandbox-development-mode.sh"

  if [ -n "$KEYCLOAK" ]; then
    echo "Patching toolchain config to use keylcoak installed on the cluster"

    BASE_URL=$(oc get ingresses.config.openshift.io/cluster -o jsonpath={.spec.domain})
    RHSSO_URL="https://keycloak-dev-sso.$BASE_URL"

    oc patch ToolchainConfig/config -n toolchain-host-operator --type=merge --patch-file=/dev/stdin << EOF
spec:
  host:
    registrationService:
      auth:
        authClientConfigRaw: '{
                  "realm": "testrealm",
                  "auth-server-url": "$RHSSO_URL/auth",
                  "ssl-required": "nones",
                  "resource": "sandbox-public",
                  "clientId": "sandbox-public",
                  "public-client": true
                }'
        authClientLibraryURL: $RHSSO_URL/auth/js/keycloak.js
        authClientPublicKeysURL: $RHSSO_URL/auth/realms/testrealm/protocol/openid-connect/certs
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
oc kustomize $ROOT/argo-cd-apps/overlays/development | yq e "select(.spec.source.repoURL==\"https://github.com/redhat-appstudio/infra-deployments.git\")
  | del(.spec.destination, .spec.syncPolicy, .spec.project, .spec.source.path, .metadata.namespace)
  | .spec.source.repoURL=\"$MY_GIT_REPO_URL\"
  | .spec.source.targetRevision=\"$PREVIEW_BRANCH\"
  | .metadata.finalizers=[\"resources-finalizer.argocd.argoproj.io\"]" > $ROOT/argo-cd-apps/overlays/development/repo-overlay.yaml


# delete argoCD applications which are not in DEPLOY_ONLY env var if it's set
if [ -n "$DEPLOY_ONLY" ]; then
  APPLICATIONS=$(oc kustomize argo-cd-apps/base/ | yq e --no-doc .metadata.name)
  DELETED=$(yq e --no-doc .metadata.name $ROOT/argo-cd-apps/overlays/development/delete-applications.yaml)
  for APP in $APPLICATIONS; do
    if ! grep -q "\b$APP\b" <<< $DEPLOY_ONLY && ! grep -q "\b$APP\b" <<< $DELETED; then
      echo Disabling $APP based on DEPLOY_ONLY variable
      echo '---' >> $ROOT/argo-cd-apps/overlays/development/delete-applications.yaml
      yq e -n ".apiVersion=\"argoproj.io/v1alpha1\"
                 | .kind=\"Application\"
                 | .metadata.name = \"$APP\"
                 | .\$patch = \"delete\"" >> $ROOT/argo-cd-apps/overlays/development/delete-applications.yaml
    fi
  done
fi

# set the API server which SPI uses to authenticate users to empty string (by default) so that multi-cluster
# setup is not needed
yq -i e ".0.value=\"$SPI_API_SERVER\"" $ROOT/components/spi/overlays/staging/oauth-service-config-patch.json
# patch the SPI configuration with the Vault host configuration to provided VAULT_HOST variable or to current cluster
# and the base URL set to the SPI_BASE_URL variable or the URL of the  route to the SPI OAuth service in the current cluster
# This script also sets up the Vault client to accept insecure TLS connections so that the custom vault host doesn't have
# to serve requests using a trusted TLS certificate.
$ROOT/hack/util-patch-spi-config.sh
# configure the secrets and providers in SPI
TMP_FILE=$(mktemp)
yq e ".serviceProviders[0].type=\"${SPI_TYPE:-GitHub}\"" $ROOT/components/spi/base/config.yaml | \
    yq e ".serviceProviders[0].clientId=\"${SPI_CLIENT_ID:-app-client-id}\"" - | \
    yq e ".serviceProviders[0].clientSecret=\"${SPI_CLIENT_SECRET:-app-secret}\"" - | \
    yq e ".serviceProviders[1].type=\"${SPI_TYPE:-Quay}\"" - | \
    yq e ".serviceProviders[1].clientId=\"${SPI_CLIENT_ID:-app-client-id}\"" - | \
    yq e ".serviceProviders[1].clientSecret=\"${SPI_CLIENT_SECRET:-app-secret}\"" - > $TMP_FILE
oc create namespace spi-system --dry-run=client -o yaml | oc apply -f -
oc create -n spi-system secret generic shared-configuration-file --from-file=config.yaml=$TMP_FILE --dry-run=client -o yaml | oc apply -f -
echo "SPI configured"
rm $TMP_FILE

$ROOT/hack/util-set-github-org $MY_GITHUB_ORG

domain=$(kubectl get ingresses.config.openshift.io cluster --template={{.spec.domain}})

if [ -n "$DOCKER_IO_AUTH" ]; then
    AUTH=$(mktemp)
    # Set global pull secret
    oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > $AUTH
    oc registry login --registry=docker.io --auth-basic=$DOCKER_IO_AUTH --to=$AUTH
    oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=$AUTH
    # Set current namespace pipeline serviceaccount which is used by buildah
    oc create secret docker-registry docker-io-pull --from-file=.dockerconfigjson=$AUTH -o yaml --dry-run=client | oc apply -f-
    oc create serviceaccount pipeline -o yaml --dry-run=client | oc apply -f-
    oc secrets link pipeline docker-io-pull
    rm $AUTH
fi

rekor_server="rekor.$domain"
sed -i.bak "s/rekor-server.enterprise-contract-service.svc/$rekor_server/" $ROOT/argo-cd-apps/base/enterprise-contract.yaml && rm $ROOT/argo-cd-apps/base/enterprise-contract.yaml.bak
yq -i e ".data |= .\"transparency.url\"=\"https://$rekor_server\"" $ROOT/components/pipeline-service/tekton-chains/chains-config.yaml

[ -n "${BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/build-service\")) |=.newName=\"${BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build-service/kustomization.yaml
[ -n "${BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/build-service\")) |=.newTag=\"${BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build-service/kustomization.yaml
[[ -n "${BUILD_SERVICE_PR_OWNER}" && "${BUILD_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/build-service*\")) |= \"https://github.com/${BUILD_SERVICE_PR_OWNER}/build-service/config/default?ref=${BUILD_SERVICE_PR_SHA}\"" $ROOT/components/build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newName=\"${JVM_BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/jvm-build-service/kustomization.yaml
[ -n "${JVM_BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"hacbs-jvm-operator\")) |=.newTag=\"${JVM_BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/jvm-build-service/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_PR_OWNER}" && "${JVM_BUILD_SERVICE_PR_SHA}" ]] && sed -i -e "s|\(https://github.com/\)redhat-appstudio\(/jvm-build-service/.*?ref=\)\(.*\)|\1${JVM_BUILD_SERVICE_PR_OWNER}\2${JVM_BUILD_SERVICE_PR_SHA}|" -e "s|\(https://raw.githubusercontent.com/\)redhat-appstudio\(/jvm-build-service/\)[^/]*\(/.*\)|\1${JVM_BUILD_SERVICE_PR_OWNER}\2${JVM_BUILD_SERVICE_PR_SHA}\3|" $ROOT/components/jvm-build-service/kustomization.yaml
[[ -n "${JVM_BUILD_SERVICE_CACHE_IMAGE}" && ${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE} ]] && yq -i e "select(.[].path == \"/spec/template/spec/containers/0/env\") | .[].value |= . + [{\"name\" : \"JVM_BUILD_SERVICE_CACHE_IMAGE\", \"value\": \"${JVM_BUILD_SERVICE_CACHE_IMAGE}\"}, {\"name\": \"JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE\", \"value\": \"${JVM_BUILD_SERVICE_REQPROCESSOR_IMAGE}\"}] | (.[].value[] | select(.name == \"IMAGE_TAG\")) |= .value = \"\"" $ROOT/components/jvm-build-service/operator_env_patch.yaml
[ -n "${DEFAULT_BUILD_BUNDLE}" ] && yq -i e "(.configMapGenerator[].literals[] | select(. == \"default_build_bundle*\")) |= \"default_build_bundle=${DEFAULT_BUILD_BUNDLE}\"" $ROOT/components/build-templates/kustomization.yaml
[ -n "${HACBS_BUILD_BUNDLE}" ] && yq -i e "(.configMapGenerator[].literals[] | select(. == \"hacbs_build_bundle*\")) |= \"hacbs_build_bundle=${HACBS_BUILD_BUNDLE}\"" $ROOT/components/build-templates/kustomization.yaml
# for jvm-build-service, since its service registry tests produces a ton of pipelineruns, and even the simple ones can produced more than 10,
# we bump the keep setting to better allow for debug in jvm-build-service PRs
[[ -n "${JVM_BUILD_SERVICE_PR_OWNER}" && ${JVM_BUILD_SERVICE_PR_SHA} ]] && yq -i e "(.spec.pruner.keep = 1000)" $ROOT/components/pipeline-service/openshift-pipelines/config.yaml

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
yq e ".spec.source.path=\"argo-cd-apps/overlays/development\"
    | .spec.source.repoURL=\"$MY_GIT_REPO_URL\"
    | .spec.source.targetRevision=\"$PREVIEW_BRANCH\"" \
  $ROOT/argo-cd-apps/app-of-apps/all-applications-staging.yaml | oc apply -f -

while [ "$(oc get applications.argoproj.io all-components-staging -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; do
  echo Waiting for sync of all-components-staging argoCD app
  sleep 5
done

APPS=$(kubectl get apps -n openshift-gitops -o name)

if echo $APPS | grep -q spi; then
  if [ "`oc get applications.argoproj.io spi -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}'`" != "Healthy Synced" ]; then
    echo Initializing SPI
    curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/${E2E_TESTS_COMMIT_SHA:-main}/scripts/spi-e2e-setup.sh | VAULT_PODNAME='vault-0' VAULT_NAMESPACE='spi-vault' bash -s
    SPI_APP_ROLE_FILE=$ROOT/.tmp/approle_secret.yaml
    if [ -f "$SPI_APP_ROLE_FILE" ]; then
        echo "$SPI_APP_ROLE_FILE exists."
        kubectl apply -f $SPI_APP_ROLE_FILE  -n spi-system
    fi
    echo "Vault init complete"

  fi
fi

# Configure Pipelines as Code and required credentials
$ROOT/hack/build/setup-pac-integration.sh

# trigger refresh of apps
for APP in $APPS; do
  kubectl patch $APP -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}' &
done
wait

# wait for the refresh
while [ -n "$(oc get applications.argoproj.io -n openshift-gitops -o jsonpath='{range .items[*]}{@.metadata.annotations.argocd\.argoproj\.io/refresh}{end}')" ]; do
  sleep 5
done

INTERVAL=10
while :; do
  STATE=$(kubectl get apps -n openshift-gitops --no-headers)
  NOT_DONE=$(echo "$STATE" | grep -v "Synced[[:blank:]]*Healthy" || true)
  echo "$NOT_DONE"
  if [ -z "$NOT_DONE" ]; then
     echo All Applications are synced and Healthy
     break
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

# Wait for all tekton components to be installed
# The status of a tektonconfig CR should be "type: Ready, status: True" once the install is completed
# More info: https://tekton.dev/docs/operator/tektonconfig/#tekton-config
while :; do
  STATE=$(oc get tektonconfig config -o json | jq -r '.status.conditions[] | select(.type == "Ready")')
  [ "$(jq -r '.status' <<< "$STATE")" == "True" ] && echo All required tekton resources are installed and ready && break
  echo Some tekton resources are not ready yet:
  jq -r '.message' <<< "$STATE"
  sleep $INTERVAL
done


if [ -n "$KEYCLOAK" ] && [ -n "$TOOLCHAIN" ]; then
  echo "Restarting toolchain registration service to pick up keycloak's certs."
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
