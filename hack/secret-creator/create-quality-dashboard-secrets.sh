#!/bin/bash -e

##### TESTING
main() {
    echo "Setting secrets for Quality Dashboard"
    kubectl create namespace quality-dashboard -o yaml --dry-run=client | oc apply -f-
    if ! kubectl get secret -n quality-dashboard quality-dashboard-secrets &>/dev/null; then
        kubectl create secret generic quality-dashboard-secrets \
            --namespace=quality-dashboard \
            --from-literal=rds-endpoint=REPLACE_WITH_RDS_ENDPOINT \
            --from-literal=storage-user=postgres \
            --from-literal=storage-password=REPLACE_DB_PASSWORD \
            --from-literal=storage-database=quality \
            --from-literal=github-token=REPLACE_GITHUB_TOKEN \
            --from-literal=jira-token=REPLACE_JIRA_TOKEN
    fi
    if ! kubectl get secret -n quality-dashboard quality-dashboard-auth &>/dev/null; then
        kubectl create secret generic quality-dashboard-auth \
            --namespace=quality-dashboard \
            --from-literal=users.htpasswd=NOUSER
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
