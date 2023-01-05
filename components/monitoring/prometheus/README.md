---
title: Installing and configuring Prometheus on the workload clusters
---

Note:
This section uses the **Prometheus cluster** term to refer to the clusters on which Prometheus is deployed. 
In a multi-cluster topology, there will be a single cluster on which Grafana is deployed, whereas Prometheus will be deployed on all clusters where metrics need to be collected.

## Prerequisites

### `appstudio-workload-monitoring` Namespace

Note: The steps below should be handled by Argo CD

First, create the `appstudio-workload-monitoring` namespace on each Prometheus or Grafana cluster, if it does not exist yet:

```
$ oc create namespace appstudio-workload-monitoring
```

### OAuth2 proxy secrets

Prometheus UI is protected by an OAuth2 proxy running as a sidecar container which delegates the authentication to GitHub. 
Users must belong to the [Red Hat Appstudio SRE organization](https://github.com/redhat-appstudio-sre) team to be allowed to access the UI.

On each Prometheus cluster, create the secret with the following command:

```
$ ./hack/setup-monitoring.sh oauth2-secret prometheus-oauth2-proxy $PROMETHEUS_GITHUB_CLIENT_ID $PROMETHEUS_GITHUB_CLIENT_SECRET $PROMETHEUS_GITHUB_COOKIE_SECRET
```

The `PROMETHEUS_GITHUB_CLIENT_ID`/`PROMETHEUS_GITHUB_CLIENT_SECRET` value pair must match an existing "OAuth Application" on GitHub - see [OAuth apps](https://github.com/organizations/redhat-appstudio-sre/settings/applications) in the [Red Hat Appstudio SRE organization](https://github.com/organizations/redhat-appstudio-sre). 
The `PROMETHEUS_GITHUB_COOKIE_SECRET` can be generated using the [following instructions](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview#generating-a-cookie-secret).

Each Prometheus instance must have its own OAuth Application on GitHub and its own `prometheus-oauth2-proxy` secret.

The `prometheus-oauth2-proxy` secret must be created before deploying Prometheus, otherwise pod creation will fail.

## Installation and Configuration

Create the resources by running the following command:

```
$ kustomize build components/monitoring/prometheus/base | oc apply -f -   
```