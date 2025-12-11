# Custom Kube State Metrics

This component deploys a dedicated `kube-state-metrics` instance for exposing Prometheus metrics about **custom resources** that aren't covered by the platform's built-in kube-state-metrics.

## Overview

**Problem**: OpenShift's built-in observability includes kube-state-metrics, but it only exports metrics for standard Kubernetes resources (Pods, Deployments, etc.). Custom resources from operators like Velero, Kueue, and Kyverno are not included.

**Solution**: Deploy a separate kube-state-metrics instance that:
- Uses `--custom-resource-state-only=true` to avoid duplicating platform metrics
- Configures CustomResourceStateMetrics for CRDs we care about
- Gets scraped by User Workload Monitoring (UWM)
- Metrics automatically flow: **UWM → COO Prometheus → RHOBS → Grafana**

## Architecture

```
┌────────────────────────────────────────────────┐
│  custom-kube-state-metrics                     │
│  • Watches CRs cluster-wide                    │
│  • Exposes metrics on :8080/metrics            │
│  • Self-metrics on :8081/metrics               │
└───────────────────┬────────────────────────────┘
                    │
                    ├─ ServiceMonitor (UWM scrapes every 30s)
                    │
                    ├─ ConfigMap (CustomResourceState config)
                    │
                    └─ RBAC (ClusterRole to list/watch CRs)

                    ↓
        User Workload Monitoring (UWM)
                    ↓
        COO Prometheus (federation)
                    ↓
        RHOBS → AppSRE Grafana
```

## Current Custom Resources

This component currently exports metrics for:

### Velero (Backup/DR)
- `velero_backup_status` - Backup phase (New, InProgress, Completed, Failed, etc.)
- `velero_backup_completion_timestamp` - When backup completed
- `velero_backup_expiration_timestamp` - When backup expires
- `velero_backup_total_items` - Number of items to backup
- `velero_backup_items_backed_up` - Number of items successfully backed up
- `velero_restore_status` - Restore phase

### Kueue (Workload Management)
- `kueue_clusterqueue_pending_workloads` - Pending workloads in ClusterQueue
- `kueue_clusterqueue_admitted_workloads` - Admitted workloads in ClusterQueue
- `kueue_clusterqueue_reserving_workloads` - Reserving workloads in ClusterQueue
- `kueue_localqueue_pending_workloads` - Pending workloads in LocalQueue
- `kueue_localqueue_admitted_workloads` - Admitted workloads in LocalQueue

### Kyverno (Policy Engine)
- `kyverno_clusterpolicy_ready` - ClusterPolicy ready status (1 = ready, 0 = not ready)
- `kyverno_clusterpolicy_rule_count` - Number of rules in ClusterPolicy
- `kyverno_clusterpolicy_validation_failure_action` - Action on validation failure (Audit/Enforce)
- `kyverno_policy_ready` - Namespaced Policy ready status
- `kyverno_policy_rule_count` - Number of rules in namespaced Policy

## SOP: Adding a New Custom Resource

Follow these steps to add metrics for a new custom resource type:

### Step 1: Identify the CR Structure

First, inspect an example of the CR to understand its structure:

```bash
# Get an example CR
kubectl get <resource-type> <name> -o yaml

# Example for CertificateRequests
kubectl get certificaterequest -n cert-manager example-cert -o yaml
```

Identify:
- API group, version, kind
- Relevant fields in `.status` or `.spec` to expose as metrics
- Whether the resource is cluster-scoped or namespaced

### Step 2: Update CustomResourceState Config

Edit `base/custom-resource-state-config.yaml` and add a new resource block under the `spec.resources` list:

```yaml
# Template
- groupVersionKind:
    group: "<api-group>"        # e.g., "cert-manager.io"
    version: "<version>"         # e.g., "v1"
    kind: "<Kind>"              # e.g., "CertificateRequest"
  labelsFromPath:
    name: [metadata, name]
    namespace: [metadata, namespace]  # omit for cluster-scoped resources
  metrics:
    - name: "<metric_name>"     # e.g., "certmanager_certificaterequest_ready"
      help: "<description>"
      each:
        type: Gauge             # or StateSet, Info
        gauge:
          path: [status, someField]
          nilIsZero: true       # treat missing field as 0
```

#### Metric Types

**Gauge**: Numeric value
```yaml
- name: "velero_backup_total_items"
  help: "Total number of items in backup"
  each:
    type: Gauge
    gauge:
      path: [status, progress, totalItems]
      nilIsZero: true
```

