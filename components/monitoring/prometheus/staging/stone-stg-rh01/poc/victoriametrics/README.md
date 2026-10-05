# stone-stg-rh01 VictoriaMetrics PoC

This package runs in `appstudio-vm-poc`. VMAgent scrapes user workload
monitoring (UWM) and Tekton targets directly, and collects platform metrics
from platform Prometheus through `/federate`. It writes only to this PoC's
`vminsert`. Existing RHOBS MonitoringStacks and their deliveries are unchanged.

The operator converts existing `ServiceMonitor`, `PodMonitor`, `Probe`, and
`ScrapeConfig` objects into VM objects in their source namespaces. The agent's
selectors use the UWM namespace and monitor labels, so converted platform
monitors can exist without being scraped directly. Conversion of
`PrometheusRule` and `AlertmanagerConfig` is disabled; this package provides
its own `VMRule` and `VMAlert`.

## Collection paths

- **Platform:** `platform-federation.yaml` scrapes
  `https://prometheus-k8s.openshift-monitoring.svc:9091/federate` with the
  VMAgent service account token. A RoleBinding in `openshift-monitoring`
  grants that account the existing `cluster-monitoring-metrics-api` Role.
  The mounted OpenShift service CA verifies the server. The 2-minute job has
  a 90-second timeout and a 2GiB uncompressed response limit. Stream parsing
  avoids buffering the full response; disabling stale markers avoids retaining
  the previous response in agent memory. Removed platform series therefore
  have no explicit stale marker from this scrape. The source `prometheus` and
  `prometheus_replica` labels are dropped so the two agent replicas write the
  same series identity.
- **UWM:** Converted VM scrape objects are selected with the live UWM
  Prometheus monitor and namespace selectors. Unspecified and faster source
  intervals become 30 seconds; longer intervals remain, including
  kaexporter's 300-second interval. The service CA mount also satisfies the
  selected `openshift-logging/instance` monitor. No platform metrics client
  certificate or kubelet CA is copied into the PoC. Some selected UWM
  monitors reference their own namespace Secrets; verify the operator
  resolves them and their targets are healthy.
- **Tekton:** The two native `VMServiceScrape` objects target the pipelines
  controller and pipeline metrics exporter directly. Their current platform
  Prometheus jobs, `tekton-pipelines-controller` and
  `pipeline-metrics-exporter`, are excluded from federation to avoid
  collecting the same endpoints twice. Recheck those job labels if Tekton's
  monitors change. The Tekton Prometheus proxy is not a scrape target.

The network policies in this package permit the direct UWM and Tekton
connections where the source namespaces restrict monitoring ingress. The
PoC has no RHOBS remote-write destination.

## Validation after staging sync

Compare the direct UWM and Tekton targets on every VMAgent shard with the
source Prometheus target URLs, labels, and health. Investigate missing targets
and new failures; the source UWM snapshot on 2026-10-05 already had four down
targets. Verify both native Tekton scrapes and the selected UWM monitors,
including Secret-backed authentication and kaexporter's effective 300-second
cadence. Check that no converted platform monitor is selected for direct
scraping.

The platform proxy appears as **one target on one shard pair**, not as
hundreds of original platform targets. On both replicas of that shard, confirm
`up`, the 2-minute cadence, scrape duration below 90 seconds, response size
below 2GiB, comparable samples after metric relabeling, stable remote-write
errors, and persistent queue drain. Query representative non-Tekton platform
series through vmselect and compare their labels and current values with
platform Prometheus. Also watch platform Prometheus CPU, memory, and query
latency while this large request runs. Confirm the `VMAlert` is operational
and `vm_poc:up:sum` is queryable through vmselect.

Federation returns the latest value for each selected series at each proxy
scrape. Samples from platform's intervening scrapes do not arrive in VM.
Therefore this path cannot establish raw sample parity with direct platform
scraping. A render or dry-run cannot establish live target parity, proxy
health, or recording-rule operation.

