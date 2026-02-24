# perf-team-prometheus-reader

Component used by Konflux Perf&Scale team

## Service Accounts

- `perf-team-prometheus-reader-cluster-sa`: Used to read monitoring data from cluster Prometheus.
- `perf-team-prometheus-reader-oomcrash-sa`: Used by the [oomkill-and-crashloopbackoff-detector](https://github.com/redhat-appstudio/perfscale/tree/main/tools/oomkill-and-crashloopbackoff-detector) tool to monitor and detect OOMKills and crashloops across the cluster.
