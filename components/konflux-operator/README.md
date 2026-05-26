# Konflux Operator Component

This directory holds the **manifests** used by Argo CD on OpenShift to install the
[Konflux operator](https://github.com/konflux-ci/konflux-ci) and to define a default
`Konflux` custom resource (instance configuration). Cluster operators and maintainers
edit these files; reviewers use this file to understand layout and promotion rules.

The operator install bundle is consumed as **plain Kubernetes manifests** (CRDs, RBAC,
Deployment, and so on), not as OLM `Subscription` / `ClusterServiceVersion` objects.

## Directory layout per environment

Each **environment** (for example `development/`, and later `staging-ring-a/` or
similar) is a self-contained Kustomize root that Argo syncs. The layout is the same for
every environment:

| Path | Purpose |
|------|--------|
| `invariant/` | Operator pin (remote ref + `images`) plus **cluster-invariant** `Konflux` CR fragments. This is the **only** directory you replace as a unit when promoting a release line from one environment to another. |
| `cr/overlay-patches/<team>/` | **Per-environment or per-team** strategic-merge fragments for the same `Konflux` object (`metadata.name: konflux`). These files are **not** overwritten during a normal promotion. |

Example (`development/`):

```text
development/
  kustomization.yaml          # resources: [invariant]; components: cr/overlay-patches/...
  invariant/
    kustomization.yaml        # remote operator + images + local CR inputs
    konflux.yaml              # minimal Konflux CR shell
    release-config.yaml       # merged spec shared across clusters for this release line
  cr/
    overlay-patches/
      build/
      konflux-ui/
      ...
```

You may split a subtree such as `spec.buildService` across **`invariant/`**
(release-wide defaults) and **`overlay-patches/build/`** (environment- or team-specific
keys), as long as you avoid conflicting duplicate leaf values; see Kustomize
strategic-merge ordering if two patches touch the same field.

## Promoting to another environment

1. Copy or reconcile **only** `<target-env>/invariant/` with the content from
`<source-env>/invariant/` (or edit those files so the pull request shows exactly what
changed).
2. Leave `<target-env>/cr/overlay-patches/` unchanged unless the target cluster
genuinely needs different patches.
3. **Rollback** is a normal Git operation (`git revert`, or restore `invariant/` from
an earlier revision).

Adding a new environment: create a sibling directory with the same shape as
`development/`, seed `invariant/` once from the environment you trust, then maintain
`cr/overlay-patches/` for that cluster class.

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

`kubectl apply -k components/konflux-operator/development`

To sync **only** the operator controller and CRDs without applying the `Konflux`
instance, temporarily remove the CR inputs from `invariant/kustomization.yaml`
(for example `konflux.yaml` and the `patches` entry that references
`release-config.yaml`), apply, then restore those lines when you want the instance.

## Konflux CR ownership

- **`invariant/konflux.yaml`** — minimal `Konflux` object.
- **`invariant/release-config.yaml`** — cluster-invariant `spec` carried with the
operator pin for that release line.
- **`cr/overlay-patches/*/OWNERS`** — team ownership for overlay fragments under each
subdirectory.
