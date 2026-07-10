# Konflux Operator Component

This directory holds the **manifests** used by Argo CD on OpenShift to install the
[Konflux operator](https://github.com/konflux-ci/konflux-ci) and to define a default
`Konflux` custom resource (instance configuration). Cluster operators and maintainers
edit these files; reviewers use this file to understand layout and promotion rules.

The operator install bundle is consumed as **plain Kubernetes manifests** (CRDs, RBAC,
Deployment, and so on), not as OLM `Subscription` / `ClusterServiceVersion` objects.

## Directory layout (ring-based)

Deployments follow a **4-tier Kustomize structure** with Kargo-driven promotions:

| Tier | Path | Purpose |
|------|------|---------|
| 1 | `base/` | Shared anchor and common Konflux CR overlay patches (`cr/`). |
| 2 | `rings/ring-N/base/` | Upstream ref, image pin, `Konflux` CR. This is where Kargo writes promoted content. References `base/cr/` components. |
| 3 | `rings/ring-N/<cluster>/` | Per-cluster overlays that reference `../base`. |

Example (`ring-0`):

```text
base/
  kustomization.yaml              # resources: [] (shared anchor)
  cr/                             # Konflux CR overlay patches (shared across rings)
    build/
    konflux-ui/
    image-controller/
    ...
rings/
  ring-0/
    base/
      kustomization.yaml          # resources: [invariant], components: ../../../base/cr/...
      invariant/
        kustomization.yaml        # remote operator + images + konflux.yaml
        konflux.yaml              # minimal Konflux CR shell
        release-config.yaml       # cluster-invariant spec for this release line
```

## What gets promoted across rings

Only the **invariant content** in `rings/ring-N/base/` is promoted by Kargo:
- Upstream remote ref (the `?ref=` in `resources`)
- Image tags (the `images` block)
- Base `Konflux` CR (`konflux.yaml`) and release config (`release-config.yaml`)

The `cr/` overlay patches live in `base/cr/` and are **shared across all rings**.
If a CR needs ring-specific configuration in the future, it can be moved back to per-ring directories.

## Promoting to another ring

1. Kargo promotes the invariant content (upstream ref, image pin) from one ring's
   `base/` to the next ring's `base/`.
2. `cr/` overlay patches in `base/cr/` are shared — all rings use the same set.
3. **Rollback** is a normal Git operation (`git revert`, or restore the invariant
   from an earlier revision).

Adding a new ring: create `rings/ring-N/base/` with the same shape, seed
the invariant from the ring you trust. CR patches from `base/cr/` are inherited automatically.

## Preview script and `Konflux` readiness

`hack/preview.sh` supports **`--operator-overlay`** (OpenShift preview using the
`development-operator` Argo overlay). **By default it does not wait** for the cluster
`Konflux` object `konflux` to become ready, so preview can finish after Argo CD sync
while the operator and instance are still converging.

To **gate** preview on a healthy instance (same checks as
`konflux-ci/scripts/deploy-local.sh`), set **`PREVIEW_WAIT_KONFLUX_CR_READY=true`**
for that run.

## Applying manifests locally (optional)

From the repository root, after logging in to a cluster:

`kubectl apply -k components/konflux-operator/rings/ring-0/base`

To sync **only** the operator controller and CRDs without applying the `Konflux`
instance, temporarily remove the CR inputs from `rings/ring-0/base/invariant/kustomization.yaml`
(for example `konflux.yaml` and `release-config.yaml`), apply, then restore those lines
when you want the instance.

## Konflux CR ownership

- **`rings/ring-N/base/invariant/konflux.yaml`** — minimal `Konflux` object.
- **`rings/ring-N/base/invariant/release-config.yaml`** — cluster-invariant `spec`
  carried with the operator pin for that release line.
- **`base/cr/*/OWNERS`** — team ownership for overlay fragments under each
  subdirectory.
