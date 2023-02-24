#!/bin/bash -e

main() {
    echo "Setting secrets for GitOps"
    create_namespace
    create_db_secret
}

create_namespace() {
    if kubectl get namespace gitops &>/dev/null; then
        echo "gitops namespace already exists, skipping creation"
        return
    fi
    kubectl create namespace gitops -o yaml --dry-run=client | oc apply -f-
}

create_db_secret() {
    echo "Creating DB secret" >&2
    if kubectl get secret -n gitops gitops-postgresql-staging &>/dev/null; then
        echo "DB secret already exists, skipping creation"
        return
    fi
    kubectl create secret generic gitops-postgresql-staging \
        --namespace=gitops \
        --from-literal=postgresql-password="$(openssl rand -base64 20)"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
