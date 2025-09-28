#!/bin/bash -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"/..

main() {
    local mode obo eaas
    while [[ $# -gt 0 ]]; do
        key=$1
        case $key in
        --obo | -o)
            obo="--obo"
            shift
            ;;
        --eaas | -e)
            eaas="--eaas"
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

    echo "Setting Cluster Mode: ${mode:-Upstream}"
    case $mode in
    "" | "upstream")
        kubectl create -k $ROOT/argo-cd-apps/app-of-app-sets/staging
        # Remove any explicit transparency.url setting from chains-config which might be left from running
        # this script with the 'preview' flag to enable the cluster local
        # rekor instance. By default, chains will use the publicly accessible sandbox instance hosted by sigstore
        # of rekor
        # If, in the future, we're boostrapping to use a Red Hat internal rekor instance instead of the public,
        # default sandbox instance hosted by sigstore we would want to reconsider this approach.
        if kubectl get namespace openshift-pipelines &>/dev/null; then
            if kubectl get configmap chains-config -n openshift-pipelines &>/dev/null; then
                # Remove our transparency.url, if present, to ensure we're not using the cluster local rekor
                # which is only available in 'preview' mode.
                kubectl patch configmap/chains-config -n openshift-pipelines --type=json --patch '[{"op":"remove","path":"/data/transparency.url"}]'
                kubectl delete pod -n openshift-pipelines -l app=tekton-chains-controller
            fi
        fi
        ;;
    "preview")
        $ROOT/hack/preview.sh $obo $eaas
        ;;
    esac

    # OIDC secrets must be deployed after the MCE operator creates the local-cluster namespace
    if [ ! -z "$eaas" ]; then
        "${ROOT}/hack/bootstrap-eaas-cluster.sh"
    fi
}

print_help() {
    echo "Usae: $0 MODE [-o|--obo] [-e|--eaas] [-h|--help]"
    echo "  MODE             upstream/preview (default: upstream)"
    echo "  -o, --obo        (only in preview mode) Install Observability operator and Prometheus instance for federation"
    echo "  -e  --eaas       (only in preview mode) Install environment as a service components"
    echo "  -h, --help       Show this help message and exit"
    echo
    echo "Example usage: \`$0 preview --obo --eaas"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
