#!/bin/bash -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"/..

main() {
    local mode keycloak toolchain
    while [[ $# -gt 0 ]]; do
        key=$1
        case $key in
        --toolchain | -t)
            toolchain="--toolchain"
            shift
            ;;
        --keycloak | -kc)
            keycloak="--keycloak"
            shift
            ;;
        preview | upstream)
            mode=$1
            shift
            ;;
        -h | --help)
            print_help
            exit 0
            ;;
        *)
            shift
            ;;
        esac
    done

    "${ROOT}/hack/deploy-argocd.sh"
    "${ROOT}/hack/bootstrap-host-cluster.sh"
    "${ROOT}/hack/bootstrap-member-cluster.sh"
    "${ROOT}/hack/bootstrap-cluster-common.sh"

    echo "Setting Cluster Mode: ${MODE:-Upstream}"
    case $mode in
    "" | "upstream")
        kubectl create namespace argocd-staging
        kubectl create -k $ROOT/argo-cd-apps/app-of-app-sets/staging
        # Check if we have a tekton-chains namespace, and if so, remove any explicit transparency.url setting
        # which might be left from running this script with the 'preview' flag to enable the cluster local
        # rekor instance. By default, chains will use the publicly accessible sandbox instance hosted by sigstore
        # of rekor
        # If, in the future, we're boostrapping to use a Red Hat internal rekor instance instead of the public,
        # default sandbox instance hosted by sigstore we would want to reconsider this approach.
        if kubectl get namespace tekton-chains &>/dev/null; then
            # Remove our transparency.url, if present, to ensure we're not using the cluster local rekor
            # which is only available in 'preview' mode.
            kubectl patch configmap/chains-config -n tekton-chains --type=json --patch '[{"op":"remove","path":"/data/transparency.url"}]'
            kubectl delete pod -n tekton-chains -l app=tekton-chains-controller
        fi
        ;;
    "preview")
        $ROOT/hack/preview.sh $toolchain $keycloak
        ;;
    esac
}

print_help() {
    echo "Usae: $0 MODE [-t|--toolchain] [-kc|--keycloak] [-h|--help]"
    echo "  MODE             upstream/preview (default: upstream)"
    echo "  -t, --toolchain  (only in preview mode) Install toolchain operators"
    echo "  -kc, --keycloak  (only in preview mode) Configure the toolchain operator to use keycloak deployed on the cluster"
    echo "  -h, --help       Show this help message and exit"
    echo
    echo "Example usage: \`$0 preview --toolchain --keycloak"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
