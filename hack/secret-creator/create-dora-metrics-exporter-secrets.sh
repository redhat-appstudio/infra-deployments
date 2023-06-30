#!/bin/bash -e

main() {
    echo "Setting secrets for Dora metrics exporter"
    kubectl create namespace dora-metrics -o yaml --dry-run=client | oc apply -f-
    if ! kubectl get secret -n dora-metrics exporters-secret &>/dev/null; then
        kubectl create secret generic exporters-secret -n dora-metrics \
            --from-literal=github=${github_token:-""} --from-literal=pager-duty-token=${pager-duty-token:-""}
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
