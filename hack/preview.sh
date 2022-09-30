#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

source ${ROOT}/hack/flags.sh "The preview.sh enable preview mode used for development and testing on non-production clusters / kcp instances."
MODE=${MODE:-preview} parse_flags $@

if [ -z "$MY_GIT_FORK_REMOTE" ]; then
    echo "Set MY_GIT_FORK_REMOTE environment to name of your fork remote"
    exit 1
fi

if [ -z "${ROOT_WORKSPACE}" ]; then
    echo "Set ROOT_WORKSPACE environment variable or include to hack/preview.env"
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

if [ -n "$PIPELINE_SERVICE_IDENTITY_HASH" ]; then
  IDENTITY_HASHES="pipeline-service:$PIPELINE_SERVICE_IDENTITY_HASH"
else
  KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}
  if KUBECONFIG=${KCP_KUBECONFIG} kubectl ws pipeline-service; then
    IDENTITY_HASH=$(oc get apiexport kubernetes --kubeconfig ${KCP_KUBECONFIG} -o jsonpath='{.status.identityHash}')
    IDENTITY_HASHES="pipeline-service:$IDENTITY_HASH"
  else
    echo "pipeline-service workspace is not available"
    echo "Set existing pipeline-service workspace by PIPELINE_SERVICE_SP_WORKSPACE and PIPELINE_SERVICE_IDENTITY_HASH"
    echo "Or install pipeline-service instance using './hack/install-pipeline-service.sh'"
    exit 1
  fi
fi

# Ensure that we are in redhat-appstudio workspace
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}:redhat-appstudio

# Create preview branch for preview configuration
PREVIEW_BRANCH=preview-${MY_GIT_BRANCH}${TEST_BRANCH_ID+-$TEST_BRANCH_ID}
if git rev-parse --verify $PREVIEW_BRANCH; then
    git branch -D $PREVIEW_BRANCH
fi
git checkout -b $PREVIEW_BRANCH

# reset the default repos in the development directory to be the current git repo
# this needs to be pushed to your fork to be seen by argocd
$ROOT/hack/util-set-development-repos.sh $MY_GIT_REPO_URL development $PREVIEW_BRANCH

if [ -n "$MY_GITHUB_ORG" ]; then
    $ROOT/hack/util-set-github-org $MY_GITHUB_ORG
fi

echo "start spi config"
CLUSTER_URL_HOST=$(oc whoami --kubeconfig ${CLUSTER_KUBECONFIG} --show-console | sed 's|https://console-openshift-console.apps.||')
if ! oc get namespace spi-system --kubeconfig ${KCP_KUBECONFIG} &>/dev/null; then
  oc create namespace spi-system --kubeconfig ${KCP_KUBECONFIG}
  oc create route edge -n spi-system --service spi-oauth-service --port 8000 spi-oauth --kubeconfig ${KCP_KUBECONFIG}
fi
export SPI_BASE_URL=https://$(kubectl --kubeconfig ${KCP_KUBECONFIG} get route/spi-oauth -n spi-system -o jsonpath='{.status.ingress[0].host}')
VAULT_HOST="https://vault-spi-vault.apps.${CLUSTER_URL_HOST}"
$ROOT/hack/util-patch-spi-config.sh $VAULT_HOST $SPI_BASE_URL "true"
# configure the secrets and providers in SPI
TMP_FILE=$(mktemp)
yq e ".serviceProviders[0].type=\"${SPI_TYPE:-GitHub}\"" $ROOT/components/spi/base/config.yaml | \
    yq e ".serviceProviders[0].clientId=\"${SPI_CLIENT_ID:-app-client-id}\"" - | \
    yq e ".serviceProviders[0].clientSecret=\"${SPI_CLIENT_SECRET:-app-secret}\"" - > $TMP_FILE
