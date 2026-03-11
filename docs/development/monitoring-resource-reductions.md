# Monitoring component resource reductions in development

When using the **development** overlay for monitoring workloads, the following components are patched to use smaller resource requests/limits and (where applicable) single replica. This keeps dev clusters from running out of capacity.

## What is reduced

| Component | Patch file | Replicas | CPU (req → limit) | Memory (req → limit) |
|-----------|------------|----------|--------------------|----------------------|
| **Prometheus** (MonitoringStack) | `components/monitoring/prometheus/development/monitoringstack/monitoringstack-dev-resources-patch.yaml` | 2 → **1** | 500m → **200m** | 16Gi → **2Gi** |
| **Grafana** (Grafana CR) | `components/monitoring/grafana/development/grafana-dev-resources-patch.yaml` | **1** | **100m** → 500m | **256Mi** → 512Mi |
| **Grafana operator** (Subscription) | `components/monitoring/grafana/development/grafana-operator-dev-resources-patch.yaml` | — | 200m → **50m** (limit: **200m**) | 500Mi → **128Mi** (limit: **512Mi**) |
| **Cluster Observability operator** (Subscription) | `components/monitoring/prometheus/development/monitoringstack/observability-operator-dev-resources-patch.yaml` | — | **50m** (limit: **200m**) | **256Mi** (limit: **1Gi**) |
| **Vector Tekton logs collector** (DaemonSet) | `components/vector-tekton-logs-collector/development/vector-helm-values.yaml` | — | 512m → **100m** (limit: **500m**) | 4096Mi → **512Mi** (limit: **512Mi**) |

All of these apply only when the corresponding ApplicationSet is deployed from the **development** path (e.g. `monitoring-workload-prometheus` from `components/monitoring/prometheus/development`, `monitoring-workload-grafana` from `components/monitoring/grafana/development`). Ensure the development overlay is used for those apps (e.g. via `preview.sh`).

> **Note:** `monitoring-workload-grafana` is **skipped by default** in the development overlay. Pass `--grafana` to `preview.sh` to enable it:
> ```
> ./hack/preview.sh --obo --grafana
> ```
> For Prometheus/OBO, `preview.sh --obo` adds `monitoringstack/` to the development kustomization so the MonitoringStack and its patches are included.

## What is not reduced (in this repo)

- **Logging** (`components/monitoring/logging`): `ClusterLogForwarder` collector is 1 CPU / 4Gi. **monitoring-workload-logging** is deleted in the dev overlay, so it is not deployed in dev unless you re-enable it.

Note: If you need to reduce logging/blackbox/kanary further, add development overlays or patches in the appropriate component (or in the o11y repo for blackbox/kanary).
