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
    some_label: [status, someStringField]  # string fields go here as labels, NOT in the gauge path
  metrics:
    - name: "<metric_name>"
      help: "<description>"
      each:
        type: Gauge
        gauge:
          path: [status, someNumericField]
          nilIsZero: true
```

> **IMPORTANT:** `labelsFromPath` must be placed at the **resource level** (sibling of
> `groupVersionKind` and `metrics`), not inside individual metrics. Labels defined here
> are shared across all metrics for that resource. String fields that you want to filter
> or group by should be added here as labels — the gauge `path` must always point to a
> numeric field. See [Constraints](#constraints) for details.

### 2. Update RBAC

Add a rule to the ClusterRole in `base/rbac.yaml` for the new API group:

```yaml
- apiGroups: ["<api-group>"]
  resources: ["<plural-resource-name>"]
  verbs: ["list", "watch", "get"]
```

### 3. Validate

#### Kustomize build

```bash
kustomize build components/monitoring/custom-kube-state-metrics/base
```

#### Local Docker test — staging only

Test that your metric actually produces data by running kube-state-metrics locally
against a staging cluster. This does **not** modify the cluster — it only reads
data via the Kubernetes API.

> **NOTE:** You must be logged into a **staging** cluster (e.g. `stone-stg-rh01` or
> `stone-stage-p01`). The Docker container runs on your local machine and only
> queries the cluster API (`list`/`watch`) — no resources are created, modified,
> or deleted on the cluster. Your token may have broader permissions, but
> kube-state-metrics only performs read operations.

1. Ensure Docker is installed:

```bash
docker --version || sudo dnf install -y docker && sudo systemctl start docker
```

2. Get a login token from the [Konflux Portal](https://konflux.pages.redhat.com/konflux-portal/developer/konflux-clusters.html?env=staging&type=multi-tenant)
   (VPN/secure connection required), then log in and export a standalone kubeconfig:

```bash
oc login --token=<your-token> --server=https://api.<staging-cluster>:6443
oc config view --minify --flatten > /tmp/ksm-test-kubeconfig
```

3. Extract the inner config from the ConfigMap wrapper (strips the header and indentation):

```bash
sed '1,/custom-resource-state.yaml: |/d' \
  components/monitoring/custom-kube-state-metrics/staging/custom-resource-state-config.yaml \
  | sed 's/^    //' > /tmp/ksm-test-config.yaml
```

4. Run kube-state-metrics locally (must match the deployed version — v2.11.0):

```bash
docker run --rm -d \
  --name ksm-test \
  -p 8080:8080 \
  -v /tmp/ksm-test-kubeconfig:/kubeconfig:ro \
  -v /tmp/ksm-test-config.yaml:/etc/config/custom-resource-state.yaml:ro \
  registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.11.0 \
  --kubeconfig=/kubeconfig \
  --custom-resource-state-config-file=/etc/config/custom-resource-state.yaml \
  --custom-resource-state-only=true
```

5. Wait ~15 seconds, then check for errors and verify metrics:

```bash
# Check for errors (no output = good)
docker logs ksm-test 2>&1 | grep -i err

# Query your metric (replace <your_metric_name> with the actual name)
curl -s http://localhost:8080/metrics | grep "<your_metric_name>"
```

6. Clean up:

```bash
docker stop ksm-test
rm /tmp/ksm-test-kubeconfig /tmp/ksm-test-config.yaml
```

#### Staging cluster verification

After the staging PR is merged and synced by ArgoCD, verify the metric is scraped
correctly by UWM Prometheus before promoting to production. Query the new metric as
`kube_customresource_<new_metric_name>` using either:

- [Grafana Stage](https://grafana.stage.devshift.net/explore) (Explore view)
- The **Metrics** service inside the staging OpenShift web console

If the metric appears with expected labels and values, it is ready for production.

## Constraints

We run kube-state-metrics **v2.11.0** and our metrics are scraped by UWM
(User Workload Monitoring) Prometheus. This combination imposes the following
constraints on custom resource metrics:

- **Gauge only.** UWM Prometheus silently drops `StateSet` and `Info` metric types.
  Only `Gauge` is scraped and queryable. Always use `type: Gauge`.
- **No arbitrary string values as gauge data.** Gauge requires a value that KSM can
  convert to a float. Booleans (`true`/`false`), timestamps, and numbers work fine.
  Arbitrary strings (e.g. `status.conditions[].status` which is `"True"`, `"False"`,
  `"Unknown"`) cause `strconv.ParseFloat` errors and emit zero data points. To work
  around this, use the string as a **label** (via `labelsFromPath`) and point the
  gauge `path` at a numeric field instead.
- **No `valueMap`.** The `valueMap` option (which maps strings to numbers) does not
  exist in v2.11.0. It was introduced in a later version.

## References

- [CustomResourceState metrics docs (v2.11.0)](https://github.com/kubernetes/kube-state-metrics/blob/v2.11.0/docs/customresourcestate-metrics.md)
- See `base/custom-resource-state-config.yaml` for examples
