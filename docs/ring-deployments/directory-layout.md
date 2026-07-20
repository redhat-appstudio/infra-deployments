# Canonical Directory Layout — Ring-Based Multi-Cluster Deployments

> **Warning**
> Ring deployments are still under active development. This documentation may change as the implementation evolves.

The standard directory structure every component in infra-deployments must follow.

## Table of Contents

1. [Why This Exists](#1-why-this-exists)
2. [How It Works: Kargo + ArgoCD](#2-how-it-works-kargo--argocd)
3. [The Canonical Layout](#3-the-canonical-layout)
4. [Tier Definitions](#4-tier-definitions)
   - 4.1 [Tier 1 — Component Base](#41-tier-1--component-base)
   - 4.2 [Tier 2 — Ring Base](#42-tier-2--ring-base)
   - 4.3 [Tier 3 — Cluster Overlay](#43-tier-3--cluster-overlay)
   - 4.4 [Tier 4 — Features](#44-tier-4--features)
5. [File Conventions](#5-file-conventions)
   - 5.1 [File Naming](#51-file-naming)
   - 5.2 [One Resource Per File](#52-one-resource-per-file)
   - 5.3 [Patches Directory](#53-patches-directory)
6. [Validation](#6-validation)

---

## 1. Why This Exists

### The Problem

Konflux deploys the same components across multiple clusters — development, staging, production, and critical production. Without a controlled rollout mechanism, changes to infra-deployments land on all clusters simultaneously when they merge. A bad configuration change can break production within minutes of merging. There is no safe, gradual rollout path.

### The Solution: Ring-Based Progressive Delivery

We adopt a **ring-based progressive delivery model**. Changes roll out through a sequence of rings, starting from lower-risk environments and advancing toward production:

| Ring | Purpose | What Happens Here |
|------|---------|-------------------|
| **Ring 0** | Development | First real deployment. Conformance tests run in the PR before merge. |
| **Ring 1** | Staging | Staging clusters. Verified with live conformance tests, ArgoCD sync health, and metrics. |
| **Ring N** | ... | Each additional ring follows the same pattern — promoted only after Ring N-1 passes all verification gates. Higher rings can add stricter or additional verification steps as needed. |

A change must pass verification in Ring N before it is promoted to Ring N+1. Between rings, a **soak time** is enforced — a mandatory waiting period after verification passes, allowing the change to run in the current ring long enough to surface issues that tests alone cannot catch. The number of rings is not fixed. If verification fails at any ring, promotion stops — higher rings are never exposed to a change that broke a lower one.

---

## 2. How It Works: Kargo + ArgoCD

**ArgoCD** syncs the Git repository to each cluster. Each cluster has an ArgoCD Application pointing at its specific directory path in infra-deployments. At scale, **argocd-agent** (available in OpenShift GitOps) enables managing hundreds of spoke clusters from a central hub.

**[Kargo](https://kargo.io)** is the promotion engine that layers on top of ArgoCD. Without Kargo, ArgoCD alone would deploy changes to every cluster the moment they merge — there would be no controlled, ring-by-ring rollout. Kargo watches for new component versions — container image tags, upstream Git commits, Helm charts, or commits in infra-deployments itself (e.g. RBACs and other resources authored directly in the repo that need ring-by-ring rollout) — and promotes them ring-by-ring by updating the Git repo. Kargo serializes promotions within each Stage — it queues them and processes one at a time, with verification blocking subsequent promotions. This ensures each change is individually verified before the next one lands.

**Why this matters for directory structure:** Kargo writes to a single, computable path per ring. Given a component name and ring number, the path is always `components/{component}/rings/ring-N/base/kustomization.yaml`. Each component has its own [PromotionTask](architecture.md#8-definitions) per ring, but the canonical layout means all PromotionTasks follow the same pattern — the path is computable from `{component}` and `ring-N`, keeping PromotionTask definitions simple and consistent.

**How promotion works:** Kargo does not commit directly to `main`. Instead, it pushes to a branch, opens a PR, and either auto-merges (staging rings) or waits for manual merge approval (production rings). This means every promotion is a reviewable PR — production changes require human approval before they reach the cluster.

---

## 3. The Canonical Layout

One structure. Every component. No exceptions. Given a component name, a ring number, and a cluster name, the path to any `kustomization.yaml` is computable without looking it up.

```
components/{component-name}/
│
├── base/                              # TIER 1
│   ├── kustomization.yaml
│   ├── allow-argocd-to-manage-cr.yaml
│   └── monitoring/
│       ├── controller-sm.yaml
│       └── slo-alerts-pr.yaml
│
├── new-base/                          # TIER 1 (ring-promoted copy of base/ — see §4.1)
│
├── features/                           # TIER 4 (library — not a layer; referenced via components:)
│   ├── rh-certs/
│   │   └── kustomization.yaml
│   ├── debug/
│   │   └── kustomization.yaml
│   └── high-availability/
│       └── kustomization.yaml
│
├── rings/
│   ├── ring-0/                        # Development
│   │   └── base/                      # TIER 2
│   │       ├── kustomization.yaml     # ← Kargo writes here (image tags, Git SHAs, Helm charts, ...)
│   │       ├── ...                    # ← Helm values, Chart.yaml, or other config Kargo manages
│   │       ├── rbac/                  # ← Human-authored in first ring, promoted by Kargo to next rings
│   │       │   └── build-pipeline-runner-rb.yaml
│   │       └── patches/               # ← Human-authored
│   │           └── resource-limits-patch.yaml
│   │
│   ├── ring-1/                        # Staging
│   │   ├── base/                      # TIER 2
│   │   │   ├── kustomization.yaml     # ← Kargo writes here (image tags, Git SHAs, Helm charts, ...)
│   │   │   ├── ...                    # ← Helm values, Chart.yaml, or other config Kargo manages
│   │   │   ├── rbac/                  # ← Human-authored in first ring, promoted by Kargo to next rings
│   │   │   │   └── build-pipeline-runner-rb.yaml
│   │   │   └── patches/               # ← Human-authored
│   │   │       └── resource-limits-patch.yaml
│   │   ├── stone-stg-rh01/            # TIER 3
│   │   │   ├── kustomization.yaml
│   │   │   ├── webhook-config.json
│   │   │   └── patches/
│   │   │       └── external-secret-path-patch.yaml
│   │   └── stone-stage-p01/            # TIER 3
│   │       ├── kustomization.yaml
│   │       └── patches/
│   │           └── external-secret-path-patch.yaml
│   │
│   ├── ring-N/                        # Ring N (same structure)
│   │   ├── base/                      # TIER 2
│   │   │   ├── kustomization.yaml     # ← Kargo writes here (image tags, Git SHAs, Helm charts, ...)
│   │   │   ├── ...                    # ← Helm values, Chart.yaml, or other config Kargo manages
│   │   │   ├── rbac/                  # ← Human-authored in first ring, promoted by Kargo to next rings
│   │   │   └── patches/               # ← Human-authored
│   │   ├── {cluster-a}/               # TIER 3
│   │   │   └── kustomization.yaml
│   │   └── {cluster-b}/               # TIER 3
│   │       └── kustomization.yaml
│
├── OWNERS
└── README.md
```

> **Minimum Viable Layout — Single-Ring Component**
>
> Components that deploy to only one ring (e.g. a dev-only tool) still follow the canonical structure — just with one ring and one cluster directory:
>
> ```
> components/{dev-tool}/
> ├── base/                              # Tier 1
> ├── rings/
> │   └── ring-0/
> │       ├── base/                      # Tier 2
> │       │   └── kustomization.yaml
> │       └── {cluster}/                 # Tier 3
> │           └── kustomization.yaml
> └── OWNERS
> ```
>
> Don't create empty ring directories for rings the component doesn't deploy to.

> **Where Kargo Writes**
>
> Kargo **primarily** writes to Tier 2: `components/{component}/rings/ring-N/base/`. The path is computable from `{component}` and `ring-N`, enabling a single generic [PromotionTask](architecture.md#8-definitions). When Tier 1 (`base/`) changes need ring-by-ring rollout, Kargo promotes them via `new-base/` — a copy of `base/` that travels ring-by-ring and is renamed to `base/` at the destination.
>
> **Where does my change go?** Two questions:
>
> 1. **Is it the same everywhere and non-disruptive?** → **Tier 1**. Merges to all rings at once. Examples: ArgoCD sync permissions, monitoring ServiceMonitors.
> 2. **Does it differ?** Then ask: *at what level?*
>    - **Differs between rings, but same across all clusters within a ring** → **Tier 2**. This covers two cases with different lifecycles: (a) **promoted content** like image tags and upstream refs — authored in the first ring, Kargo copies them forward; and (b) **ring-authored config** like ExternalSecret vault paths or feature flags — authored independently in each ring because the values differ (e.g., staging vault path ≠ production vault path). Both live in Tier 2 because they apply uniformly to all clusters in the ring.
>    - **Differs between individual clusters within the same ring** → **Tier 3**. Examples: a webhook endpoint unique to one cluster, resource sizing for a differently-shaped cluster, a TLS certificate specific to one cluster's domain.

---

## 4. Tier Definitions

Every component follows the same four tiers. Tiers can be minimal but must exist. No skipping tiers, no alternative layouts. Consistency over cleverness.

### Tier Overview

| Tier | Path | Owns | Changed By |
|------|------|------|------------|
| **Tier 1** base/ | `components/{c}/base/` | Infra-authored: ArgoCD permissions, platform-level RBACs, monitoring | Component team — manual PR |
| **Tier 2** ring base | `components/{c}/rings/ring-N/base/` | Promoted: component versions, namespace, upstream refs. Ring-authored: ExternalSecret vault paths, feature flags, resource baselines | Kargo promotes versions; ring-authored config is written independently per ring |
| **Tier 3** cluster | `components/{c}/rings/ring-N/{cluster}/` | Only what differs between clusters in the same ring: unique webhook endpoints, per-cluster resource sizing, cluster-specific TLS | Manual PR |
| **Tier 4** features | `components/{c}/features/{name}/` | Opt-in capabilities (certs, debug, HA). Referenced via `components:` in any tier. | Component team — manual PR |

### 4.1 Tier 1 — Component Base

**The Foundation.**

Changes here bypass ring promotion entirely — they hit every ring simultaneously on merge. Only non-disruptive, infra-level resources belong here: ArgoCD sync permissions, monitoring. Nothing that impacts users or running workloads should be in Tier 1 — if a change could break a component or affect tenants, it must go through the rings via Tier 2.

> **Promoting base changes through rings (`new-base/`)**
>
> When a Tier 1 change needs ring-by-ring rollout (e.g., it could impact workloads), create a `new-base/` directory alongside `base/`. Kargo promotes `new-base/` ring-by-ring like any Tier 2 content, and at the destination ring it is renamed to `base/`. This gives you the safety of ring promotion for changes that would otherwise hit every ring at once.

```yaml
# components/build-service/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- allow-argocd-to-manage-cr.yaml           # ArgoCD sync permission
- monitoring.yaml                          # ServiceMonitors, alerts
- rbac/                                    # Platform-level RBACs
- build-pipeline-config/build-pipeline-config.yaml

commonAnnotations:
  argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true
```

Rules:

- **Must exist** for every component, even if it contains a single resource.
- **Infra-level resources only.** ArgoCD permissions, monitoring. Any resource that could impact workloads or tenants — including RBACs that grant new permissions — belongs in Tier 2, not here.
- **Never contains** `images:`, `namespace:`, environment URLs, or cluster-specific secrets.
- **Changed by** the component team via manual PR.

### 4.2 Tier 2 — Ring Base

**The Ring-Wide Shared Layer.**

This is the ring's shared foundation — every resource that must be consistent across all clusters in a ring lives here. Tier 2 holds two types of content with different lifecycles:

1. **Promoted content** — component versions (image tags, upstream refs), RBACs, and other resources that Kargo copies from ring to ring. Authored in the first ring, promoted automatically to subsequent rings.
2. **Ring-authored config** — configuration that is the same for every cluster in the ring but differs between rings, such as ExternalSecret vault paths, environment-specific feature flags, or resource baselines. These are authored independently in each ring's Tier 2 because the values are ring-specific — Kargo does not promote them.

Tier 2 pulls in Tier 1, pins the component version (however the component manages that), and sets the namespace. Only ring-common objects belong here; anything that differs between individual clusters within a ring belongs in Tier 3. **This is the primary tier Kargo writes to.**

**Directory convention** — separate promoted from ring-authored content visually:

```
rings/ring-1/base/
├── kustomization.yaml              # images, upstream refs — Kargo writes here (promoted)
├── rbac/                           # RBACs authored in first ring, promoted by Kargo
│   ├── role.yaml
│   └── rolebinding.yaml
├── external-secrets/               # ring-authored — Kargo ignores, authored per ring
│   └── pipelines-as-code.yaml
└── patches/                        # ring-wide patches
    └── resource-limits-patch.yaml
```

Promoted resources (RBACs, CRDs, additional manifests) and ring-authored resources (ExternalSecrets, feature flags) both live in Tier 2 but in their own subdirectories for clarity.

> **Invariant Subdirectory Pattern**
>
> Components that deploy an operator with a Custom Resource can isolate the Kargo-promoted content into an `invariant/` subdirectory within Tier 2. This separates what Kargo writes (upstream ref, image pin) from ring-authored configuration (CR components, environment-specific patches):
>
> ```
> rings/ring-0/base/
> ├── kustomization.yaml              # resources: [invariant], components: [cr/*], patches: [release-config.yaml]
> ├── invariant/                      # Kargo-promoted content only
> │   ├── kustomization.yaml          # upstream ref (?ref=SHA), images block, konflux.yaml
> │   └── konflux.yaml                # minimal CR shell
> ├── cr/                             # ring-owned CR overlay components (team-owned, with OWNERS)
> │   ├── build/
> │   ├── integration/
> │   └── ...
> └── release-config.yaml             # environment-specific CR settings (non-team-owned)
> ```
>
> Kargo promotes by updating `invariant/kustomization.yaml` (the `?ref=` and `images:` block). The CR components and `release-config.yaml` are ring-authored and not promoted — each ring defines its own values. See `components/konflux-operator/` for a concrete example.

```yaml
# components/build-service/rings/ring-1/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../../base                                                    # ← Tier 1
- rbac/                                                            # ← promoted: Kargo copies ring-to-ring
- external-secrets/                                                # ← ring-authored: Kargo ignores
- https://github.com/konflux-ci/build-service/config/default?ref=04a4744  # ← promoted: upstream ref

namespace: build-service

images:                                                            # ← promoted: Kargo writes HERE
- name: quay.io/konflux-ci/build-service
  newName: quay.io/konflux-ci/build-service
  newTag: 04a4744321a7fb747f796da783d51fc322aef598

components:
- ../../../features/rh-certs                                       # ← Tier 4 opt-in

patches:
- path: patches/resource-limits-patch.yaml
```

Rules:

- **All clusters in a ring run the same versions.** Image tags, Helm chart versions, Git refs — whatever the component pins, every cluster in the ring gets the same set. No per-cluster version overrides.
- **Upstream resource refs** (`?ref=SHA`) live here. The upstream ref pulls in all component resources — Deployments, RBACs, Services, CRDs — so they are promoted ring-by-ring with the component version.
- **RBACs and additional manifests can be promoted.** If a component needs extra Roles, RoleBindings, or ServiceAccounts authored in infra-deployments (not upstream), place them in a subdirectory (e.g. `rbac/`). Kargo promotes these ring-to-ring via Git Warehouse subscriptions — author once in the first ring, and they flow forward automatically.
- **Ring-authored config lives here too.** ExternalSecret vault paths, feature flags, resource baselines — if it's the same for every cluster in the ring but differs between rings, it's Tier 2. Place ring-authored resources in their own subdirectory (e.g. `external-secrets/`). These are authored independently in each ring (not promoted by Kargo). Don't duplicate them across Tier 3 directories.
- **Only ring-common patches.** Patches in Tier 2 must apply to all clusters in the ring. Patches that differ between individual clusters within the same ring belong in Tier 3.
- **Kargo promotes** image tags, upstream `?ref=SHA`, Helm chart versions, Git commits in infra-deployments itself (via [Git Warehouse subscriptions](https://docs.kargo.io/user-guide/how-to-guides/working-with-warehouses#git-repository-subscriptions)), and any manifests authored directly in the first ring's Tier 2 (like RBACs).
- **Path formula:** `components/{component}/rings/ring-N/base/kustomization.yaml` — always.

> **Helm-Based Components**
>
> For components deployed via Helm charts, the `HelmChartInflationGenerator` and its `valuesInline` live in Tier 2 alongside `kustomization.yaml`. Kargo promotes Helm components by updating the generator file — writing the chart `version` and image tags inside `valuesInline`:
>
> ```
> rings/ring-1/base/
> ├── kustomization.yaml                  # generators: [kargo-helm-generator.yaml]
> └── kargo-helm-generator.yaml           # ← Kargo writes version + valuesInline.image.tag here
> ```
>
> ```yaml
> # kargo-helm-generator.yaml — Kargo updates version and image tags
> apiVersion: builtin
> kind: HelmChartInflationGenerator
> metadata:
>   name: kargo
> name: kargo
> repo: oci://ghcr.io/akuity/kargo-charts
> version: 1.10.7                         # ← Kargo writes chart version
> namespace: kargo
> releaseName: kargo
> valuesInline:
>   image:
>     repository: quay.io/konflux-ci/kargo
>     tag: 1.10-2cd30bc                   # ← Kargo writes image tag
> ```
>
> The PromotionTask uses `yaml-update` steps to write these fields. Ring-authored Helm values (environment-specific endpoints, OIDC config) are part of the same file — they stay in Tier 2 and are authored independently per ring, just like ExternalSecrets.

### 4.3 Tier 3 — Cluster Overlay

**The Leaf Node — Where ArgoCD Points.**

Each ring can contain multiple clusters. ArgoCD ApplicationSets point at Tier 3 — one directory per cluster within a ring. This layer pulls in the ring base (Tier 2) and adds only what truly differs between individual clusters in the same ring. If a value is the same for every cluster in the ring, it belongs in Tier 2 — not duplicated across Tier 3 directories. Tier 3 is narrower than it might seem: most environment-level config (secret vault paths, feature flags) is ring-level and belongs in Tier 2. Tier 3 is for cluster-specific overrides like a webhook endpoint unique to one cluster, resource sizing for a differently-shaped node, or a TLS certificate tied to a specific domain. ExternalSecret resources are defined in Tier 2; Tier 3 patches only cluster-specific fields such as a vault path segment that includes the cluster name.

```yaml
# components/build-service/rings/ring-1/stone-stg-rh01/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../base                                                          # ← Tier 2

patches:
- path: patches/external-secret-path-patch.yaml                    # cluster-specific secret vault
  target:
    name: pipelines-as-code-secret
    kind: ExternalSecret

configMapGenerator:
- name: webhook-config                                             # cluster-specific webhook endpoint
  files:
  - webhook-config.json
```

Rules:

- **Must exist** for every deployed cluster. ArgoCD ApplicationSets target Tier 3 — without a cluster directory, ArgoCD has nothing to sync.
- **Must reference `../base`.** A cluster kustomization that doesn't include its ring base is a broken deployment.
- **Never sets image tags.** The application version is owned by Tier 2.
- **Changed by** manual PR.
- **Adding a cluster** requires two steps: (1) create a subdirectory with a `kustomization.yaml` referencing `../base`, and (2) ensure the ArgoCD ApplicationSet generator includes the new cluster (see `argo-cd-apps/overlays/` for ring overlay definitions). Without step 2, the directory exists but ArgoCD won't create an Application for it.
- **Removing a cluster** — reverse the process: remove the ArgoCD Application (or update the ApplicationSet generator), verify no active workloads remain, then delete the directory.

### 4.4 Tier 4 — Features

**Opt-In Capabilities.**

Kustomize Components for optional cross-cutting concerns. Any tier (1, 2, or 3) can opt in by adding a `components:` reference. Use for capabilities that not every ring or cluster needs — RH certificates, debug mode, high-availability patches, profiling. Features are never mandatory.

**Why not in `base/`?** Resources in Tier 1 apply to every ring and cluster unconditionally. Features exist separately because not all environments need them — `debug/` belongs only in Ring 0, `high-availability/` only in production rings, `rh-certs/` only on Red Hat-hosted clusters. Keeping them in `features/` lets each ring or cluster opt in selectively without forcing the capability on environments where it is unnecessary or harmful.

```yaml
# components/build-service/features/rh-certs/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component                                                    # ← Kustomize Component, not Kustomization

patches:
- path: add-rh-certs-patch.yaml
  target:
    name: build-service-controller-manager
    kind: Deployment

configMapGenerator:
- name: trusted-ca
  options:
    labels:
      config.openshift.io/inject-trusted-cabundle: "true"

namespace: build-service
```

Rules:

- **Kind is `Component`**, not `Kustomization`. Referenced via `components:`, not `resources:`.
- **Any tier can opt in:** Ring 1 can enable `rh-certs` while Ring 0 doesn't. A specific cluster can enable `debug` while others in the same ring don't.
- **Changed by** the component team via manual PR.

> **Reference Direction — Always Upward**
>
> References always go upward: Tier 3 → Tier 2 (`../base`), Tier 2 → Tier 1 (`../../../base`). ArgoCD ApplicationSets point at Tier 3 — the leaf. Each leaf assembles the full stack by pulling its parents.

---

## 5. File Conventions

### 5.1 File Naming

Every resource file is named `{resource-name}-{type-abbreviation}.yaml`. The resource name comes first, followed by a short abbreviation of the Kubernetes resource type. This makes `ls` output self-documenting — you know what each file contains without opening it.

| Resource Kind | Abbreviation | Example Filename |
|--------------|-------------|-----------------|
| ServiceAccount | `sa` | `konflux-ui-sa.yaml` |
| Role | `role` | `pod-logs-reader-role.yaml` |
| ClusterRole | `cr` | `test-runner-cr.yaml` |
| RoleBinding | `rb` | `pipeline-runner-rb.yaml` |
| ClusterRoleBinding | `crb` | `argocd-manager-crb.yaml` |
| ConfigMap | `cm` | `build-pipeline-config-cm.yaml` |
| Service | `svc` | `webhook-svc.yaml` |
| Deployment | `deploy` | `controller-deploy.yaml` |
| Namespace | `ns` | `build-service-ns.yaml` |
| ExternalSecret | `es` | `pipelines-as-code-es.yaml` |
| ServiceMonitor | `sm` | `controller-sm.yaml` |
| PrometheusRule | `pr` | `slo-alerts-pr.yaml` |

For custom resources without a standard abbreviation, use a short descriptive suffix: `release-plan-admission-rpa.yaml`, `build-pipeline-selector-bps.yaml`. When no sensible abbreviation exists, the full kind in lowercase is acceptable: `pac-repository.yaml`.

### 5.2 One Resource Per File

Each Kubernetes resource lives in its own YAML file. No multi-document files (`---` separators), no bundled manifests. This makes diffs atomic — a PR that modifies a RoleBinding only touches the RoleBinding file.

Correct — one resource per file:

```
base/
├── kustomization.yaml
├── build-service-ns.yaml                # Namespace
├── allow-argocd-to-manage-cr.yaml       # ClusterRole
├── pipeline-runner-rb.yaml              # RoleBinding
├── controller-sm.yaml                   # ServiceMonitor
└── external-secrets/                    # subdirectory for related group
    ├── kustomization.yaml
    ├── pipelines-as-code-es.yaml
    └── snyk-token-es.yaml
```

Wrong — multiple resources in one file:

```
base/
├── kustomization.yaml
└── resources.yaml                       # 4 resources with --- separators
```

- **Exception:** CRDs or upstream manifests pulled via `?ref=SHA` are not split — they are consumed as-is from upstream.
- **Subdirectories** for related groups (e.g. `external-secrets/`, `rbac/`, `monitoring/`) are encouraged when a tier has more than ~6 resource files.

### 5.3 Patches Directory

Strategic merge patches and JSON patches live in a `patches/` subdirectory, not mixed in with resource files. This separates *what gets deployed* (resources) from *what gets modified* (patches). When scanning a directory, you immediately know which files create resources and which modify them.

```
rings/ring-1/base/
├── kustomization.yaml
└── patches/
    ├── resource-limits-patch.yaml         # CPU/memory overrides
    ├── replica-count-patch.yaml           # Ring-specific replica count
    └── env-config-patch.yaml              # Environment-specific env vars
```

```
rings/ring-1/stone-stg-rh01/
├── kustomization.yaml
└── patches/
    ├── external-secret-path-patch.yaml    # Cluster-specific vault path
    └── deployment-resources-patch.yaml    # Cluster-specific resource sizing
```

Reference in kustomization.yaml:

```yaml
patches:
- path: patches/resource-limits-patch.yaml
  target:
    kind: Deployment
    name: controller-manager
- path: patches/replica-count-patch.yaml
  target:
    kind: Deployment
```

Rules:

- **Patch files end with `-patch.yaml`** — distinguishes them from resource files at a glance.
- **One patch per concern.** Don't bundle resource limits and replica counts in the same patch file.
- **Inline patches** (the `patch: |-` form in kustomization.yaml) are acceptable for single-line changes. Move to a file in `patches/` when the patch exceeds ~5 lines.
- **When a directory has only one patch**, the `patches/` subdirectory is still preferred for consistency, but a single patch file at the directory root (e.g. `resources_patch.yaml`) is tolerated.

> **Legacy Naming**
>
> Existing files like `resources_patch.yaml` at the directory root predate this convention. New components must follow the `patches/` directory pattern. Existing components should migrate when next modified — don't create PRs solely to rename files.

---

## 6. Validation

Before submitting a PR, verify that every tier builds successfully. Run `kustomize build` at the leaf (Tier 3) — this exercises the full chain (Tier 3 → Tier 2 → Tier 1 + Tier 4 features):

```bash
# Build a specific cluster overlay (Tier 3)
kustomize build --enable-helm components/{component}/rings/ring-N/{cluster}/

# Build the ring base (Tier 2) — useful when no Tier 3 exists yet
kustomize build --enable-helm components/{component}/rings/ring-N/base/

# Build the component base (Tier 1) — sanity check
kustomize build components/{component}/base/
```

If any of these fail, the layout is broken. Common errors:

- **`resource not found`** — a `resources:` entry references a file or directory that doesn't exist.
- **`no such file or directory`** — a `patches:` path is wrong, or `../base` reference points to the wrong level.
- **`namespace mismatch`** — a Component sets a different `namespace:` than its parent kustomization.
- **`multiple matches`** — a `target:` selector in a patch matches more than one resource. Narrow it with `name:` or `group:`.

For rollback and hotfix procedures, see [Rollback and Reliability](architecture.md#38-rollback-and-reliability) in the architecture doc.
