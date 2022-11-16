# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

export WORKSPACE=$(dirname $(dirname $(readlink -f "$0")));

export TEST_BRANCH_ID=$(date +%s)
export MY_GIT_FORK_REMOTE="qe"
export MY_GITHUB_ORG="redhat-appstudio-qe"
export MY_GITHUB_TOKEN="${GITHUB_TOKEN}"
export E2E_APPLICATIONS_NAMESPACE=appstudio-e2e-test
export SHARED_SECRET_NAMESPACE="build-templates"

# REPO_OWNER/REPO_NAME/PULL_NUMBER are not defined in openshift-ci periodics jobs
export REPO_OWNER=${REPO_OWNER:-"redhat-appstudio"}
export REPO_NAME=${REPO_NAME:-"infra-deployments"}
export PULL_NUMBER=${PULL_NUMBER:-"periodic"}

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
    echo -e "[INFO] Creating application-service related secrets in $SHARED_SECRET_NAMESPACE namespace"

    echo "$QUAY_TOKEN" | base64 --decode > docker.config
    kubectl create secret docker-registry redhat-appstudio-user-workload -n $SHARED_SECRET_NAMESPACE --from-file=.dockerconfigjson=docker.config || true
    rm docker.config
}

function executeE2ETests() {
    # E2E instructions can be found: https://github.com/redhat-appstudio/e2e-tests
    # The e2e binary is included in Openshift CI test container from the dockerfile: https://github.com/redhat-appstudio/infra-deployments/blob/main/.ci/openshift-ci/Dockerfile
    curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/pre-kcp/scripts/e2e-openshift-ci.sh | bash -s

    # The bin will be installed in tmp folder after executing e2e-openshift-ci.sh script
    ${WORKSPACE}/tmp/e2e-tests/bin/e2e-appstudio --ginkgo.timeout=90m --ginkgo.junit-report="${ARTIFACTS_DIR}"/e2e-report.xml -webhookConfigPath="./webhookConfig.yml" -config-suites="${WORKSPACE}/tmp/e2e-tests/tests/e2e-demos/config/default.yaml"
}

function prepareWebhookVariables() {
    #Export variables
    export webhook_salt=123456789
    export webhook_target=https://smee.io/JgVqn2oYFPY1CF
    export webhook_repositoryURL=https://github.com/$REPO_OWNER/$REPO_NAME
    export webhook_repositoryFullName=$REPO_OWNER/$REPO_NAME
    export webhook_pullNumber=$PULL_NUMBER
    # Rewrite variables in webhookConfig.yml
    curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/pre-kcp/webhookConfig.yml | envsubst > webhookConfig.yml
}

installCITools

git remote add ${MY_GIT_FORK_REMOTE} https://github.com/redhat-appstudio-qe/infra-deployments.git

# Initiate openshift ci users
export KUBECONFIG_TEST="/tmp/kubeconfig"
curl https://raw.githubusercontent.com/redhat-appstudio/e2e-tests/pre-kcp/scripts/provision-openshift-user.sh | bash -s
export KUBECONFIG="${KUBECONFIG_TEST}"

timeout --foreground 15m "$WORKSPACE"/hack/bootstrap-cluster.sh preview

prepareWebhookVariables
createQuayPullSecrets
executeE2ETests
