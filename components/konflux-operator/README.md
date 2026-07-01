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
| 1 | `base/` | Minimal shared anchor (empty `resources: []`). |
| 2 | `rings/ring-N/base/` | Upstream ref, image pin, `Konflux` CR, and ring-specific overlay patches under `cr/`. This is where Kargo writes promoted content. |
| 3 | `rings/ring-N/<cluster>/` | Per-cluster overlays that reference `../base`. |
| 4 | `rings/ring-N/base/cr/<team>/` | Per-ring, per-team `Konflux` CR overlay patches (`kind: Component`). These are **ring-specific** and not promoted between rings. |

Example (`ring-0`):

```text
base/
  kustomization.yaml              # resources: [] (shared anchor)
rings/
  ring-0/
    base/
      kustomization.yaml          # remote operator + images + components: cr/...
      konflux.yaml                # minimal Konflux CR shell
      patches/
        release-config-patch.yaml # cluster-invariant spec for this release line
      cr/
        build/
        konflux-ui/
        image-controller/
        ...
```

## What gets promoted across rings

Only the **invariant content** in `rings/ring-N/base/` is promoted by Kargo:
- Upstream remote ref (the `?ref=` in `resources`)
- Image tags (the `images` block)
- Base `Konflux` CR (`konflux.yaml`) and invariant patches (`patches/`)

The `cr/` overlay patches are **ring-specific** — each ring owns its own set.
They are not overwritten during promotion.

## Promoting to another ring

1. Kargo promotes the invariant content (upstream ref, image pin) from one ring's
   `base/` to the next ring's `base/`.
2. `cr/` overlay patches in the target ring remain unchanged unless that ring
   genuinely needs different configuration.
3. **Rollback** is a normal Git operation (`git revert`, or restore the invariant
   from an earlier revision).

Adding a new ring: create `rings/ring-N/base/` with the same shape, seed
the invariant from the ring you trust, then maintain `cr/` patches for that ring.

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
instance, temporarily remove the CR inputs from `rings/ring-0/base/kustomization.yaml`
(for example `konflux.yaml` and the `patches` entry that references
`release-config-patch.yaml`), apply, then restore those lines when you want the instance.

## Konflux CR ownership

- **`rings/ring-N/base/konflux.yaml`** — minimal `Konflux` object.
- **`rings/ring-N/base/patches/release-config-patch.yaml`** — cluster-invariant `spec`
  carried with the operator pin for that release line.
- **`rings/ring-N/base/cr/*/OWNERS`** — team ownership for overlay fragments under each
  subdirectory.
