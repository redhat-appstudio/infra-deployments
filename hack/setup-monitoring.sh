#!/bin/bash

# ----------------------------------------------------------------
# Service Monitor secrets (in each operator namespace)
# ----------------------------------------------------------------

service-monitor-secret() {

  
  if [[ $# -ne 3 ]]; then
    echo "invalid number of arguments"
    echo "usage:"
    echo "  $0 service-monitor-secret OPERATOR_NAMESPACE SERVICE_MONITOR_NAME SECRET_NAME"
    exit 1
  fi

  OPERATOR_NAMESPACE=$1
  SERVICE_MONITOR_NAME=$2
  SECRET_NAME=$3

  PROMETHEUS_UWM_SECRET_NAME=`oc -n openshift-user-workload-monitoring get sa/prometheus-user-workload -o jsonpath="{.secrets[0].name}"`
  PROMETHEUS_UWM_TOKEN=`oc -n openshift-user-workload-monitoring create token prometheus-user-workload --bound-object-kind Secret --bound-object-name $PROMETHEUS_UWM_SECRET_NAME --duration=8760h` # requesting a token valid for 1 year

  # use `--dry-run=client -o yaml | oc apply -f -` to overwrite the secret if it already exists
  oc create secret generic -n $OPERATOR_NAMESPACE $SECRET_NAME --from-literal=token=$PROMETHEUS_UWM_TOKEN --dry-run=client -o yaml | oc apply -f -
  
  # annotate the ServiceMonitor with a timestamp (date expressed in UTC time)
  # This will trigger a reconcile loop in the Prometheus Operator
  oc annotate -n $OPERATOR_NAMESPACE servicemonitor $SERVICE_MONITOR_NAME last-secret-update=`date -u +%Y-%m-%dT%H:%M:%SZ` --overwrite=true 
}

# ----------------------------------------------------------------
# Grafana OAuth2 Proxy Secret (only 1, for Grafana)
# ----------------------------------------------------------------

grafana-oauth2-secret() {
  if [[ $# -ne 3 ]]; then
    echo "invalid number of arguments"
    echo "usage:"
    echo "  $0 grafana-oauth2-secret CLIENT_ID CLIENT_SECRET COOKIE_SECRET"
    exit 1
  fi

  NAME=grafana-oauth2-proxy 
  CLIENT_ID=$1
  CLIENT_SECRET=$2
  COOKIE_SECRET=$3

  # use `--dry-run=client -o yaml | oc apply -f -` to overwrite the resource if it already exists
  # (there may be a warning on the first time, if `oc create` was not executed with `--save-config`)
  oc create secret generic $NAME \
    -n openshift-user-workload-monitoring  \
    --from-literal=client-id=$CLIENT_ID \
    --from-literal=client-secret=$CLIENT_SECRET \
    --from-literal=cookie-secret=$COOKIE_SECRET \
    --dry-run=client -o yaml | oc apply -f -
}

# ----------------------------------------------------------------
# Grafana Datasource Secrets (1 per Prometheus instance)
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
  
  # use `--dry-run=client -o yaml | oc apply -f -` to overwrite the resource if it already exists
  oc create secret generic $NAME -n openshift-user-workload-monitoring --from-literal=$NAME.yaml="$DATA" --dry-run=client -o yaml | oc apply -f -
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
    echo "grafana-oauth2-secret       Create the secret for Grafana's OAuth2 proxy"
    echo "grafana-datasource-secret   Create the secret for a Grafana datasource"
    echo "service-monitor-secret      Create the secret for a target to scrape "
    echo ""
    exit 1
fi