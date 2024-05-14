#!/bin/bash -e

main() {
    echo "Creating secret for CI Helper App"

    NS=ci-helper-app
    SECRET_NAME=$NS-secrets

    APP_ID=${CI_HELPER_GITHUB_APP_INTEGRATION_ID:-}
    PRIVATE_KEY=${CI_HELPER_GITHUB_APP_PRIVATE_KEY:-}
    WEBHOOK_SECRET=${CI_HELPER_GITHUB_APP_WEBHOOK_SECRET:-}

    kubectl create namespace $NS -o yaml --dry-run=client | oc apply -f-

    if ! kubectl get secret $SECRET_NAME -n $NS  &>/dev/null; then
        kubectl create secret generic $SECRET_NAME \
            --namespace=$NS \
            --from-literal=app-id="$APP_ID" \
            --from-literal=app-private-key="$PRIVATE_KEY" \
            --from-literal=webhook-secret="$WEBHOOK_SECRET"
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi