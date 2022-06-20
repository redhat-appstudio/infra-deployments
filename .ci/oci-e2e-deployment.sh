# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

export WORKSPACE=$(dirname $(dirname $(readlink -f "$0")));
export APPLICATION_NAMESPACE="openshift-gitops"
export APPLICATION_NAME="all-components-staging"

export TEST_BRANCH_ID=$(date +%s)
export MY_GIT_FORK_REMOTE="qe"
export MY_GITHUB_ORG="redhat-appstudio-qe"
export MY_GITHUB_TOKEN="${GITHUB_TOKEN}"
export E2E_APPLICATIONS_NAMESPACE=appstudio-e2e-test

# Environment variable used to override the default "protected" image repository in HAS
# https://github.com/redhat-appstudio/application-service/blob/6b9d21b8f835263b2e92f1e9343a1453caa2e561/gitops/generate_build.go#L50
# Users are allowed to push images to this repo only in case the image contains a tag that consists of "<USER'S_NAMESPACE_NAME>-<CUSTOM-TAG>"
# For example: "quay.io/redhat-appstudio-qe/test-images-protected:appstudio-e2e-test-mytag123"
export HAS_DEFAULT_IMAGE_REPOSITORY="quay.io/redhat-appstudio-qe/test-images-protected"


export PATH=$PATH:/tmp/bin
mkdir -p /tmp/bin

# Available openshift ci environments https://docs.ci.openshift.org/docs/architecture/step-registry/#available-environment-variables
export ARTIFACTS_DIR=${ARTIFACT_DIR:-"/tmp/appstudio"}

if [[ -z "${GITHUB_TOKEN}" ]]; then
  echo - e "[ERROR] GITHUB_TOKEN env is not set. Aborting."
fi

if [[ -z "${QUAY_TOKEN}" ]]; then
  echo - e "[ERROR] QUAY_TOKEN env is not set. Aborting."
fi

#Stop execution on any error
trap "catchFinish" EXIT SIGINT

# Don't remove appstudio. Can broke development cluster
function catchFinish() {
    JOB_EXIT_CODE=$?
    if [[ "$JOB_EXIT_CODE" != "0" ]]; then
        echo "[ERROR] Job failed with code ${JOB_EXIT_CODE}."
    else
        echo "[INFO] Job completed successfully."
    fi

    MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    git push $MY_GIT_FORK_REMOTE --delete preview-${MY_GIT_BRANCH}-${TEST_BRANCH_ID}

    exit $JOB_EXIT_CODE
}

function installCITools() {
    curl -H "Authorization: token $GITHUB_TOKEN" -LO https://github.com/mikefarah/yq/releases/download/v4.20.2/yq_linux_amd64 && \
    chmod +x ./yq_linux_amd64 && \
    mv ./yq_linux_amd64 /tmp/bin/yq && \
    yq --version
}

# Secrets used by pipelines to push component containers to quay.io
function createQuayPullSecrets() {
    echo "$QUAY_TOKEN" | base64 --decode > docker.config
    oc create namespace $E2E_APPLICATIONS_NAMESPACE --dry-run=client -o yaml | oc apply -f -
    kubectl create secret docker-registry redhat-appstudio-registry-pull-secret -n  $E2E_APPLICATIONS_NAMESPACE --from-file=.dockerconfigjson=docker.config
    kubectl create secret docker-registry redhat-appstudio-staginguser-pull-secret -n  $E2E_APPLICATIONS_NAMESPACE --from-file=.dockerconfigjson=docker.config
    rm docker.config
}

function waitAppStudioToBeReady() {
    while [ "$(kubectl get applications.argoproj.io ${APPLICATION_NAME} -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.health.status}')" != "Healthy" ] ||
          [ "$(kubectl get applications.argoproj.io ${APPLICATION_NAME} -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.sync.status}')" != "Synced" ]; do
        sleep 1m
        echo "[INFO] Waiting for AppStudio to be ready."
    done
}

function waitBuildToBeReady() {
    while [ "$(kubectl get applications.argoproj.io build -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.health.status}')" != "Healthy" ] ||
          [ "$(kubectl get applications.argoproj.io build -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.sync.status}')" != "Synced" ]; do
        sleep 1m
        echo "[INFO] Waiting for Build to be ready."
    done
}

function waitSPIToBeReady() {
    while [ "$(kubectl get applications.argoproj.io spi -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.health.status}')" != "Healthy" ] ||
          [ "$(kubectl get applications.argoproj.io spi -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.sync.status}')" != "Synced" ]; do
        sleep 1m
        echo "[INFO] Waiting for spi to be ready."
    done
}

function checkHASGithubOrg() {
    while [[ "$(kubectl get configmap application-service-github-config -n application-service -o jsonpath='{.data.GITHUB_ORG}')" != "${MY_GITHUB_ORG}" ]]; do
        sleep 3m
        echo "[INFO] Waiting for HAS to be ready."
    done
}

function executeE2ETests() {
    # E2E instructions can be found: https://github.com/redhat-appstudio/e2e-tests
    # The e2e binary is included in Openshift CI test container from the dockerfile: https://github.com/redhat-appstudio/infra-deployments/blob/main/.ci/openshift-ci/Dockerfile
    curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/main/scripts/e2e-openshift-ci.sh | bash -s

    # The bin will be installed in tmp folder after executing e2e-openshift-ci.sh script
    ${WORKSPACE}/tmp/e2e-tests/bin/e2e-appstudio  --ginkgo.junit-report="${ARTIFACTS_DIR}"/e2e-report.xml -webhookConfigPath="./webhookConfig.yml" -config-suites="${WORKSPACE}/tmp/e2e-tests/tests/e2e-demos/config/default.yaml"
}

function prepareWebhookVariables() {
    #Export variables
    export webhook_salt=123456789
    export webhook_target=https://smee.io/JgVqn2oYFPY1CF
    export webhook_repositoryURL=https://github.com/redhat-appstudio/infra-deployments
    export webhook_repositoryFullName=redhat-appstudio/infra-deployments
    # Rewrite variables in webhookConfig.yml
    curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/main/webhookConfig.yml | envsubst > webhookConfig.yml
}

installCITools
createQuayPullSecrets

git remote add ${MY_GIT_FORK_REMOTE} https://github.com/redhat-appstudio-qe/infra-deployments.git

# Initiate openshift ci users
export KUBECONFIG_TEST="/tmp/kubeconfig"
curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/main/scripts/provision-openshift-user.sh | bash -s
export KUBECONFIG="${KUBECONFIG_TEST}"

/bin/bash "$WORKSPACE"/hack/bootstrap-cluster.sh preview
curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/main/scripts/spi-e2e-setup.sh | bash -s

export -f waitAppStudioToBeReady
export -f waitBuildToBeReady
export -f checkHASGithubOrg
export -f waitSPIToBeReady

timeout --foreground 10m bash -c waitAppStudioToBeReady
timeout --foreground 10m bash -c waitBuildToBeReady
timeout --foreground 3m bash -c checkHASGithubOrg
timeout --foreground 10m bash -c waitSPIToBeReady
prepareWebhookVariables
executeE2ETests
