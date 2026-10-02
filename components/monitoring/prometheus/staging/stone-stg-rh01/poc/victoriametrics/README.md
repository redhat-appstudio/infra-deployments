# stone-stg-rh01 VictoriaMetrics PoC cleanup

This package runs in `appstudio-vm-poc`. Its operator also converts existing
`ServiceMonitor`, `PodMonitor`, `Probe`, and `ScrapeConfig` objects into VM scrape
objects **in the source namespaces**. It does not convert `PrometheusRule` or
`AlertmanagerConfig`; the `VMRule` in this package is a native PoC resource.

## When a source monitor is deleted

`operator.enable_converter_ownership: true` adds an owner reference from each
converted object to its source. Kubernetes garbage collection should delete the
converted object when the source is deleted. Check the corresponding VM object
in the same namespace and confirm it disappears. Investigate any copy that
remains instead of deleting unrelated VM objects.

## Before the first PoC sync

Save an inventory of existing VM scrape objects so teardown can distinguish
this PoC's conversions from objects managed by other operators:

```sh
oc get vmservicescrapes,vmpodscrapes,vmprobes,vmscrapeconfigs -A -o json > vm-scrapes-before-poc.json
```

Keep this snapshot outside the Git repository. Record any other VM operators
running in the cluster.

## After the first PoC sync

The staging Argo CD Application has automated pruning enabled. Argo CD prunes
resources it tracks; it does not make every resource absent from Git disappear.
The converter adds `argocd.argoproj.io/sync-options: Prune=false` to each
converted object. That is the sync-prune guard. Its `IgnoreExtraneous`
compare annotation only keeps generated objects from making the Application
appear out of sync; it does not prevent deletion.

Inspect a live converted object and confirm both annotations and its source
owner reference are present. Check the Argo CD Application resource tree and
sync result for unexpected prune candidates. If the annotations are absent,
resolve that before relying on normal syncs. `Prune=false` does not prevent
Kubernetes owner garbage collection, direct deletion, namespace deletion, or
CRD deletion.

## Retiring the PoC

1. In a separate GitOps change, set `operator.disable_prometheus_converter`
   to `true` in `operator-values.yaml` and sync. Keep the rest of the package
   deployed. Wait for the operator Deployment in `appstudio-vm-poc` to roll out
   and verify its pods have the converter disabled.
2. Inventory the same four VM scrape kinds across **all namespaces**, saving
   the result as `vm-scrapes-at-teardown.json` without overwriting the baseline.
   For each proposed deletion, inspect its namespace, name, UID,
   `metadata.ownerReferences`, and Argo annotations. The converted copies
   have a `monitoring.coreos.com` source owner and the converter-added
   `IgnoreExtraneous` / `Prune=false` annotations. Compare with the pre-PoC
   inventory and confirm that another operator does not manage it. If the
   baseline is unavailable or ownership is ambiguous, resolve that before
   deleting anything.
3. If the inventory confirms that this is the only VM converter and the
   candidate objects belong to this PoC, delete the reviewed set in one
   batch. Otherwise exclude objects another operator or VM stack uses. Use
   explicit kind, namespace, and name pairs from the reviewed inventory;
   do not use a cluster-wide `--all` deletion. Re-list the four kinds,
   check that the copies stay gone, and confirm no converted copy remains for
   the PoC. Disabling conversion first prevents this operator from recreating
   them while this check runs. Do not rely on another operator to recreate a
   deleted copy promptly: the v0.74.0 converter watches source monitors, so
   deleting a converted object alone does not trigger its reconciliation.
4. Remove `poc/victoriametrics` from the parent `stone-stg-rh01/kustomization.yaml`
   and sync with pruning. Verify the PoC's `VMAgent`, `VMCluster`, `VMAlert`,
   `VMRule`, operator Deployment and RBAC, namespace, and generated pods are
   gone. Check for remaining PoC PVCs and PVs before considering teardown
   complete; retained volumes need an explicit data deletion decision.
5. The operator CRDs use `Prune=false` and can be shared with other VM users.
   Inventory **all** VM custom resources and operators cluster-wide before
   removing CRDs in a separate decommission. Never delete the CRDs merely to
   clear this PoC's converted objects.

Normal Argo CD sync will not perform step 3 for this package because the
converted objects carry `Prune=false`. The source-owner reference handles
source deletion, while the explicit inventory and deletion handles PoC
retirement.

A later, freshly deployed VM operator will convert existing source monitors
again in each cluster if conversion is enabled and it watches their
namespaces. Check target parity after that rollout. This startup conversion
is different from relying on an already running operator to notice deletion
of a converted object. The PoC's native `VMRule` and `VMAlert` are not
recreated by monitor conversion.
