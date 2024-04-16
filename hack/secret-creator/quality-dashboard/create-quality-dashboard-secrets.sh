#!/bin/bash -e
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

main() {
    echo "Setting secrets for Quality Dashboard"
    DOMAIN=$(oc get ingresses.config.openshift.io cluster --template={{.spec.domain}})

    NS=quality-dashboard

    BACKEND_URL="https://backend-$NS.${DOMAIN}"
    DEX_URL="https://dex-$NS.${DOMAIN}"
    FRONTEND_URL="https://frontend-$NS.${DOMAIN}"
    
    SECRET_NAME=$NS-secrets

    APP_ID=${QD_APP_ID:-redhat-quality-studio-app}

    IN_CLUSTER_DB=${QD_IN_CLUSTER_DB:-false}
    DB_NAME=${QD_DB_NAME:-quality}
    DB_USER=${QD_DB_USER:-qd}
    DB_PASSWORD=${QD_DB_PASSWORD:-qd}
    DB_HOST=${QD_DB_HOST:-postgresql}

    JIRA_TOKEN=${QD_JIRA_TOKEN:-}
    GITHUB_TOKEN=${QD_GITHUB_TOKEN:-$MY_GITHUB_TOKEN}
    SLACK_TOKEN=${QD_SLACK_TOKEN:-}

    DEX_CONFIG_TEMPLATE_PATH=$SCRIPT_DIR/dex-config-template.yaml
    DEX_CONFIG_PATH=$SCRIPT_DIR/dex-config.yaml

    GITHUB_ORG=${QD_GITHUB_ORG:-$MY_GITHUB_ORG}
    OAUTH_CLIENT_ID=${QD_OAUTH_CLIENT_ID:-}
    OAUTH_CLIENT_SECRET=${QD_OAUTH_CLIENT_SECRET:-}

    kubectl create namespace $NS -o yaml --dry-run=client | oc apply -f-

    if [ "${IN_CLUSTER_DB}" = "true" ]; then
        oc process postgresql-ephemeral -n openshift -p POSTGRESQL_USER="$DB_USER" -p POSTGRESQL_PASSWORD="$DB_PASSWORD" -p POSTGRESQL_DATABASE="$DB_NAME" | oc apply -n $NS -f -
    fi

    cp "$DEX_CONFIG_TEMPLATE_PATH" "$DEX_CONFIG_PATH"
    yq -i ".issuer=\"https://dex-$NS.$DOMAIN/dex\"" $DEX_CONFIG_PATH
    yq -i "(.staticClients.[] | select(.name==\"Red Hat Quality Studio\")) |=(.id=\"$APP_ID\",.redirectURIs=[\"$FRONTEND_URL/home/overview\", \"$FRONTEND_URL/login\", \"$FRONTEND_URL/\"])" $DEX_CONFIG_PATH
    yq -i "(.connectors.[] | select(.name==\"GitHub\").config) |=(.clientID=\"$OAUTH_CLIENT_ID\",.clientSecret=\"$OAUTH_CLIENT_SECRET\",.redirectURI=\"$DEX_URL/dex/callback\",.orgs=[{\"name\":\"$GITHUB_ORG\"}])" $DEX_CONFIG_PATH

    if ! kubectl get secret $SECRET_NAME -n $NS  &>/dev/null; then
        kubectl create secret generic $SECRET_NAME \
            --namespace=$NS \
            --from-literal=backend-route=$BACKEND_URL \
            --from-literal=dex-application-id=$APP_ID \
            --from-literal=dex-issuer="$DEX_URL/dex" \
            --from-literal=frontend-route=$FRONTEND_URL/login \
            --from-literal=github-token=$GITHUB_TOKEN \
            --from-literal=jira-token=$JIRA_TOKEN \
            --from-literal=rds-endpoint=$DB_HOST \
            --from-literal=slack_token=$SLACK_TOKEN \
            --from-literal=storage-database=$DB_NAME \
            --from-literal=storage-password=$DB_PASSWORD \
            --from-literal=storage-user=$DB_USER \
            --from-file=$DEX_CONFIG_PATH
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi