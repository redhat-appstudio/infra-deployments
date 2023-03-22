#!/bin/bash -e


main() {
    echo "Setting secrets for Pact Broker"
    kubectl create namespace hac-pact-broker -o yaml --dry-run=client | oc apply -f-
    if ! kubectl get secret -n hac-pact-broker pact-broker-secrets &>/dev/null; then
        kubectl create secret generic pact-broker-secrets \
            --namespace=hac-pact-broker \
            --from-literal=pact_broker_admin=$BROKER_USERNAME \
            --from-literal=pact_broker_admin_password=$BROKER_PASSWORD \
            --from-literal=pact_broker_user=$BROKER_USERNAME \
            --from-literal=pact_broker_user_password=$BROKER_PASSWORD \
            --from-literal=username=user \
            --from-literal=password=abc123
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
