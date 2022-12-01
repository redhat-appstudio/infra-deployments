# Monitoring for Prometheus clusters

Note:
This section uses **Grafana cluster** and **Prometheus cluster** to refer to the clusters on which Grafana and Prometheus are deployed, respectively. In a multi-cluster topology, there will be a single cluster on which Grafana is deployed, whereas Prometheus will be deployed on all clusters where metrics need to be collected.

## Prerequisites

Note: The steps below should be handled by Argo CD

First, create the `appstudio-workload-monitoring` namespace on each Prometheus or Grafana cluster, if it does not exist yet:

```
$ oc create namespace appstudio-workload-monitoring
```

and create the "base" resources by running the following command:

```
$ kustomize build components/monitoring/base | oc apply -f -   
```

## OAuth2 proxy secrets

Both Prometheus and Grafana UIs are protected by an OAuth2 proxy running as a sidecar container and which delegates the authentication to GitHub. 
Users must belong to the [Red Hat Appstudio SRE organization](https://github.com/redhat-appstudio-sre) team configured in the OAuth2 proxy to be allowed to access the Web UIs.

Create the secrets with the following commands:

```
# on each Prometheus cluster
$ ./hack/setup-monitoring.sh oauth2-secret prometheus-oauth2-proxy $PROMETHEUS_GITHUB_CLIENT_ID $PROMETHEUS_GITHUB_CLIENT_SECRET $PROMETHEUS_GITHUB_COOKIE_SECRET

# on the Grafana cluster
$ ./hack/setup-monitoring.sh oauth2-secret grafana-oauth2-proxy $GRAFANA_GITHUB_CLIENT_ID $GRAFANA_GITHUB_CLIENT_SECRET $GRAFANA_GITHUB_COOKIE_SECRET
```

The `PROMETHEUS_GITHUB_CLIENT_ID`/`PROMETHEUS_GITHUB_CLIENT_SECRET` and `GRAFANA_GITHUB_CLIENT_ID`/`GRAFANA_GITHUB_CLIENT_SECRET` value pairs must match an existing "OAuth Application" on GitHub - see [OAuth apps](https://github.com/organizations/redhat-appstudio-sre/settings/applications) in the [Red Hat Appstudio SRE organization](https://github.com/organizations/redhat-appstudio-sre). The `PROMETHEUS_GITHUB_COOKIE_SECRET` and `GRAFANA_GITHUB_COOKIE_SECRET` can be generated using the [following instructions](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview#generating-a-cookie-secret).

Each Prometheus instance must have its own OAuth Application on GitHub and its own `prometheus-oauth2-proxy` secret, whereas Grafana needs a single OAuth Application on GitHub since it is only deployed once.

These `prometheus-oauth2-proxy` and `grafana-oauth2-proxy` secrets must be created before deploying Prometheus and Grafana, otherwise the pods will fail to run.

## Grafana Datasources

Grafana datasources contain the connection settings to the Prometheus instances. These datasources are stored in secrets in the `appstudio-workload-monitoring` namespace of the **Grafana cluster**.

The Prometheus endpoints called by Grafana are protected by an OAuth proxy running as a sidecar container and which checks that the incoming requests contain a valid token. A token is valid if it belongs to a service account of the **Prometheus cluster** which has the RBAC permission to "get namespaces". Such a permission can be obtained with the `cluster-monitoring-view` cluster role.

In a multi-cluster setup, Grafana will have a datasource secret for each instance of Prometheus. 
A datasource has a name (`DATASOURCE_NAME`), an URL (`PROMETHEUS_URL`) and a token (`GRAFANA_OAUTH_TOKEN`) obtained as follow:

`DATASOURCE_NAME` is the name of the datasource as it will appear in Grafana. It is also the name of the secret which will contain the YAML file defining the datasource itself.
`DATASOURCE_NAME` is an arbitrary name, for example `cluster-1-prometheus-openshift-ds` for Prometheus running in the `openshift-monitoring` namespace of Cluster-1.

`PROMETHEUS_URL` is obtained from the route created for Prometheus in the `openshift-monitoring` and `appstudio-workload-monitoring` namespaces in the **Prometheus cluster**:
```
$ PROMETHEUS_URL=`oc get route/prometheus-k8s -n openshift-monitoring -o json | jq -r '.status.ingress[0].host'`

$ PROMETHEUS_URL=`oc get route/prometheus-oauth -n appstudio-workload-monitoring -o json | jq -r '.status.ingress[0].host'`
```

`GRAFANA_OAUTH_TOKEN` is obtained by requesting a token for the `grafana-oauth` service account in the **Prometheus cluster**:
```
$ GRAFANA_OAUTH_TOKEN=`oc create token grafana-oauth -n appstudio-workload-monitoring`
```
Notes: 
- The `grafana-oauth` service account is created by `components/monitoring/base/configure-prometheus.yaml` along with a binding to the `cluster-monitoring-view` cluster role. 
- The same token can be used in datasources secrets related to the Prometheus instances deployed in the `openshift-monitoring` and `appstudio-workload-monitoring` namespaces.

Using the values obtained from the **Prometheus cluster**, run the following command on the **Grafana cluster**:

```
$ ./hack/setup-monitoring.sh grafana-datasource-secret $DATASOURCE_NAME $PROMETHEUS_URL $GRAFANA_OAUTH_TOKEN
```