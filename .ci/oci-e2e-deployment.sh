# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

export WORKSPACE=$(dirname $(dirname $(readlink -f "$0")));
export APPLICATION_NAMESPACE="openshift-gitops"
export APPLICATION_NAME="gitops"

#Stop execution on any error
trap "catchFinish" EXIT SIGINT

function catchFinish() {
    JOB_EXIT_CODE=$?
    if [[ "$JOB_EXIT_CODE" != "0" ]]; then
        echo "[ERROR] Job failed with code ${JOB_EXIT_CODE}."
    else
        echo "[INFO] Job completed successfully."
    fi
    /bin/bash "$WORKSPACE"/hack/destroy-cluster.sh

    exit $JOB_EXIT_CODE
}

function checkApplicationHealth() {
    while [ "$(kubectl get application ${APPLICATION_NAME} -n ${APPLICATION_NAMESPACE} -o jsonpath='{.status.health.status}')" != "Healthy" ]; do
        sleep 3
        echo "[INFO] Waiting for AppStudio to be ready."
    done
}

command -v yq >/dev/null 2>&1 || { echo "yq is not installed. Aborting."; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is not installed. Aborting."; exit 1; }

/bin/bash "$WORKSPACE"/hack/bootstrap-cluster.sh

export -f checkApplicationHealth
timeout --foreground 10m bash -c checkApplicationHealth
###