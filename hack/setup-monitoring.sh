#!/bin/bash
# This script ensures that PROMETHEUS and GRAFANA environment variables are set and creates all required secrets to 
# install and configure Prometheus and Grafana on the workload clusters.

#Check if the variables for prometheus are set or not
[[ -z "$PROMETHEUS_GITHUB_CLIENT_ID" ]] && { echo "Please add github oauth client id for prometheus access"; exit 1; }
[[ -z "$PROMETHEUS_GITHUB_CLIENT_SECRET" ]] && { echo "Please add github oauth client secret for prometheus access"; exit 1; }
[[ -z "$PROMETHEUS_GITHUB_COOKIE_SECRET" ]] && { echo "Please add github cookie secret for prometheus access"; exit 1; }

#Check if the variables for grafana are set or not
[[ -z "$GRAFANA_GITHUB_CLIENT_ID" ]] && { echo "Please add github oauth client id for grafana access"; exit 1; }
[[ -z "$GRAFANA_GITHUB_CLIENT_SECRET" ]] && { echo "Please add github oauth client secret for grafana access"; exit 1; }
[[ -z "$GRAFANA_GITHUB_COOKIE_SECRET" ]] && { echo "Please add github cookie secret for grafana access"; exit 1; }

oc create namespace appstudio-workload-monitoring

oc create secret generic prometheus-proxy-config \
  -n appstudio-workload-monitoring  \
  --from-literal=client-id=$PROMETHEUS_GITHUB_CLIENT_ID \
  --from-literal=client-secret=$PROMETHEUS_GITHUB_CLIENT_SECRET \
  --from-literal=cookie-secret=$PROMETHEUS_GITHUB_COOKIE_SECRET

oc create secret generic grafana-oauth2-proxy-config \
  -n appstudio-workload-monitoring  \
  --from-literal=client-id=$GRAFANA_GITHUB_CLIENT_ID \
  --from-literal=client-secret=$GRAFANA_GITHUB_CLIENT_SECRET \
  --from-literal=cookie-secret=$GRAFANA_GITHUB_COOKIE_SECRET