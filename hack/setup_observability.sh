#!/bin/bash
#This script installs the prometheus and granafa operator
#from community operator hub. It then configures them with
#github oauth

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [ -f $ROOT/hack/monitoring.env ]; then
    source $ROOT/hack/monitoring.env
else
    echo "Please create the hack/monitoring.env file with oauth values"
    exit 1
fi

#Check if the variables for prometheus are set or not
[[ -z "$PROMETHEUS_GITHUB_CLIENT_ID" ]] && { echo "Please add github oauth client id for prometheus access"; exit 1; }
[[ -z "$PROMETHEUS_GITHUB_CLIENT_SECRET" ]] && { echo "Please add github oauth client secret for prometheus access"; exit 1; }
[[ -z "$PROMETHEUS_GITHUB_COOKIE_SECRET" ]] && { echo "Please add github cookie secret for prometheus access"; exit 1; }

#Check if the variables for grafana are set or not
[[ -z "$GRAFANA_GITHUB_CLIENT_ID" ]] && { echo "Please add github oauth client id for grafana access"; exit 1; }
[[ -z "$GRAFANA_GITHUB_CLIENT_SECRET" ]] && { echo "Please add github oauth client secret for grafana access"; exit 1; }
[[ -z "$GRAFANA_GITHUB_COOKIE_SECRET" ]] && { echo "Please add github cookie secret for grafana access"; exit 1; }
[[ -z "$GRAFANA_ADMIN_USER" ]] && { echo "Please add Grafana admin username"; exit 1; }
[[ -z "$GRAFANA_ADMIN_PASSWD" ]] && { echo "Please add password for grafana admin"; exit 1; }


cd $ROOT/hack/monitoring

#Install prometheus operator
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -f install_prometheus.yaml|oc apply -f -

#Name of the secret token used by prometheus-k8s serviceaccount
export PROMETHEUS_K8S_SECRET_NAME=$(oc -n $MONITORING_NAMESPACE get sa/prometheus-k8s -o jsonpath="{.secrets[0].name}")

#Check if prometheus name of the prometheus token secret is set
[[ -z "$PROMETHEUS_K8S_SECRET_NAME" ]] && { echo "could not retrieve name of the secret for prometheus serviceaccount token"; exit 1; }

#Configure prometheus with github oauth
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -p PROMETHEUS_GITHUB_CLIENT_ID=$PROMETHEUS_GITHUB_CLIENT_ID -p PROMETHEUS_GITHUB_CLIENT_SECRET=$PROMETHEUS_GITHUB_CLIENT_SECRET -p PROMETHEUS_GITHUB_COOKIE_SECRET=$PROMETHEUS_GITHUB_COOKIE_SECRET -p PROMETHEUS_K8S_SECRET_NAME=$PROMETHEUS_K8S_SECRET_NAME -f configure_prometheus.yaml|oc apply -f -

#Add ServiceMonitors to prometheus
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -p PROMETHEUS_K8S_SECRET_NAME=$PROMETHEUS_K8S_SECRET_NAME -f prometheus_servicemonitors.yaml|oc apply -f -

#Give prometheus service account access to scrape the operators
oc policy add-role-to-user view system:serviceaccount:openshift-customer-monitoring:prometheus-k8s -n toolchain-host-operator
oc policy add-role-to-user view system:serviceaccount:openshift-customer-monitoring:prometheus-k8s -n toolchain-member-operator


#Install grafana operator
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -f install_grafana.yaml|oc apply -f -

#Configure grafana with github oauth
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -p GRAFANA_GITHUB_CLIENT_ID=$GRAFANA_GITHUB_CLIENT_ID -p GRAFANA_GITHUB_CLIENT_SECRET=$GRAFANA_GITHUB_CLIENT_SECRET -p GRAFANA_GITHUB_COOKIE_SECRET=$GRAFANA_GITHUB_COOKIE_SECRET -p GRAFANA_ADMIN_PASSWD=$GRAFANA_ADMIN_PASSWD -p GRAFANA_ADMIN_USER=$GRAFANA_ADMIN_USER -f configure_grafana.yaml|oc apply -f -

#Give grafana serviceaccount view permission to datasources
oc adm policy add-cluster-role-to-user cluster-monitoring-view system:serviceaccount:openshift-customer-monitoring:grafana-serviceaccount

#Configure grafana datasource
oc apply -f grafanadatasource_prometheus.yaml

#Configure grafana dashboard
oc apply -f grafanadashboard_sandbox_usage.yaml

cd $ROOT
