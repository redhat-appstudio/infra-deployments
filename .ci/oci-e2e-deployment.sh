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

command -v yq >/dev/null 2>&1 || { echo "yq is not installed. Aborting."; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is not installed. Aborting."; exit 1; }

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

function provideOpenshiftUser() {
   htpasswd -c -B -b htpasswd test-user test-user
   oc create secret generic htpass-secret --from-file=htpasswd=./htpasswd -n openshift-config
   oc create -f - -o jsonpath='{.metadata.name}' <<EOF
apiVersion: config.openshift.io/v1
kind: OAuth
metadata:
  name: cluster
spec:
  identityProviders:
  - name: htpasswd
    mappingMethod: claim 
    type: HTPasswd
    htpasswd:
      fileData:
        name: htpass-secret
EOF
    oc adm policy add-cluster-role-to-user cluster-admin test-user

    echo -e "[INFO] Waiting for htpasswd auth to be working up to 5 minutes"
    CURRENT_TIME=$(date +%s)
    ENDTIME=$(($CURRENT_TIME + 300))
    while [ $(date +%s) -lt $ENDTIME ]; do
        if oc login -u user -p user --insecure-skip-tls-verify=false; then
            break
        fi
        sleep 10
    done
}

function downloadE2ERepo() {
    git clone https://github.com/redhat-appstudio/e2e-tests.git
}

# Run the script: https://github.com/redhat-appstudio/e2e-tests/blob/main/scripts/run-tests-in-k8s.sh
function runE2ETestsFramework() {
   /bin/bash ./e2e-tests/scripts/run-tests-in-k8s.sh
}

/bin/bash "$WORKSPACE"/hack/bootstrap-cluster.sh
provideOpenshiftUser
downloadE2ERepo

# Execute the e2e tests
export -f runE2ETestsFramework
timeout --foreground 10m bash -c runE2ETestsFramework
