#!/bin/bash

# Display help information about this script bash
function helpUsage() {
    echo -e "Deploy HAC and connect it to existing Stonesoup cluster\n"
    echo 
    echo -e "This script requires you to have access to HAC ephemeral cluster (c-rh-c-eph)."
    echo -e "Options:"
    echo -e "   -h,  --help                   Print this help message."
    echo -e "   -ehk, --eph-hac-kubeconfig    A valid kubeconfig pointing to HAC Ephemeral cluster"
    echo -e "   -sk, --stonesoup-kubeconfig   A valid kubeconfig pointing to a cluster where Stonesoup controllers are installed."
    echo
    echo -e "This command uses internal app-interface endpoint https://app-interface.apps.appsrep05ue1.zqxk.p1.openshiftapps.com/graphql (VPN required)"
    echo -e "In order to use this without VPN, env vars QONTRACT_BASE_URL, QONTRACT_USERNAME and QONTRACT_PASSWORD need to be set."
}

while [[ $# -gt 0 ]]
do
    case "$1" in
        -h|--help)
            helpUsage
            exit 0
            ;;
        -ehk|--eph-hac-kubeconfig)
            export HAC_KUBECONFIG=$2
            ;;
        -sk|--stonesoup-kubeconfig)
            export STONESOUP_KUBECONFIG=$2
            ;;
        *)
            ;;
    esac
    shift
done

if [[ -z "$QONTRACT_BASE_URL" ]]; then
    echo "[INFO] QONTRACT_BASE_URL env variable was not provided. Using default endpoint (RH VPN required)"
    if ! curl --connect-timeout 3 https://app-interface.apps.appsrep05ue1.zqxk.p1.openshiftapps.com/graphql; then
        echo "[ERROR] QONTRACT_BASE_URL was not provided and default app-interface endpint cannot be reached (Are you on VPN?)."
        helpUsage & exit 1
    fi
else
    if [[ -z "$QONTRACT_USERNAME" || -z "$QONTRACT_PASSWORD" ]]; then 
        echo "[ERROR] QONTRACT_USERNAME and QONTRACT_PASSWORD needs to be set when QONTRACT_BASE_URL is provided."
        helpUsage & exit 1
    fi
fi

if [[ -z "$HAC_KUBECONFIG" ]]; then
    echo "[ERROR] Ephemeral HAC cluster kubeconfig not defined. Please use flag '-ehk' or '--eph-hac-kubeconfig' to define the ephemeral hac cluster kubeconfig." 
    helpUsage & exit 1
fi

if [[ -z "$STONESOUP_KUBECONFIG" ]]; then
    echo "[ERROR] stonesoup cluster kubeconfig not defined. Please use flag '-sk' or '--stonesoup-kubeconfig' to define the stonestoup cluster kubeconfig." 
    helpUsage & exit 1
fi

installBonfire(){
    echo "Installing bonfire."
    VENV_DIR=$(mktemp -d)
    python3 -m venv "$VENV_DIR"
    . "$VENV_DIR"/bin/activate
    pip install 'crc-bonfire>=4.18.0'
}

reserveNamespace() {
    echo "Reserving namespace."
    NAMESPACE=$(KUBECONFIG=$HAC_KUBECONFIG bonfire namespace reserve -f)
}

installHac() {
    # Only deploy necessary frontend dependencies
    export BONFIRE_FRONTEND_DEPENDENCIES=chrome-service,insights-chrome

    echo "Installing HAC on Ephemeral cluster"
    KUBECONFIG=$HAC_KUBECONFIG bonfire deploy hac --frontends true --source=appsre --clowd-env env-"${NAMESPACE}" --namespace="$NAMESPACE"
}

patchfeenv() {
    KEYCLOAK_ENDPOINT=https://$(oc get route/keycloak --kubeconfig="$STONESOUP_KUBECONFIG" -n dev-sso -o jsonpath="{.spec.host}")/auth
    oc patch feenv/env-"$NAMESPACE" --kubeconfig="$HAC_KUBECONFIG" --type=merge --patch-file=/dev/stdin << EOF
    spec:
        sso: $KEYCLOAK_ENDPOINT
EOF
}

deployProxy() {
    STONESOUP_API_ENDPOINT=https://$(oc get route/api --kubeconfig="$STONESOUP_KUBECONFIG" -n toolchain-host-operator  -o jsonpath="{.spec.host}")
    oc process --kubeconfig="$HAC_KUBECONFIG" -f https://raw.githubusercontent.com/openshift/hac-dev/main/tmp/hac-proxy.yaml -n "$NAMESPACE" -p NAMESPACE="$NAMESPACE" -p ENV_NAME=env-"$NAMESPACE" -p HOSTNAME=$(oc get --kubeconfig="$HAC_KUBECONFIG" feenv env-"$NAMESPACE" -o=jsonpath='{.spec.hostname}') | oc create --kubeconfig="$HAC_KUBECONFIG" -f -
    oc set env Deployment/hac-proxy --kubeconfig="$HAC_KUBECONFIG" -n "$NAMESPACE" HJ_K8S="$STONESOUP_API_ENDPOINT" HJ_PROXY_SSL=false
}

installBonfire
reserveNamespace
patchfeenv
deployProxy
installHac

echo "Eph cluster namespace: $NAMESPACE"
echo "Stonesoup URL: https://$(oc get feenv env-$NAMESPACE --kubeconfig="$HAC_KUBECONFIG" -o jsonpath="{.spec.hostname}")/hac/stonesoup"