oc --kubeconfig ${KCP_KUBECONFIG}  create -n spi-system secret generic shared-configuration-file --from-file=config.yaml=$TMP_FILE --dry-run=client -o yaml | oc  --kubeconfig ${KCP_KUBECONFIG}  apply -f -
rm $TMP_FILE
echo "SPI configured"


[ -n "${HAS_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newName=\"${HAS_IMAGE_REPO}\"" $ROOT/components/application-service/kustomization.yaml
[ -n "${HAS_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newTag=\"${HAS_IMAGE_TAG}\"" $ROOT/components/application-service/kustomization.yaml
[[ -n "${HAS_PR_OWNER}" && "${HAS_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/application-service*\")) |= \"https://github.com/${HAS_PR_OWNER}/application-service/config/default?ref=${HAS_PR_SHA}\"" $ROOT/components/application-service/kustomization.yaml
[ -n "${HAS_DEFAULT_IMAGE_REPOSITORY}" ] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name ==\"IMAGE_REPOSITORY\").value) |= \"${HAS_DEFAULT_IMAGE_REPOSITORY}\"" $ROOT/components/application-service/base/manager_resources_patch.yaml

# evaluate APIExports pointers
evaluate_apiexports() {
  APIEXPORT_FILES=$1
  NEW_IDENTITY_HASHES=$(oc get apiexports.apis.kcp.dev -o jsonpath='{range .items[*]}{@.metadata.name}:{@.status.identityHash}{"\n"}{end}' --kubeconfig ${KCP_KUBECONFIG})
  IDENTITY_HASHES=$(echo -e "$IDENTITY_HASHES\n$NEW_IDENTITY_HASHES")
  for APIEXPORTFILE in $APIEXPORT_FILES; do
    REQUESTS=$(yq e '.spec.permissionClaims.[] | select(.identityHash).identityHash' $APIEXPORTFILE | sort -u)
    for REQUEST in $REQUESTS; do
      IDENTITY_HASH=$(echo "$IDENTITY_HASHES" | grep "^$REQUEST:" | cut -f2 -d":")
      if [ -z "$IDENTITY_HASH" ]; then
        # find APIExport
        for APIEXPORTFILE2 in $APIEXPORT_FILES; do
          if [ "$(yq '.metadata.name' $APIEXPORTFILE2)" == "$REQUEST" ]; then
            oc apply -f $APIEXPORTFILE2 --kubeconfig ${KCP_KUBECONFIG}
            IDENTITY_HASH=$(oc get -f $APIEXPORTFILE2 --kubeconfig ${KCP_KUBECONFIG} -o jsonpath='{.status.identityHash}')
            IDENTITY_HASHES=$(echo -e "$IDENTITY_HASHES\n$REQUEST:$IDENTITY_HASH")
            break
          fi
        done
      fi
      if [ -z "$IDENTITY_HASH" ]; then
        echo Unknown IDENTITY_HASH for $REQUEST
      else
        sed -i "s/\b$REQUEST\b/$IDENTITY_HASH/g" $APIEXPORTFILE
      fi
    done
  done
}
APIEXPORTS=$(find -name '*apiexport*.yaml' | grep overlays/dev)
evaluate_apiexports "$(echo $APIEXPORTS | grep -v '/hacbs/')"
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}:redhat-hacbs
evaluate_apiexports "$(echo $APIEXPORTS | grep '/hacbs/')"
KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}:redhat-appstudio

if ! git diff --exit-code --quiet; then
    git commit -a -m "Preview mode, do not merge into main"
    git push -f --set-upstream $MY_GIT_FORK_REMOTE $PREVIEW_BRANCH
fi

git checkout $MY_GIT_BRANCH

#set the local cluster to point to the current git repo and branch and update the path to development
$ROOT/hack/util-update-app-of-apps.sh $MY_GIT_REPO_URL development $PREVIEW_BRANCH

while [ "$(oc get --kubeconfig ${CLUSTER_KUBECONFIG} applications.argoproj.io all-components -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}')" != "Healthy Synced" ]; do
  sleep 5
done

APPS=$(kubectl get --kubeconfig ${CLUSTER_KUBECONFIG} apps -n openshift-gitops -o name)

if echo $APPS | grep -q spi-vault; then
  if [ "`oc get --kubeconfig ${CLUSTER_KUBECONFIG} applications.argoproj.io spi-vault -n openshift-gitops -o jsonpath='{.status.health.status} {.status.sync.status}'`" != "Healthy Synced" ]; then
    echo "Initializing Vault"
    export VAULT_KUBE_CONFIG=${CLUSTER_KUBECONFIG}
    export VAULT_NAMESPACE=spi-vault
    bash <(curl -s https://raw.githubusercontent.com/redhat-appstudio/service-provider-integration-operator/59ec1d2765769a5447c05be46ca4f082879f51b1/hack/vault-init.sh)
    SPI_APP_ROLE_FILE=.tmp/approle_secret.yaml
    if [ -f "$SPI_APP_ROLE_FILE" ]; then
        echo "$SPI_APP_ROLE_FILE exists."
        kubectl apply -f $SPI_APP_ROLE_FILE  -n spi-system  --kubeconfig ${KCP_KUBECONFIG}
    fi
    echo "Vault init complete"
    echo "========================================================================="
  fi
fi
echo

# trigger refresh of apps
for APP in $APPS; do
  kubectl patch --kubeconfig ${CLUSTER_KUBECONFIG} $APP -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}'
done

# wait for the refresh
while [ -n "$(oc get --kubeconfig ${CLUSTER_KUBECONFIG} applications.argoproj.io -n openshift-gitops -o jsonpath='{range .items[*]}{@.metadata.annotations.argocd\.argoproj\.io/refresh}{end}')" ]; do
  sleep 5
done

# Create customers workspace with bindings
${ROOT}/hack/create-user-workspace.sh appstudio
${ROOT}/hack/create-user-workspace.sh hacbs

KUBECONFIG=${KCP_KUBECONFIG} kubectl ws ${ROOT_WORKSPACE}

INTERVAL=10
# Disabling check of healthy apps for now till envineronment is more stable
while false; do
  STATE=$(kubectl get --kubeconfig ${CLUSTER_KUBECONFIG} apps -n openshift-gitops --no-headers)
  NOT_DONE=$(echo "$STATE" | grep -v "Synced[[:blank:]]*Healthy")
  echo "$NOT_DONE"
  if [ -z "$NOT_DONE" ]; then
     echo All Applications are synced and Healthy
     exit 0
  else
     UNKNOWN=$(echo "$NOT_DONE" | grep Unknown | grep -v Progressing | cut -f1 -d ' ')
     if [ -n "$UNKNOWN" ]; then
       for app in $UNKNOWN; do
         ERROR=$(oc get --kubeconfig ${CLUSTER_KUBECONFIG} -n openshift-gitops applications.argoproj.io $app -o jsonpath='{.status.conditions}')
         if echo "$ERROR" | grep -q 'context deadline exceeded'; then
           echo Refreshing $app
           kubectl patch --kubeconfig ${CLUSTER_KUBECONFIG} applications.argoproj.io $app -n openshift-gitops --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "soft"}}}'
           while [ -n "$(oc get --kubeconfig ${CLUSTER_KUBECONFIG} applications.argoproj.io -n openshift-gitops $app -o jsonpath='{.metadata.annotations.argocd\.argoproj\.io/refresh}')" ]; do
             sleep 5
           done
           echo Refresh of $app done
           continue 2
         fi
         echo $app failed with:
         if [ -n "$ERROR" ]; then
           echo "$ERROR"
         else
           oc get --kubeconfig ${CLUSTER_KUBECONFIG} -n openshift-gitops applications.argoproj.io $app -o yaml
         fi
       done
       exit 1
     fi
     echo Waiting $INTERVAL seconds for application sync
     sleep $INTERVAL
  fi
done
