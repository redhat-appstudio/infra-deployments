# stone-stg-rh01 VictoriaMetrics PoC

This package runs in `appstudio-vm-poc`. VMAgent scrapes user workload
monitoring (UWM) and Tekton targets directly. Alloy's existing platform
`/federate` scrape also forwards platform samples to both shard-0 VMAgent
replicas, which write to this PoC's `vminsert`. Alloy continues to send its
existing data to Mimir; RHOBS MonitoringStacks and their deliveries are
unchanged.

The operator converts existing `ServiceMonitor`, `PodMonitor`, `Probe`, and
`ScrapeConfig` objects into VM objects in their source namespaces. The agent's
selectors use the UWM namespace and monitor labels, so converted platform
monitors can exist without being scraped directly. Conversion of
`PrometheusRule` and `AlertmanagerConfig` is disabled; this package provides
its own `VMRule` and `VMAlert`.

## Collection paths

- **Platform:** Alloy's existing clustered platform scrape reads
  `https://prometheus-k8s.openshift-monitoring.svc:9091/federate` every
  60 seconds with a 55-second timeout, using its service account and the
  OpenShift service CA. One scrape feeds both the existing Mimir branch and
  a separate VM branch without another platform request. The VM branch drops
  the source `prometheus` and `prometheus_replica` labels. It adds missing PoC
  `cluster`, `collector`, and `source_environment` labels, then excludes the
  two Tekton jobs below.
  Alloy sends that same VM branch to two distinct shard-0 VMAgent replicas
  through `/api/v1/write`; each agent forwards it to `vminsert`. VMAgent's
  `externalLabels` apply to its own scrapes, not to pushed samples. No
  VMAgent platform scrape or platform Prometheus API RoleBinding is needed.
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
  `pipeline-metrics-exporter`, are dropped only from Alloy's VM branch to
  avoid collecting the same endpoints twice in VM. Mimir's platform path
  remains unchanged. Recheck those job labels if Tekton's monitors change.
  The Tekton Prometheus proxy is not a VM scrape target.

Network policies permit the Alloy-to-VMAgent writes and the direct UWM and
Tekton connections where source namespaces restrict monitoring ingress. The
PoC has no RHOBS remote-write destination.

## Validation after staging sync

Compare the direct UWM and Tekton targets on every VMAgent shard with the
source Prometheus target URLs, labels, and health. Investigate missing targets
and new failures; the source UWM snapshot on 2026-10-05 already had four down
targets. Verify both native Tekton scrapes and the selected UWM monitors,
including Secret-backed authentication and kaexporter's effective 300-second
cadence. Check that no converted platform monitor is selected for direct
scraping.

The platform proxy appears as **one target in the Alloy cluster**, not as a
VMAgent target or hundreds of original platform targets. Confirm Alloy's
60-second cadence, scrape duration below 55 seconds, successful Mimir writes,
and equal sample delivery to its two distinct VM endpoints. Watch each Alloy
endpoint's failures, pending samples, WAL use, and queue lag; its VM WAL can
discard samples once they exceed the configured 15-minute keepalive. Confirm
that both shard-0 VMAgents receive platform samples and drain their persistent
queues without drops. There should be no platform `VMStaticScrape` or platform
target on VMAgent. Query representative non-Tekton platform series through
vmselect and compare their labels and
current values with platform Prometheus; check that Mimir-only labels and
duplicate Tekton platform jobs are absent. Watch VMAgent CPU, memory and
queue PVCs, as well as platform Prometheus CPU, memory, and query latency.
Confirm the `VMAlert` is operational and `vm_poc:up:sum` is queryable through
vmselect.

Federation returns the latest value for each selected series at each Alloy
scrape. Alloy uses its scrape time rather than the source sample timestamp;
samples from platform's intervening scrapes do not arrive in VM. Therefore
this path cannot establish raw sample parity with direct platform scraping.
A render or dry-run cannot establish live target parity, Alloy delivery,
queue health, or recording-rule operation.

## Capacity gate before staging sync

Measure active series, ingestion rate, new-series churn, and source cadence
separately for platform, UWM, and Tekton. For platform, use Alloy's **VM-branch
samples after the Tekton drop per successful scrape divided by its observed
60-second interval**. Do not normalize this path to a 30-second scrape. For
each direct target group, estimate one replica's samples/second at its
effective interval, `max(30s, source interval)`, and reconcile that estimate
with the agent's observed samples and target inventory. Include the two native
Tekton jobs;
their platform copies are excluded by the Alloy VM branch.

`raw VM samples/second = 2 * (Alloy platform-to-VM samples/second
+ direct UWM samples/second + direct Tekton samples/second)`.
The factor of two counts both VMAgent copies, including the two copies of
each Alloy platform sample. Three shards split direct scrape targets, while
the entire platform stream enters the shard-0 pair. Size that pair's CPU,
memory, 10Gi queues, network throughput, and recovery margin separately.
Also size the Alloy pod that owns the platform scrape, its shared 5Gi PVC, and
both VM endpoint queues while keeping the existing Mimir queue healthy.
Measure queue drain and recovery before either the Alloy WAL or VMAgent's
10Gi persistent queue fills. Count distinct new-series and index growth once
only after verifying that both VM writes have identical series labels.

