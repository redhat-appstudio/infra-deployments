#!/bin/bash
#This script installs the prometheus and granafa operator
#from community operator hub. It then configures them with
#github oauth

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

cd $ROOT/hack/monitoring

#Namespace to which we want to install observability stack
#default is openshift-customer-monitoring
export MONITORING_NAMESPACE="openshift-customer-monitoring"

#Values for the below variables need to be replaced with actual oauth values from github
#Check readme for details about how to get github oauth details
#Github access tokens for prometheus oauth
export PROMETHEUS_GITHUB_CLIENT_ID="<base64_encoded_github_client_id_for_prometheus>"
export PROMETHEUS_GITHUB_CLIENT_SECRET="<base64_encoded_github_client_secret_for_prometheus>"
export PROMETHEUS_GITHUB_COOKIE_SECRET="<base64_encoded_github_cookie_secret>"
export PROMETHEUS_K8S_SECRET_NAME="<prometheus_secret_for_serviceaccounts>"

#Github access tokens for grafana oauth
export GRAFANA_GITHUB_CLIENT_ID="<base64_encoded_github_clientid_for_grafana>"
export GRAFNA_GITHUB_CLIENT_SECRET="<base64_encoded_github_client_secret_for_grafana>"
export GRAFANA_GITHUB_COOKIE_SECRET="<base64_encoded_github_cookie_secret>"
export GRAFANA_ADMIN_USER="<admin_userid_for_grafana>"
export GRAFANA_ADMIN_PASSWD="<password_for_grafana_admin_user>"

#Install prometheus operator
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -f install_prometheus.yaml|oc apply -f -

#Configure prometheus with github oauth
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -p PROMETHEUS_GITHUB_CLIENT_ID=$PROMETHEUS_GITHUB_CLIENT_ID -p PROMETHEUS_GITHUB_CLIENT_SECRET=$PROMETHEUS_GITHUB_CLIENT_SECRET -p PROMETHEUS_GITHUB_COOKIE_SECRET=$PROMETHEUS_GITHUB_COOKIE_SECRET -p PROMETHEUS_K8S_SECRET_NAME=$PROMETHEUS_K8S_SECRET_NAME -f configure_prometheus.yaml|oc apply -f -

#Give prometheus service account access to scrape the operators
oc policy add-role-to-user view system:serviceaccount:openshift-customer-monitoring:prometheus-k8s -n toolchain-host-operator
oc policy add-role-to-user view system:serviceaccount:openshift-customer-monitoring:prometheus-k8s -n toolchain-member-operator


#Install grafana operator
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -f install_grafana.yaml|oc apply -f -

#Configure grafana with github oauth
oc process -p MONITORING_NAMESPACE=$MONITORING_NAMESPACE -p GRAFANA_GITHUB_CLIENT_ID=$GRAFANA_GITHUB_CLIENT_ID -p GRAFNA_GITHUB_CLIENT_SECRET=$GRAFNA_GITHUB_CLIENT_SECRET -p GRAFANA_GITHUB_COOKIE_SECRET=$GRAFANA_GITHUB_COOKIE_SECRET -p GRAFANA_ADMIN_PASSWD=$GRAFANA_ADMIN_PASSWD -p GRAFANA_ADMIN_USER=$GRAFANA_ADMIN_USER -f configure_grafana.yaml|oc apply -f -

#Configure grafana datasource
oc apply -f grafanadatasource_prometheus.yaml

#Configure grafana dashboard
oc apply -f grafanadashboard_sandbox_usage.yaml

cd $ROOT
