#!/bin/bash

# ----------------------------------------------------------------
# OAuth2 Proxy Secrets
# ----------------------------------------------------------------

oauth2-secret() {
  if [[ $# -ne 4 ]]; then
    echo "invalid number of arguments"
    echo "usage:"
    echo "  $0 oauth2-secret NAME CLIENT_ID CLIENT_SECRET COOKIE_SECRET"
    exit 1
  fi

  NAME=$1
  CLIENT_ID=$2
  CLIENT_SECRET=$3
  COOKIE_SECRET=$4

  oc create secret generic $NAME \
    -n appstudio-workload-monitoring  \
    --from-literal=client-id=$CLIENT_ID \
    --from-literal=client-secret=$CLIENT_SECRET \
    --from-literal=cookie-secret=$COOKIE_SECRET \
    --dry-run=client -o yaml | oc apply -f -
}

# ----------------------------------------------------------------
# Grafana Datasource Secrets
# ----------------------------------------------------------------

grafana-datasource-secret() {

  if [[ $# -ne 3 ]]; then
    echo "invalid number of arguments"
    echo "usage:"
    echo "  $0 grafana-datasource-secret NAME URL TOKEN"
    exit 1
  fi

  NAME=$1
  URL=$2
  TOKEN=$3

  DATA="apiVersion: 1
datasources: 
- name: $NAME
  type: prometheus
  access: proxy
  url: https://$URL
  basicAuth: false
  withCredentials: true
  isDefault: false
  jsonData:
    httpHeaderName1: 'Authorization'
    timeInterval: 5s
    tlsSkipVerify: true
  secureJsonData:
    httpHeaderValue1: 'Bearer $TOKEN'"
  
  oc create secret generic $NAME -n appstudio-workload-monitoring --from-literal=$NAME.yaml="$DATA" --dry-run=client -o yaml | oc apply -f -
}

# -----------------------------------------------------------------
if declare -f "$1" > /dev/null
then
    # call arguments verbatim
    "$@"
else
    # Show a helpful error
    echo "'$1' is not a valid command" >&2
    echo "available commands:"
    echo "oauth2-secret               Create the secret for Grafana's or Prometheus's OAuth2 proxy"
    echo "grafana-datasource-secret   Create the secret for a Grafana datasource"
    echo ""
    exit 1
fi