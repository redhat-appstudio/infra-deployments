---
title: Installing and configuring Prometheus on the workload clusters
---

Note:
This section uses the **Prometheus cluster** term to refer to the clusters on which Prometheus is deployed. 
In a multi-cluster topology, there will be a single cluster on which Grafana is deployed, whereas Prometheus will be deployed on all clusters where metrics need to be collected.

## Enabling User Workload Monitoring

The following command will trigger the installation of the Prometheus Operator in the `openshift-user-workload-monitoring` namespace and the deployment of Prometheus (2 replicas) and Thanos ruler (2 replicas):

```
$ kustomize build components/monitoring/prometheus/base | oc apply -f -   
```

Prometheus can be configured via the `user-workload-monitoring-config` configmap created in the `openshift-monitoring`. See the reference docs: https://docs.openshift.com/container-platform/4.12/monitoring/enabling-monitoring-for-user-defined-projects.html

Note: Prometheus will look for `ServiceMonitor` and `PodMonitor` resources in _all namespaces_. User namespaces can excluded by labelling them with `openshift.io/user-monitoring=false`.

## Configuring ServiceMonitors and PodMonitors

Each operator may provide a `ServiceMonitor` or a `PodMonitor` in its own namespace. 
The Prometheus Operator will scan all namespaces for such resources, and configure Prometheus targets accordingly.

Metric endpoints exposed by the operator pods may be secured or unsecured, which has some impact on how to configure their associated ServiceMonitor or PodMonitor.

### Unsecured Endpoints

When an operator exposes its endpoint in an unsecured manner (see example below), then no extra action is necessary, Prometheus will be able to fetch the metrics out-of-the-box.


```
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  namespace: camel-k-operator
  name: camel-k-operator
spec:
  selector:
    matchLabels:
      app: "camel-k"
      camel.apache.org/component: operator
  podMetricsEndpoints:
  - interval: 30s
    port: metrics
    scheme: http
```

Note: Proper RBAC and NetworkPolicies in the operator namespace should help preventing other users from accessing the metrics endpoint.

### Secured Endpoints

When an operator exposes its endpoint in a secured manner (see example below), then some extra actions need to be undertaken, so that Prometheus can include a valid bearer token in the requests sent to fetch the metrics.

```
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
  name: metrics-monitor-uwm
  namespace: system
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      # The secret exists in the same namespace as this service monitor and accessible by the *Prometheus Operator*.
      bearerTokenSecret:
        name: host-operator-prometheus-user-workload
        key: token
      tlsConfig:
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
```

The ServiceMonitor above references a `Secret` which contains the bearer token that Prometheus will use when sending requests to the metrics endpoint. This secret needs to be in the same namespace as the ServiceMonitor itself. 


Running the following script will create the secret:

```
$ ./hack/setup-monitoring.sh service-monitor-secret OPERATOR_NAMESPACE SERVICE_MONITOR_NAME SECRET_NAME      
```

Note: the script also annotates the ServiceMonitor, which will notify the Prometheus Operator that the config changed (the operator watches ServiceMonitors but not Secrets)


Note: by default, Operator SDK generates a ServiceMonitor resource which uses a `bearerTokenFile` such as below:

```
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
  name: metrics-monitor
  namespace: system
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
```

But Prometheus will ignore such a ServiceMonitor with the following reason:

```
level=warn ts=2023-01-25T11:05:13.374328714Z caller=operator.go:2255 component=prometheusoperator msg="skipping servicemonitor" error="it accesses file system via bearer token file which Prometheus specification prohibits" servicemonitor=toolchain-member-operator/member-operator-metrics-monitor namespace=openshift-user-workload-monitoring prometheus=user-workload
```

## Verifying the Prometheus Targets

User Workload Monitoring is integrated into the OpenShift Web Console. 
Admins can check the target status in the [Observe > Targets menu](https://console-openshift-console.apps.sandbox-stage.gb17.p1.openshiftapps.com/monitoring/targets) and expect to see the metrics endpoint in "Up" state.