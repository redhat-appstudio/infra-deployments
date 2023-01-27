---
title: Installing and configuring Grafana on the control-plane cluster
---

Note:
This section uses the **Grafana cluster** term to refer to the cluster on which Grafana is deployed. 
In a multi-cluster topology, there will be a single cluster on which Grafana is deployed, whereas Prometheus will be deployed on all clusters where metrics need to be collected.

## Prerequisites

### OAuth2 proxy secrets

Grafana UI is protected by an OAuth2 proxy running as a sidecar container and which delegates the authentication to GitHub. 
Users must belong to the [Red Hat Appstudio SRE organization](https://github.com/redhat-appstudio-sre) team to be allowed to access the UI.

Create the secret with the following commands:

```
$ ./hack/setup-monitoring.sh grafana-oauth2-secret $GRAFANA_GITHUB_CLIENT_ID $GRAFANA_GITHUB_CLIENT_SECRET $GRAFANA_GITHUB_COOKIE_SECRET    
```

The `GRAFANA_GITHUB_CLIENT_ID`/`GRAFANA_GITHUB_CLIENT_SECRET` value pair must match an existing "OAuth Application" on GitHub - see [OAuth apps](https://github.com/organizations/redhat-appstudio-sre/settings/applications) in the [Red Hat Appstudio SRE organization](https://github.com/organizations/redhat-appstudio-sre). 
The `GRAFANA_GITHUB_COOKIE_SECRET` can be generated using the [following instructions](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview#generating-a-cookie-secret).


The `grafana-oauth2-proxy` secret must be created before deploying Prometheus and Grafana, otherwise the pods will fail to run.

### Grafana Datasources

Grafana datasources contain the connection settings to the Thanos instances. These datasources are stored in secrets in the `openshift-user-workload-monitoring` namespace of the **Grafana cluster**.

The Thanos endpoints called by Grafana are protected by an OAuth proxy running as a sidecar container and which checks that the incoming requests contain a valid token. A token is valid if it belongs to a service account of the **Prometheus cluster** which has the RBAC permission to "get namespaces". Such a permission can be obtained with the `cluster-monitoring-view` cluster role.

In a multi-cluster setup, Grafana will have one datasource secret per remote instance of Thanos. 
A datasource has a name (`DATASOURCE_NAME`), an URL (`THANOS_QUERIER_URL`) and a token (`GRAFANA_OAUTH_TOKEN`) obtained as follow:

- `DATASOURCE_NAME` is the name of the datasource as it will appear in Grafana. It is also the name of the secret which will contain the YAML file defining the datasource itself.
`DATASOURCE_NAME` is an arbitrary name, for example `cluster-1-prometheus-openshift-ds` for Prometheus running in the `openshift-monitoring` namespace of Cluster-1.
- `THANOS_QUERIER_URL` is obtained from the route created for Thanos Querier in the `openshift-monitoring` namespace in the **Prometheus cluster**:
- `GRAFANA_OAUTH_TOKEN` is obtained by requesting a token for the `grafana-oauth` service account in the **Prometheus cluster**:

```
$ export GRAFANA_SECRET_NAME=`oc -n openshift-user-workload-monitoring get sa/grafana-oauth -o jsonpath="{.secrets[0].name}"`

$ export GRAFANA_OAUTH_TOKEN=`oc -n openshift-user-workload-monitoring create token grafana-oauth --bound-object-kind Secret --bound-object-name $GRAFANA_SECRET_NAME --duration=8760h` # requesting a token valid for 1 year

$ export THANOS_QUERIER_URL=`oc get route/thanos-querier -n openshift-monitoring -o json | jq -r '.status.ingress[0].host'`
```

Using these values, run the following command on the **Grafana cluster**:

```
$ export DATASOURCE_NAME="thanos-querier-ds" # unique per Prometheus cluster

$ ./hack/setup-monitoring.sh grafana-datasource-secret $DATASOURCE_NAME $THANOS_QUERIER_URL $GRAFANA_OAUTH_TOKEN
```

Notes: 
- The `grafana-oauth` service account is created by `components/monitoring/base/prometheus/configure-prometheus.yaml` along with a binding to the `cluster-monitoring-view` cluster role. 
- The same token can be used in datasources secrets related to the Prometheus instances deployed in the `openshift-monitoring` and `openshift-user-workload-monitoring` namespaces.

## Installation and Configuration

create the "base" resources by running the following command:

```
$ kustomize build components/monitoring/grafana/base | oc apply -f -   
```

Once the pods are running, you can access Grafana at the following URL

```
$ echo https://`oc get route/grafana -o json -n openshift-user-workload-monitoring | jq -r '.status.ingress[0].host'`/
```