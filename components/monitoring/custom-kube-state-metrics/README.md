# Custom Kube State Metrics

Create by issue [PVO11Y-5051](https://uat-3-2-redhat.atlassian.net/browse/PVO11Y-5051).

Deploys a dedicated `kube-state-metrics` instance for exposing Prometheus metrics from custom resources.

Uses `--custom-resource-state-only=true` to avoid duplicating platform metrics.

## Adding a New Custom Resource

### 1. Update the config

Edit `base/custom-resource-state-config.yaml` and add a new entry under `spec.resources`:

```yaml
- groupVersionKind:
    group: "<api-group>"
    version: "<version>"
    kind: "<Kind>"
  labelsFromPath:
    name: [metadata, name]
    namespace: [metadata, namespace]  # omit for cluster-scoped resources
  metrics:
    - name: "<metric_name>"
      help: "<description>"
      each:
        type: Gauge  # or StateSet, Info
        gauge:
          path: [status, someField]
          nilIsZero: true
```

### 2. Update RBAC

Add a rule to the ClusterRole in `base/rbac.yaml` for the new API group:

```yaml
- apiGroups: ["<api-group>"]
  resources: ["<plural-resource-name>"]
  verbs: ["list", "watch", "get"]
```

### 3. Validate

```bash
kustomize build components/monitoring/custom-kube-state-metrics/base
```

## References

- [CustomResourceState metrics docs](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/customresourcestate-metrics.md)
- See `base/custom-resource-state-config.yaml` for examples
