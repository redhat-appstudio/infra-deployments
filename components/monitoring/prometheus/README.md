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
Scrapes generic metrics produced by built-in exports such as cAdvisor,
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

This instance collect selected metrics from Platform Prometheus and UWM Prometheus, and
remote-writes selected labels for those metrics to RHOBS, which in turn, makes the
metrics accessible to AppSRE Grafana.

This Prometheus instance is deployed using a MonitoringStack custom resource provided
by the Observability Operator. This operator is installed by default in Production and Staging clusters.  
In Development clusters, it's not installed by default to prevent conflicts with other deployments. 
It can be installed and configured in development by using the `--obo/-o` flags.  
For example:  
`./hack/bootstrap-cluster.sh preview --obo`  
`./hack/bootstrap-cluster.sh preview -o`
