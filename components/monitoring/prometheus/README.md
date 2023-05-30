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
In Development it's enabled by deploying a configmap using ArgoCD.

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