**StateSet**: Enumerated states (creates one time series per possible value)
```yaml
- name: "velero_backup_status"
  help: "Current status/phase of Velero backup"
  each:
    type: StateSet
    stateSet:
      labelName: phase
      path: [status, phase]
      list: [New, InProgress, Completed, Failed]
```

**Info**: Expose labels only (value always 1)
```yaml
- name: "certmanager_certificate_info"
  help: "Certificate information"
  each:
    type: Info
    info:
      labelsFromPath:
        issuer: [spec, issuerRef, name]
        common_name: [spec, commonName]
```

**Counting Array Elements**: Use `valueFrom: [length]`
```yaml
- name: "kyverno_clusterpolicy_rule_count"
  help: "Number of rules in ClusterPolicy"
  each:
    type: Gauge
    gauge:
      path: [spec, rules]
      valueFrom: [length]
      nilIsZero: true
```

### Step 3: Update RBAC

Edit `base/rbac.yaml` and add the necessary permissions:

```yaml
# Add to ClusterRole rules
- apiGroups: ["cert-manager.io"]
  resources:
    - certificates
    - certificaterequests
    - issuers
    - clusterissuers
  verbs: ["list", "watch"]
```

### Step 4: Test Locally

```bash
# Build and validate Kustomize
kustomize build components/monitoring/custom-kube-state-metrics/development

# Apply to development cluster
kustomize build components/monitoring/custom-kube-state-metrics/development | kubectl apply -f -

# Check pod is running
kubectl get pods -n custom-kube-state-metrics

# Port-forward to test metrics
kubectl port-forward -n custom-kube-state-metrics svc/custom-kube-state-metrics 8080:8080

# Verify metrics are exposed
curl localhost:8080/metrics | grep <your_metric_name>
```

### Step 5: Verify in User Workload Monitoring

After deploying, verify metrics are being scraped by UWM:

**Via OpenShift Console:**
1. Navigate to **Observe → Metrics**
2. Switch to **User Workload** context
3. Query: `{__name__=~"<your_metric_prefix>.*"}`

**Via CLI:**
```bash
# Get UWM Prometheus route
oc get route -n openshift-user-workload-monitoring

# Query metrics
curl -k -H "Authorization: Bearer $(oc whoami -t)" \
  "https://<prometheus-route>/api/v1/query?query=<your_metric_name>"
```

### Step 6: Create PR

Follow the standard preview mode workflow:
1. Create feature branch
2. Commit changes
3. Run `./hack/preview.sh` (if testing on personal cluster)
4. Create PR with description of what CRs you're adding

## Troubleshooting

### Metrics not appearing

1. **Check pod logs:**
   ```bash
   kubectl logs -n custom-kube-state-metrics deployment/custom-kube-state-metrics
   ```

2. **Verify RBAC:**
   ```bash
   # Check if ServiceAccount has permissions
   kubectl auth can-i list <resource> --as=system:serviceaccount:custom-kube-state-metrics:custom-kube-state-metrics
   ```

3. **Check ServiceMonitor:**
   ```bash
   kubectl get servicemonitor -n custom-kube-state-metrics
   kubectl describe servicemonitor custom-kube-state-metrics -n custom-kube-state-metrics
   ```

4. **Verify CR exists:**
   ```bash
   kubectl get <resource-type>
   ```

### Configuration errors

If the pod fails to start with config errors:

```bash
# Get detailed error
kubectl describe pod -n custom-kube-state-metrics -l app.kubernetes.io/name=custom-kube-state-metrics

# Common issues:
# - Invalid YAML syntax in ConfigMap
# - Incorrect path in 'gauge.path' or 'stateSet.path'
# - Missing 'nilIsZero' for fields that might not exist
```

### Testing config changes quickly

```bash
# Edit ConfigMap directly (for testing only, don't commit)
kubectl edit configmap custom-kube-state-metrics-config -n custom-kube-state-metrics

# Restart deployment to reload config
kubectl rollout restart deployment/custom-kube-state-metrics -n custom-kube-state-metrics

# Watch rollout
kubectl rollout status deployment/custom-kube-state-metrics -n custom-kube-state-metrics
```

## Resources

- [kube-state-metrics CustomResourceState docs](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/customresourcestate-metrics.md)
- [Konflux Monitoring Architecture](../prometheus/README.md)
- [OpenShift User Workload Monitoring](https://docs.openshift.com/container-platform/latest/monitoring/enabling-monitoring-for-user-defined-projects.html)

## Examples

See `base/custom-resource-state-config.yaml` for complete examples of:
- Velero Backups and Restores
- Kueue ClusterQueues and LocalQueues
- Kyverno ClusterPolicies and Policies