A read-only full-platform request on 2026-10-05 returned about 2.04 million
metric lines, 845 MB uncompressed, in 50 seconds. That measurement predates
the VM-branch Tekton job exclusion. Alloy still downloads the full response
for Mimir. At 60 seconds and two VM endpoints, roughly 2.04 million platform
series would produce 68,000 raw VM samples/second before that exclusion.
Two 30-second direct scrapers of those same series would produce about
136,000 samples/second. The proxy therefore tests lower write throughput
than future direct platform scraping. Its very large response can still
burden Prometheus: [Red Hat recommends a limited, aggregated federation
selection](https://docs.redhat.com/en/documentation/monitoring_stack_for_red_hat_openshift/4.21/html/accessing_metrics/accessing-monitoring-apis-by-using-the-cli#querying-metrics-by-using-the-federation-endpoint-for-prometheus).
The 50-second measurement leaves little margin under Alloy's 55-second
timeout. Recheck response size and duration before staging sync and stop the
test if platform monitoring degrades.

Project four days (72-hour retention plus one retention cycle) using
`345600 * (raw VM samples/second * measured VM bytes/raw sample
+ distinct new-series/second * measured index bytes/new series)`.
Measure the byte factors with both replicas writing, including transient disk
used before deduplication. Do not infer raw ingestion from query-visible
samples. The 30-second storage and select deduplication can reduce later
disk use and query-visible samples, but both copies are ingested. For
platform, compare Alloy's per-endpoint sent counts with each VMAgent's
received and forwarded counts; the two writes have the same scrape timestamp.
For slow direct scrapes, replica scrapes in different 30-second buckets may
both remain visible. Compare their per-replica scrape counts with a stable
`up` series using `count_over_time(up{job="<job>",instance="<instance>"}[1h])`.

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

Confirm Alloy's current platform scrape and Mimir delivery are healthy.
Before enabling the VM fan-out, check that both shard-0 VMAgent pods are
Ready and their distinct pod DNS names resolve from Alloy, with network
policy access on port 8429. If both changes sync together, watch Alloy's
new VM queues while the receivers start. Confirm the PoC namespace's
`openshift-service-ca.crt` ConfigMap after namespace creation; selected UWM
monitors still use it. Save an inventory of VM scrape objects so teardown can
distinguish this PoC's conversions from objects managed by other operators.
Also save the rule and AlertmanagerConfig inventory to confirm conversion
stays disabled.
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

1. Let both Alloy VM endpoint queues drain. In a separate GitOps change,
   remove only Alloy's VM forwarding branch, its two endpoints, and its
   VM-specific egress rule; keep the platform scrape and Mimir path running.
   Sync Alloy and confirm Mimir delivery remains healthy. Stop these writes
   before removing the VMAgents so Alloy cannot build a backlog.
2. In a separate GitOps change, set
   `operator.disable_prometheus_converter` to `true` in
   `operator-values.yaml` and sync. Keep the rest of the package deployed.
   Wait for the operator Deployment in `appstudio-vm-poc` to roll out and
   verify its pods have the converter disabled.
3. Inventory the same four VM scrape kinds across **all namespaces**,
   saving the result as `vm-scrapes-at-teardown.json` without overwriting
   the baseline. For each proposed deletion, inspect its namespace, name,
   UID, `metadata.ownerReferences`, and Argo annotations. The converted
   copies have a `monitoring.coreos.com` source owner and the converter's
   `IgnoreExtraneous` / `Prune=false` annotations. Compare with the
   pre-PoC inventory and confirm that another operator does not manage it.
   If ownership is ambiguous, resolve that before deleting anything.
4. Delete only the reviewed PoC-generated converted objects, using
   explicit kind, namespace, and name pairs. Do not use a cluster-wide
   `--all` deletion. Re-list all four kinds and check that the copies
   remain gone. Disabling conversion first prevents this operator from
   recreating them. Deleting a converted copy alone does not reliably
   trigger a running converter to rebuild it.
5. Remove `poc/victoriametrics` from the parent
   `stone-stg-rh01/kustomization.yaml` and remove
   `vm-poc-datasource.yaml` from the staging Grafana Kustomization. Sync
   both with pruning. Verify the PoC's native Tekton `VMServiceScrape`
   objects, VMAgent receiver Service, `VMAgent`, `VMCluster`, `VMAlert`,
   `VMRule`, operator Deployment and RBAC, network policies, Grafana
   datasource, namespace, and generated pods are gone. Check remaining PoC
   PVCs and PVs before considering teardown complete; retained volumes need
   an explicit data deletion decision.
6. The operator CRDs use `Prune=false` and can be shared with other VM
   users. Inventory **all** VM custom resources and operators cluster-wide
   before removing CRDs in a separate decommission. Never delete the CRDs
   merely to clear this PoC's converted objects.

Normal Argo CD sync will not perform step 4 for this package because the
converted objects carry `Prune=false`. The source owner reference handles
source deletion, while explicit inventory and deletion handle PoC
retirement.

A later, freshly deployed VM operator will convert existing source monitors
again in each cluster if conversion is enabled and it watches their
namespaces. Check target parity after that rollout. This startup conversion
is different from relying on an already running operator to notice
deletion of a converted object. The PoC's native `VMRule` and `VMAlert`
are not recreated by monitor conversion.
