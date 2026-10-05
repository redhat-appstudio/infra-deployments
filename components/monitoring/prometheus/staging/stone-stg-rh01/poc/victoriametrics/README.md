# stone-stg-rh01 VictoriaMetrics PoC

This package runs in `appstudio-vm-poc`. Its operator also converts existing
`ServiceMonitor`, `PodMonitor`, `Probe`, and `ScrapeConfig` objects into VM scrape
objects **in the source namespaces**. It does not convert `PrometheusRule` or
`AlertmanagerConfig`; the `VMRule` in this package is a native PoC resource.

The native Tekton `VMServiceScrape` objects scrape the pipelines controller
and pipeline metrics exporter directly. The exporter also has a converted
platform monitor, but its `job` label differs from the Tekton Prometheus label.
The native scrapes and controller network policy are Git-managed and pruned
with the PoC. The Tekton Prometheus proxy and RHOBS federation monitors are
not used as scrape targets by this PoC.

## Direct scrape coverage

The PoC VMAgent writes only to its own `vminsert`; it has no RHOBS remote-write
endpoint. Existing RHOBS MonitoringStacks and their deliveries are unchanged.
The agent uses 30 seconds for scrapes without an interval and raises faster
source intervals to 30 seconds. It preserves longer source intervals such as
kaexporter's 300 seconds and its 180-second scrape timeout. Use the resulting
per-target cadence and both VMAgent replicas per shard when estimating
ingestion and retention capacity.
The VM operator converts the cluster's `monitoring.coreos.com` ServiceMonitors
and PodMonitor, which cover the platform and UWM targets and the Tekton pipeline
metrics exporter. `selectAllByDefault` deliberately includes every converted
monitor, including a few source monitors excluded from the current Prometheus
selectors; account for any extra samples in the capacity check. The native
`VMServiceScrape` covers all Tekton controller pods because the platform
controller monitor drops the `tekton_pipelines_controller_*` series. A second
native scrape keeps the Tekton exporter series under its original `job` label;
the converted platform scrape keeps the platform `job` label. The old Tekton
Prometheus `prometheus-self` target is replaced by VMAgent self-scraping.

Some converted platform monitors retain Prometheus file paths for the service
CA, kubelet CA, and metrics client certificate. VMAgent mounts the automatic
`openshift-service-ca.crt` ConfigMap and two local copies at those exact paths.
The `vm-poc-scrape-credential-sync` CronJob copies the client certificate and
kubelet CA from `openshift-monitoring` every minute, with read access limited to
those two source objects. It places no credential data in Git. The VMAgent pods
may wait for the first successful sync before their volumes become available.
Check the CronJob and its generated Secret and ConfigMap before diagnosing
missing platform targets. NetworkPolicies permit the PoC pods to reach target
namespaces whose existing policies allow only the current Prometheus pods.

After deployment, compare the union of `/targets` from **all** VMAgent shards
with live platform, UWM, and Tekton Prometheus target URLs and labels. Check
that the controller and pipeline exporter are both present and healthy, and
investigate every missing or down target. The source snapshot on 2026-10-05 had
524 platform targets, 95 UWM targets (4 already down), and five healthy Tekton
application targets; the sixth Tekton target was its own Prometheus. Also
check that the mirrored certificate and CA update on source rotation, remote
write errors remain stable, and VM queries contain the expected series. A
render or dry-run cannot establish live target parity.

## Capacity gate before staging sync

Record active series, scraped samples/second, new-series churn, and the source
scrape interval separately for platform, UWM, Tekton, and any extra converted
targets. Group targets with different intervals separately; do not treat every
target as a 30-second scrape. Reconcile these source figures with the PoC's
live target inventory before using them for sizing.

For each group, estimate one agent replica's samples/second at its effective
interval, `max(30s, source interval)`. Use the measured source scrape rate
scaled by `source interval / effective interval`, then compare it with active
series divided by effective interval. **Multiply the resulting rate by two**:
each of the two VMAgent replicas per shard scrapes and remote-writes its own
raw samples. The three shards partition targets; they do not multiply this
rate by three. The replicas use identical external labels, so count distinct
new series and index growth once, then confirm that identity in live data.
Include the native Tekton scrapes and any overlapping converted scrape in
the measured totals.

Project four days (72-hour retention plus a retention cycle) using
`345600 * (2 * sum(single-replica samples/second) * measured VM bytes/raw sample
+ distinct new-series/second * measured index bytes/new series)`. Measure both
byte factors on VM with both replicas writing, including transient disk used
before background deduplication; do not derive the sample rate from
query-visible samples. The 30-second storage/select deduplication can reduce
query-visible samples and later disk use, but both copies are ingested and
initially written. Size the busiest shard, not just the 20-shard average.
Require at least 20% free space on every storage PVC after the four-day
projection. The initial 20Gi PVC is a test default, not a capacity assertion.

On both replicas that own each slow target, inspect `/targets` repeatedly to
confirm the effective interval and observed scrape timestamps. For
kaexporter, expect roughly one scrape per 300 seconds on **each** replica.
Query one stable `up` series through vmselect with
`count_over_time(up{job="<job>",instance="<instance>"}[1h])`: about 12
query-visible samples indicates one retained sample per scrape cycle; about
24 indicates both replicas remain visible. Compare this with the two
replicas' scrape counts and investigate missed scrapes or unexpected series
labels. A 30-second deduplication window may not merge 300-second scrapes
whose timestamps fall in different 30-second buckets.

Before applying to staging, check actual shard placement and the largest
number of storage pods that would land on any surviving worker after one
worker is lost (at least `ceil(20 / (eligible workers - 1))`). Include their
projected CPU/RAM and all other workloads on that worker; require at least
50% spare allocatable CPU and RAM in the worker-loss case. Repeat the disk
check with measured per-shard skew and expand PVCs if needed.

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