## Capacity gate before staging sync

Measure active series, ingestion rate, new-series churn, and source cadence
separately for platform, UWM, and Tekton. For platform, use the configured
federation response's **samples after relabeling per successful scrape divided
by its observed interval**. For each direct target group, estimate one
replica's samples/second at its effective interval,
`max(30s, source interval)`, and reconcile that estimate with the agent's
observed samples and target inventory. Include the two native Tekton jobs;
their platform copies are excluded by the federation selector.

`raw VM samples/second = 2 * (platform federation samples/second
+ direct UWM samples/second + direct Tekton samples/second)`.
The factor of two is the two VMAgent replicas per shard. Three shards split
targets, but the single large federation target lands on one shard pair.
Size that pair's CPU, memory, 10Gi queues, scrape response, and recovery
margin separately. The replicas have identical external labels, so count
distinct new-series and index growth once only after verifying their
series identity in live data.

A read-only full-platform request on 2026-10-05 returned about 2.04 million
metric lines, 845 MB uncompressed, in 50 seconds. That measurement predates
the Tekton job exclusion. At two minutes and two agent replicas, roughly
2.04 million platform series would produce 34,000 raw VM samples/second.
Two 30-second direct scrapers of those same series would produce about
136,000 samples/second. The proxy therefore tests lower write throughput
than future direct platform scraping. Its very large response can still
burden Prometheus: [Red Hat recommends a limited, aggregated federation
selection](https://docs.redhat.com/en/documentation/monitoring_stack_for_red_hat_openshift/4.21/html/accessing_metrics/accessing-monitoring-apis-by-using-the-cli#querying-metrics-by-using-the-federation-endpoint-for-prometheus).
Recheck one response's size and duration before staging sync and stop the
test if platform monitoring degrades.

Project four days (72-hour retention plus one retention cycle) using
`345600 * (raw VM samples/second * measured VM bytes/raw sample
+ distinct new-series/second * measured index bytes/new series)`.
Measure the byte factors with both replicas writing, including transient disk
used before deduplication. Do not infer raw ingestion from query-visible
samples. The 30-second storage and select deduplication can reduce later
disk use and query-visible samples, but both copies are ingested. At long
cadences, replica scrapes in different 30-second buckets may both remain
visible. Compare per-replica scrape counts with a stable slow `up` series
using `count_over_time(up{job="<job>",instance="<instance>"}[1h])`.

Size the busiest of 20 storage shards, not just the average. Require at
least 20% free space on every storage PVC after the four-day projection;
the initial 20Gi PVC is a test default, not a sizing assertion. Check actual
placement and the largest number of shards on any surviving worker after
one worker is lost, at least `ceil(20 / (eligible workers - 1))`. Include
their projected load and other workloads; require at least 50% spare
allocatable CPU and RAM in that case. Expand PVCs before applying if the
measured projection requires it. A proxy-based sandbox or staging test
cannot prove 20-shard capacity for future direct platform scraping.

## When a source monitor is deleted

`operator.enable_converter_ownership: true` adds an owner reference from
each converted object to its source. Kubernetes garbage collection should
delete the converted object when the source is deleted. Check the
corresponding VM object in the same namespace and confirm it disappears.
Investigate any copy that remains instead of deleting unrelated VM objects.

## Before the first PoC sync

Confirm that `openshift-monitoring` has the
`cluster-monitoring-metrics-api` Role, port 9091 of the
`prometheus-k8s` Service, and the PoC namespace's
`openshift-service-ca.crt` ConfigMap after namespace creation. Save an
inventory of VM scrape objects so teardown can distinguish this PoC's
conversions from objects managed by other operators. Also save the rule
and AlertmanagerConfig inventory to confirm conversion stays disabled.
First check `oc api-resources --api-group=operator.victoriametrics.com -o name`.
If no VM kinds are installed yet, record an empty baseline; the following
`oc get` commands would fail until this PoC installs the CRDs. If the kinds
already exist, save their current objects:

```sh
oc get vmservicescrapes,vmpodscrapes,vmprobes,vmscrapeconfigs -A -o json > vm-scrapes-before-poc.json
oc get vmrules,vmalertmanagerconfigs -A -o json > vm-rules-before-poc.json
```

Keep these snapshots outside the Git repository. Record any other VM
operators running in the cluster.

## After the first PoC sync

The staging Argo CD Application has automated pruning enabled. Argo CD
prunes resources it tracks; it does not make every resource absent from
Git disappear. The converter adds
`argocd.argoproj.io/sync-options: Prune=false` to each converted object.
That is the sync-prune guard. Its `IgnoreExtraneous` compare annotation
only keeps generated objects from making the Application appear out of
sync; it does not prevent deletion.

Inspect a live converted object and confirm both annotations and its
source owner reference are present. Compare all `VMRule` and
`VMAlertmanagerConfig` objects outside `appstudio-vm-poc` with the
pre-sync inventory; this PoC must not create any there. Check the Argo
CD resource tree and sync result for unexpected prune candidates. If
the annotations are absent, resolve that before relying on normal syncs.
`Prune=false` does not prevent Kubernetes owner garbage collection,
direct deletion, namespace deletion, or CRD deletion.

## Retiring the PoC

1. In a separate GitOps change, set
   `operator.disable_prometheus_converter` to `true` in
   `operator-values.yaml` and sync. Keep the rest of the package deployed.
   Wait for the operator Deployment in `appstudio-vm-poc` to roll out and
   verify its pods have the converter disabled.
2. Inventory the same four VM scrape kinds across **all namespaces**,
   saving the result as `vm-scrapes-at-teardown.json` without overwriting
   the baseline. For each proposed deletion, inspect its namespace, name,
   UID, `metadata.ownerReferences`, and Argo annotations. The converted
   copies have a `monitoring.coreos.com` source owner and the converter's
   `IgnoreExtraneous` / `Prune=false` annotations. Compare with the
   pre-PoC inventory and confirm that another operator does not manage it.
   If ownership is ambiguous, resolve that before deleting anything.
3. Delete only the reviewed PoC-generated converted objects, using
   explicit kind, namespace, and name pairs. Do not use a cluster-wide
   `--all` deletion. Re-list all four kinds and check that the copies
   remain gone. Disabling conversion first prevents this operator from
   recreating them. Deleting a converted copy alone does not reliably
   trigger a running converter to rebuild it.
4. Remove `poc/victoriametrics` from the parent
   `stone-stg-rh01/kustomization.yaml` and remove
   `vm-poc-datasource.yaml` from the staging Grafana Kustomization. Sync
   both with pruning. Verify the PoC's native `VMStaticScrape`, Tekton
   `VMServiceScrape` objects, `VMAgent`, `VMCluster`, `VMAlert`, `VMRule`,
   operator Deployment and RBAC (including the RoleBinding in
   `openshift-monitoring`), network policies, Grafana datasource, namespace,
   and generated pods are gone. Check remaining PoC PVCs and PVs before
   considering teardown complete; retained volumes need an explicit data
   deletion decision.
5. The operator CRDs use `Prune=false` and can be shared with other VM
   users. Inventory **all** VM custom resources and operators cluster-wide
   before removing CRDs in a separate decommission. Never delete the CRDs
   merely to clear this PoC's converted objects.

Normal Argo CD sync will not perform step 3 for this package because the
converted objects carry `Prune=false`. The source owner reference handles
source deletion, while explicit inventory and deletion handle PoC
retirement.

A later, freshly deployed VM operator will convert existing source monitors
again in each cluster if conversion is enabled and it watches their
namespaces. Check target parity after that rollout. This startup conversion
is different from relying on an already running operator to notice
deletion of a converted object. The PoC's native `VMRule` and `VMAlert`
are not recreated by monitor conversion.
