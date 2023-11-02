## RHTAP Central Monitoring
The RHTAP monitoring solution is based on three Prometheus instances deployed to each
Production and Staging host and member clusters. Each cluster writes a subset of the
metrics it generates into Observatorium (RHOBS), marking each metric with a label
indicating its cluster of origin.

Observatorium holds metrics for RHTAP's tenant in two RHOBS environments – Production
and one Staging. The metrics collected for each of those environments are available
for presentation via the AppSRE Grafana instance.

```mermaid
%%{init: {'theme':'forest'}}%%
flowchart BT
  services(RHTAP Services) --> |scrape|UWM
  pods(kubelet, pods, etc.) --> |scrape|Platform
  UWM(User Workload Monitoring) --> |federate| MS(Monitoring Stack)
  Platform --> |federate|MS(Monitoring Stack)
  MS --> |remote-write|rhobs(Observatorium)

  services2(RHTAP Services) --> |scrape|UWM2
  pods2(kubelet, pods, etc.) --> |scrape|Platform2
  UWM2(User Workload Monitoring) --> |federate| MS2(Monitoring Stack)
  Platform2(Platform) --> |federate|MS2(Monitoring Stack)
  MS2 --> |remote-write|rhobs(Observatorium)

  rhobs --> |scrape|grafana(AppSRE Grafana)

  subgraph member[RHTAP Member Clusters]
    services
    pods
    subgraph "cmo member"[Cluster Monitoring Operator]
      UWM
      Platform
    end
    MS
  end

  subgraph host["RHTAP Host Cluster"]
    services2
    pods2
    subgraph "cmo host"[Cluster Monitoring Operator]
      UWM2
      Platform2
    end
    MS2
  end

  style member color:blue
  style host color:red
```
### Data Plane Clusters Prometheus Instances
We use the
[Openshift-provided](https://docs.openshift.com/container-platform/4.12/monitoring/monitoring-overview.html)
Prometheus deployments, Platform and user-workload-monitoring (UWM), alongside a
Prometheus instance deployed by the RHOBS
[Observability Operator](https://github.com/rhobs/observability-operator).

#### Platform Prometheus
Mainly scrapes generic metrics produced by built-in exports such as cAdvisor,
kube-state-metrics, etc.

#### User Workload Monitoring (UWM) Prometheus
Scrapes custom metrics provided by services deployed by the different RHTAP teams, and
collected by Service Monitors, also provided by the teams.

In Production and Staging, UWM Prometheus is enabled using OCM (since it maintains the
Prometheus configurations).  
The retention is set to 3 days and the retention size is set to 10GiB.  
It is defined in `components/monitoring/prometheus/base/uwm-config/uwm-config.yaml`
and it is controlled by ArgoCD.


In Development it's enabled without deploying a ConfigMap using ArgoCD 
(The ConfigMap is created automatically when UWM is enabled)  

The retention is set to default (24h).  
To configure the retention for development environment, edit the 
`user-workload-monitoring-config` ConfigMap in `openshift-user-workload-monitoring` namespace.  
For example:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: user-workload-monitoring-config
  namespace: openshift-user-workload-monitoring
data: 
  config.yaml: | 
    prometheus: 
      retention: 2d 
      retentionSize: 1GiB
```

#### Observability Operator (OBO) Prometheus
Federates the Platform and UWM Prometheus instances.

There are limitations for both built-in Prometheus instances that do not allow us to
use them to write metrics to RHOBS:

- The configurations of the Platform Prometheus instance are owned by SRE Platform, thus
we cannot configure this instance to remote-write.
- Service Monitors for The UWM Prometheus instance are limited for scraping metrics
from the same namespace in which the Service Monitor is defined. it means that this
instance cannot federate the Platform Prometheus instance, thus cannot hold all data
needed to be exported (it also cannot remote-write metrics coming from different
namespaces).

For those reasons, another instance is needed to federate the other instances, and
write metrics to RHOBS.

This instance collects selected metrics from Platform Prometheus and UWM Prometheus, and
remote-writes selected labels for those metrics to RHOBS, which in turn, makes the
metrics accessible to AppSRE Grafana.

This Prometheus instance is deployed using a MonitoringStack custom resource provided
by the Observability Operator. This operator is installed by default in Production and Staging clusters.  
In Development clusters, it's not installed by default to prevent conflicts with other deployments. 
It can be installed and configured in development by using the `--obo/-o` flags.  
For example:

`./hack/bootstrap-cluster.sh preview --obo`  
`./hack/bootstrap-cluster.sh preview -o`

#### Federation and Remote-write

Through Federation and remote-write configurations, only a subset of the metrics and
the labels collected within the data plane clusters reach RHOBS. For that reason, it
might be that metrics that are visible via the OCP web console (under Observe -->
Metrics) do not reach RHOBS and are not visible in AppSRE Grafana.

The Platform Prometheus instance monitors a wide variety of resources which are, in
nature, of an unbound cardinality (e.g. containers, pods, jobs). Consequently, it
generates a substantial amount of metrics time series that cannot all be forwarded to
RHOBS. For that reason, we only allow a very small subset of the metrics it scrapes
to be federated by the OBO Prometheus instance (and later remote-written to RHOBS).

The UWM Prometheus instance, on the other hand, generates a very few time series by
default, and the metrics it is configured to scrape for services are generally of
constant cardinality. I.e. the number of time series stored for a given service does
not grow based on the service load. For this reason, we allow all metrics it scrapes
to be federated by the OBO Prometheus instance (and reach RHOBS).

All **metrics** reaching the OBO instance are remote-written to RHOBS, but not all
**labels** are. This means that it might be that time series visible in AppSRE Grafana
will not include some of the labels the same time series have on the data plane
clusters. The OBO instance is configured to remote-write only specific labels, and if
the presence of a new label is required in alerting rules or AppSRE Grafana dashboards,
then this label should be added to the configurations.

##### Troubleshooting Missing Metrics and Labels

There are a few steps to follow when specific metrics or labels are required for new
alerting rules or Grafana dashboards, but are not present in AppSRE Grafana.

> **_NOTE:_**  While we remote-write the metrics to RHOBS rather than to AppSRE Grafana,
we don't have an easy way to directly confirm whether metrics are reaching RHOBS or not.
Instead, we check AppSRE Grafana, assuming no metrics/labels are dropped between RHOBS
and AppSRE Grafana. Nevertheless, such drops are possible, although far less probable
comparing to MonitoringStack misconfigurations.

If the metric is missing altogether:

1. Verify that the metric does exist inside its expected cluster of origin by querying
   for it on the Observe --> Metrics screen on the OCP web console.
    * If it doesn't, further troubleshoot the service monitor expected to collect the
      metric and the Kubernetes resource expected to generate it.
2. While querying for the metric, check the value of its `prometheus` label.
    * if the value is `openshift-monitoring/k8s`, it means it's being collected by the
      Platform Prometheus instance. As we only federate specific metrics from this
      instance, the metric needs to be added to the `match` list under the
      `appstudio-federate-smon` ServiceMonitor resource within the
      [MonitoringStack configurations].
    * if the label value is different, reach out to the o11y team on [Slack][o11y-slack]
3. Once added, the metric should be federated by the OBO instance and remote-written to
   RHOBS.

If the metric is present, but labels are missing:

1. Verify that the labels do exist when querying for the metric through the OCP web
   console.
    * If not, further troubleshoot the resource that should export or instrument
      the metric.
2. Add the missing labels to the `LabelKeep` action's `regex` field within the
   `MonitoringStack` resource definition in the [MonitoringStack configurations].
3. Once added, the label should be remote-written by the OBO instance to RHOBS.

For further assistance, reach out to the o11y team on [Slack][o11y-slack].

[MonitoringStack configurations]: base/monitoringstack/monitoringstack.yaml
[o11y-slack]: https://redhat-internal.slack.com/archives/C04FDFTF8EB
