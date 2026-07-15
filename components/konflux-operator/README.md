# Konflux Operator Component

This directory holds the **manifests** used by Argo CD on OpenShift to install the
[Konflux operator](https://github.com/konflux-ci/konflux-ci) and to define a default
`Konflux` custom resource (instance configuration). Cluster operators and maintainers
edit these files; reviewers use this file to understand layout and promotion rules.

The operator install bundle is consumed as **plain Kubernetes manifests** (CRDs, RBAC,
Deployment, and so on), not as OLM `Subscription` / `ClusterServiceVersion` objects.

## Directory layout (ring-based)

Deployments follow a **ring-based Kustomize structure** with Kargo-driven promotions:

| Tier | Path | Purpose |
|------|------|---------|
| 1 | `rings/ring-N/base/` | Upstream ref, image pin, `Konflux` CR, CR overlay patches, and release config. Each ring owns its full configuration. |
| 2 | `rings/ring-N/<cluster>/` | Per-cluster overlays that reference `../base`. |

Example (`ring-0`):

```text
rings/
  ring-0/
    base/
      kustomization.yaml          # resources: [invariant], components: cr/..., patches: [release-config.yaml]
      release-config.yaml         # environment-specific spec (defaultTenant, certManager, etc.)
      cr/                         # Konflux CR overlay patches (ring-owned)
        build/
        konflux-ui/
        image-controller/
        ...
      invariant/
        kustomization.yaml        # remote operator + images + konflux.yaml
        konflux.yaml              # minimal Konflux CR shell
```

## Why `invariant/` is a separate subdirectory

`invariant/` is the **Kargo promotion boundary** — the only directory Kargo writes
to when promoting a new operator version across rings. Isolating it lets Kargo
update the upstream ref and image pin without touching ring-authored CR components
or environment config. `konflux.yaml` must remain inside `invariant/` because
kustomize requires resources to reside within the kustomization directory boundary;
it cannot reference files outside its own directory tree.

## What gets promoted across rings

Only the **invariant content** in `rings/ring-N/base/invariant/` is promoted by Kargo:
- Upstream remote ref (the `?ref=` in `resources`)
- Image tags (the `images` block)
- Base `Konflux` CR (`konflux.yaml`)

The `cr/` overlay patches and `release-config.yaml` live inside each ring's `base/`
directory and are **ring-specific**. When adding a new ring, copy or adapt them as needed.

**Where does my change go?**

| Change | Path |
|--------|------|
| Operator version / upstream ref | `rings/ring-N/base/invariant/kustomization.yaml` |
| Platform settings (certManager, defaultTenant, …) | `rings/ring-N/base/release-config.yaml` |
| Team service config | `rings/ring-N/base/cr/<service>/` |
| Cluster-specific override | `rings/ring-N/<cluster>/` |

## Promoting to another ring

1. Kargo promotes the invariant content (upstream ref, image pin) from one ring's
   `base/invariant/` to the next ring's `base/invariant/`.
2. `cr/` overlay patches and `release-config.yaml` are ring-local — adapt per ring as needed.
3. **Rollback** is a normal Git operation (`git revert`, or restore the invariant
   from an earlier revision).

Adding a new ring: create `rings/ring-N/base/` with the same shape, seed
the invariant from the ring you trust, and copy or adapt `cr/` patches and `release-config.yaml`.

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
instance, apply the invariant layer directly after temporarily removing `konflux.yaml`:

1. Remove the `konflux.yaml` resource from `rings/ring-0/base/invariant/kustomization.yaml`
2. `kubectl apply -k components/konflux-operator/rings/ring-0/base/invariant`
3. Restore `konflux.yaml` in the kustomization when you want the instance

Do not apply `rings/ring-0/base` without the `Konflux` CR, as the CR components
and patches at that level require it as a target.

## CI (E2E testing with the operator overlay)

The CI overlay for operator-based E2E tests lives in
[`ci/openshift-overlay-e2e/`](ci/openshift-overlay-e2e/README.md). See its README
for how the overlay is wired into Prow presubmit jobs and what it validates.

`hack/preview.sh --operator-overlay` patches
`rings/ring-0/base/cr/image-controller/image-controller.yaml` to enable or
disable image-controller (`.spec.imageController.enabled`) based on whether
`IMAGE_CONTROLLER_QUAY_ORG` and `IMAGE_CONTROLLER_QUAY_TOKEN` are set in
`hack/preview.env`. It does not inject credential values into the CR — it only
toggles the boolean flag; the credentials themselves are consumed from the env
file by other parts of the preview flow. This edit is local to the preview run
and is not committed.

## Konflux CR ownership

- **`rings/ring-N/base/invariant/konflux.yaml`** — minimal `Konflux` object.
- **`rings/ring-N/base/release-config.yaml`** — environment-specific `spec`
  (defaultTenant, certManager, internalRegistry, etc.).
- **`rings/ring-N/base/cr/*/OWNERS`** — team ownership for overlay fragments under each
  subdirectory.
