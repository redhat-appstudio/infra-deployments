# Monitoring component resource reductions in development

When using the **development** overlay for monitoring workloads, the following components are patched to use smaller resource requests/limits and (where applicable) single replica. This keeps dev clusters from running out of capacity.

## What is reduced

| Component | Patch file | Replicas | Resources (requests → limits) |
|-----------|------------|----------|-------------------------------|
| **Prometheus** (MonitoringStack) | `components/monitoring/prometheus/development/monitoringstack/monitoringstack-dev-resources-patch.yaml` | 2 → **1** | 500m / 16Gi → **200m / 2Gi** (limits: 2Gi) |
| **Grafana** (Grafana CR) | `components/monitoring/grafana/development/grafana-dev-resources-patch.yaml` | **1** | **100m / 256Mi** → 500m / 512Mi (staging/prod: 1 CPU / 4Gi) |
| **Grafana operator** (Subscription) | `components/monitoring/grafana/development/grafana-operator-dev-resources-patch.yaml` | — | 200m / 500Mi → **50m / 128Mi** (limits: **200m / 512Mi**; base: 400m / 1Gi) |
| **Cluster Observability operator** (Subscription) | `components/monitoring/prometheus/development/monitoringstack/observability-operator-dev-resources-patch.yaml` | — | — → **50m / 256Mi** requests, **200m / 1Gi** limits (base: 4Gi memory limit only) |

All of these apply only when the corresponding ApplicationSet is deployed from the **development** path (e.g. `monitoring-workload-prometheus` from `components/monitoring/prometheus/development`, `monitoring-workload-grafana` from `components/monitoring/grafana/development`). Ensure the development overlay is used for those apps (e.g. via `preview.sh`). For Prometheus, `preview.sh` adds `monitoringstack/` to the development kustomization so the stack and patches are included.

## What is not reduced (in this repo)

- **Logging** (`components/monitoring/logging`): `ClusterLogForwarder` collector is 1 CPU / 4Gi. **monitoring-workload-logging** is deleted in the dev overlay, so it is not deployed in dev unless you re-enable it.
- **Blackbox** (`components/monitoring/blackbox`): Workload comes from the o11y repo; no resource overrides in infra-deployments. If you deploy monitoring-blackbox in dev, consider reducing resources in the o11y config or via a local patch.
- **Kanary** (`components/monitoring/kanary`): Deployed from o11y repo; no dev-specific resource patches here.
- **Registry** (`components/monitoring/registry`): RBAC/config only in this repo; no deployable workload with resources to reduce.

If you need to reduce logging/blackbox/kanary further, add development overlays or patches in the appropriate component (or in the o11y repo for blackbox/kanary).
