## Data Plane Clusters Prometheus Instances
We use the Openshift-provided Prometheus deployments, platform and
user-workload-monitoring (UWM), alongside a Prometheus instance deployed by the RHOBS
[Observability Operator](https://github.com/rhobs/observability-operator).

### Platform Prometheus
Scrapes generic metrics produced by cAdvisor, kube-state-metrics, etc.

### User Workload Monitoring (UWM) Prometheus
Scrapes custom metrics provided by services deployed by the different teams.

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

### Observability Operator (OBO) Prometheus
Federates the Platform Prometheus instance.

There are limitations for both built-in Prometheus instances that do no allow us to
use them to write metrics to RHOBS:

- The configurations of the Platform Prometheus instance are owned by SRE Platform, thus
we cannot configure this instance to remote-write.
- Service Monitors for The UWM Prometheus instance are limited for scraping metrics
from the same namespace in which the service monitor is defined. it means that this
instance cannot federate the Platform Prometheus instance.

For those reasons, another instance is needed that will federate the other
instances, and will later on write metrics to RHOBS.

At the moment, only the Platform Prometheus instance is being federated.

The Observability Operator is installed by default in production and staging clusters.
In Development it's installed and configured using ArgoCD.
