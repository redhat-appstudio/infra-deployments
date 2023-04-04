---
title: Installing and configuring Grafana on the control-plane cluster
---

Note:
This section uses the **Grafana cluster** term to refer to the cluster on which Grafana is deployed. 
In a multi-cluster topology, there will be a single cluster on which Grafana is deployed, whereas Prometheus will be deployed on all clusters where metrics need to be collected.

## Prerequisites

### `appstudio-workload-monitoring` Namespace

Note: The steps below should be handled by Argo CD

First, create the `appstudio-workload-monitoring` namespace on each Prometheus or Grafana cluster, if it does not exist yet:

```
$ oc create namespace appstudio-workload-monitoring
```
### Dev environment only

The Grafana instance will be deployed by the Grafana operator and a route will be created automatically.

After bootstrapping the cluster in `preview` mode, create the following yaml file:
```
spec:
  config:
    security:
      admin_user: <username>
      admin_password: <password> 
```
Add your desired username and password,  
Save it as `grafana-login.yaml` and run the command:  
```
$ oc patch grafana appstudio-grafana -n appstudio-workload-monitoring --type merge --patch-file grafana-login.yaml
```

Now you can login to the Grafana UI using the provided credentials.
 

### OAuth2 proxy secrets

Grafana UI is protected by an OAuth2 proxy running as a sidecar container and which delegates the authentication to GitHub. 
Users must belong to the [Red Hat Appstudio SRE organization](https://github.com/redhat-appstudio-sre) team to be allowed to access the UI.

Create the secret with the following commands:

```
$ ./hack/setup-monitoring.sh oauth2-secret grafana-oauth2-proxy $GRAFANA_GITHUB_CLIENT_ID $GRAFANA_GITHUB_CLIENT_SECRET $GRAFANA_GITHUB_COOKIE_SECRET
```

The `GRAFANA_GITHUB_CLIENT_ID`/`GRAFANA_GITHUB_CLIENT_SECRET` value pair must match an existing "OAuth Application" on GitHub - see [OAuth apps](https://github.com/organizations/redhat-appstudio-sre/settings/applications) in the [Red Hat Appstudio SRE organization](https://github.com/organizations/redhat-appstudio-sre). 
The `GRAFANA_GITHUB_COOKIE_SECRET` can be generated using the [following instructions](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview#generating-a-cookie-secret).


The `grafana-oauth2-proxy` secret must be created before deploying Prometheus and Grafana, otherwise the pods will fail to run.

### Grafana Datasources

Grafana datasources contain the connection settings to the Prometheus instances. These datasources are stored in secrets in the `appstudio-workload-monitoring` namespace of the **Grafana cluster**.

The Prometheus endpoints called by Grafana are protected by an OAuth proxy running as a sidecar container and which checks that the incoming requests contain a valid token. A token is valid if it belongs to a service account of the **Prometheus cluster** which has the RBAC permission to "get namespaces". Such a permission can be obtained with the `cluster-monitoring-view` cluster role.

In a multi-cluster setup, Grafana will have a datasource secret for each instance of Prometheus. 
A datasource has a name (`DATASOURCE_NAME`), an URL (`PROMETHEUS_URL`) and a token (`GRAFANA_OAUTH_TOKEN`) obtained as follow:

`DATASOURCE_NAME` is the name of the datasource as it will appear in Grafana. It is also the name of the secret which will contain the YAML file defining the datasource itself.
`DATASOURCE_NAME` is an arbitrary name, for example `cluster-1-prometheus-openshift-ds` for Prometheus running in the `openshift-monitoring` namespace of Cluster-1.

`PROMETHEUS_URL` is obtained from the route created for Prometheus in the `openshift-monitoring` and `appstudio-workload-monitoring` namespaces in the **Prometheus cluster**:

```
$ export PROMETHEUS_URL_OPENSHIFT=`oc get route/prometheus-k8s -n openshift-monitoring -o json | jq -r '.status.ingress[0].host'`

$ export PROMETHEUS_URL_APPSTUDIO=`oc get route/prometheus-oauth -n appstudio-workload-monitoring -o json | jq -r '.status.ingress[0].host'`
```

`GRAFANA_OAUTH_TOKEN` is obtained by requesting a token for the `grafana-oauth` service account in the **Prometheus cluster**:
```
$ export GRAFANA_SECRET_NAME=$(oc -n appstudio-workload-monitoring get sa/grafana-oauth -o jsonpath="{.secrets[0].name}")

$ export GRAFANA_OAUTH_TOKEN=`oc -n appstudio-workload-monitoring create token grafana-oauth --bound-object-kind Secret --bound-object-name $GRAFANA_SECRET_NAME --duration=8760h`
```
Note: We are keeping expiration duration to one year. So we need to keep this renewing.
TODO: find a way to get token and renew this automatically.

Using the values obtained from the **Prometheus cluster**, run the following command on the **Grafana cluster**:
For current setup we have two datasource `prometheus-appstudio-ds` and `prometheus-openshift-ds`

```
$ export DATASOURCE_APPSTUDIO="prometheus-appstudio-ds"
$ export DATASOURCE_OPENSHIFT="prometheus-openshift-ds"

$ ./hack/setup-monitoring.sh grafana-datasource-secret $DATASOURCE_APPSTUDIO $PROMETHEUS_URL_APPSTUDIO $GRAFANA_OAUTH_TOKEN

$ ./hack/setup-monitoring.sh grafana-datasource-secret $DATASOURCE_OPENSHIFT $PROMETHEUS_URL_OPENSHIFT $GRAFANA_OAUTH_TOKEN
```

Notes: 
- The `grafana-oauth` service account is created by `components/monitoring/base/prometheus/configure-prometheus.yaml` along with a binding to the `cluster-monitoring-view` cluster role. 
- The same token can be used in datasources secrets related to the Prometheus instances deployed in the `openshift-monitoring` and `appstudio-workload-monitoring` namespaces.

## Installation and Configuration

create the "base" resources by running the following command:

```
$ kustomize build components/monitoring/grafana/base | oc apply -f -   
```