---
name: deploy-custom-ksm
description: Deploy custom-kube-state-metrics and its dependencies to a local OpenShift development cluster
user-invocable: true
---

Deploy the custom-kube-state-metrics monitoring stack to the user's local OpenShift development cluster. This involves checking prerequisites, deploying dependent components in order, and validating each step.

## Prerequisites

1. Verify cluster connectivity: run `oc whoami` and `oc cluster-info` to confirm the user is logged into an OpenShift cluster.
2. If the cluster URL contains "sandbox", this is a sandbox cluster — safe to proceed without confirmation. Otherwise, confirm with the user that this is not a shared staging/production cluster before proceeding.

## Deployment steps (execute in order)

### Step 1: Namespace
Ensure the `appstudio-monitoring` namespace exists. Create it if it doesn't:
```
oc create namespace appstudio-monitoring --dry-run=client -o yaml | oc apply -f -
```

### Step 2: Platform Monitoring and UWM
1. Verify platform monitoring is running: check that pods exist in the `openshift-monitoring` namespace.
2. Check if UWM is enabled by inspecting the `cluster-monitoring-config` ConfigMap in `openshift-monitoring` for `enableUserWorkload: true`.
   - If not enabled, enable it by patching the ConfigMap.
   - Wait for the `prometheus-user-workload` pods to appear in `openshift-user-workload-monitoring`.

### Step 3: Custom kube-state-metrics
Apply the component:
```
kubectl apply -k components/monitoring/custom-kube-state-metrics/base/
```

### Step 4: Add test metric
Patch the `custom-kube-state-metrics-config` ConfigMap in `appstudio-monitoring` to add a test metric for OpenShift Routes (present on every OpenShift cluster). Add this resource entry to the `spec.resources` list in the `custom-resource-state.yaml` data key:
```yaml
- groupVersionKind:
    group: "route.openshift.io"
    version: "v1"
    kind: "Route"
  labelsFromPath:
    route: [metadata, name]
    namespace: [metadata, namespace]
  metrics:
    - name: "openshift_route_generation"
      help: "The metadata generation of an OpenShift Route"
      each:
        type: Gauge
        gauge:
          path: [metadata, generation]
          nilIsZero: true
```
Note: Do not use the `Info` metric type — UWM Prometheus rejects it with `invalid metric type "info"`. Use `Gauge` instead.
Also patch the `custom-kube-state-metrics` ClusterRole to add RBAC for Routes:
```yaml
- apiGroups: ["route.openshift.io"]
  resources: ["routes"]
  verbs: ["list", "watch", "get"]
```
After patching, restart the deployment to pick up the new config:
```
oc rollout restart deployment/custom-kube-state-metrics -n appstudio-monitoring
oc rollout status deployment/custom-kube-state-metrics -n appstudio-monitoring --timeout=60s
```

### Step 5: Validation
1. Verify the deployment is running: `oc rollout status deployment/custom-kube-state-metrics -n appstudio-monitoring`
2. Verify the ServiceMonitor exists: `oc get servicemonitor custom-kube-state-metrics -n appstudio-monitoring`
3. Verify metrics are being served: port-forward to the `custom-kube-state-metrics` service on port 8080 and curl `localhost:8080/metrics`. Check that `kube_customresource_openshift_route_generation` metrics appear in the output.
4. Query UWM Prometheus for the metrics: port-forward to `prometheus-user-workload` pod in `openshift-user-workload-monitoring` on port 9090 and query the Prometheus API (e.g. `curl localhost:9090/api/v1/query?query=kube_customresource_openshift_route_generation`). If metrics are not yet available, note that scraping may take up to 30 seconds based on the ServiceMonitor interval.
5. Report the status of each check to the user.

## Error handling
- If any step fails, stop and report the error with context. Do not proceed to the next step.
- If UWM pods do not appear, check `oc get pods -n openshift-user-workload-monitoring` and report pod statuses.
- If the custom-kube-state-metrics deployment does not become ready, check `oc get pods -n appstudio-monitoring` and report pod statuses.
