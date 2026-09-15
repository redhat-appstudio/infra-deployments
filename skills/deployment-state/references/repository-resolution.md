# Resolving Desired Deployment State

## Ensure a local clone

First locate an existing `infra-deployments` checkout. If none is available, create a
local clone before searching:

```bash
git clone https://github.com/redhat-appstudio/infra-deployments.git <safe-local-path>
cd <safe-local-path>
```

Choose an explicit empty destination in a suitable workspace or temporary directory.
Do not overwrite an existing directory. A local clone is required because effective-state
resolution relies on repository-wide search, Git history, inheritance tracing, and sometimes
Kustomize rendering.

## Establish the revision

Run `git fetch origin main`, then record `git rev-parse origin/main` and its commit time.
Use Git object commands such as `git grep <pattern> origin/main -- <paths>` and
`git show origin/main:<path>` so an unrelated checkout cannot affect the answer.

If rendering is necessary and the working tree is not at `origin/main`, materialize
`origin/main` in a temporary directory (for example with `git archive`) and render there.
Do not switch, reset, or modify the user's checkout. Preserve the temporary render when
it contains evidence needed by a challenger; otherwise remove only the known temporary
directory after the investigation.

## Resolve cluster to source path

Do not infer a cluster's deployment path solely from directory names or a global ring map.
Resolve it through the component's Argo CD ApplicationSet under `argo-cd-apps/`.

Normalize names cautiously. A workload or image name may differ from its component directory,
and an umbrella operator may own resources that previously had a standalone ApplicationSet.
Treat near-matching cluster names (for example `prod` versus `prd`) as distinct until exact
ApplicationSet evidence proves an alias. A cluster name appearing in a URL, secret, or config
value is not evidence that the workload targets that cluster.

When a standalone component is excluded, removed, or only targets part of the fleet, check
whether an umbrella operator now deploys that operand. Resolve the operator image to its
source with `working-with-provenance`, then inspect the operator source revision's embedded
manifests or dependency pin for the operand. Report standalone and operator-managed coverage
separately; do not assume one supersedes the other on every cluster.

1. Locate the component ApplicationSet and its `sourceRoot`, `environment`, `ring`, and
   `clusterDir` values.
2. Follow the overlay's `kustomization.yaml` and shared list patches to confirm that the
   requested cluster is targeted and to obtain its exact path.
3. For legacy layouts, expect paths under
   `components/<component>/{development,staging,production}/`, sometimes with `base` or
   per-cluster overlays.
4. For ring deployments, expect
   `components/<component>/rings/ring-N/{base,<cluster>}`. Ring assignments are
   component-specific. ApplicationSets currently track `main`; Kargo promotes by changing
   ring subdirectories through pull requests, not by assigning a branch per ring.
5. Confirm whether a similarly named legacy and `-rd` component coexist during migration.

Some ApplicationSet generators depend on Argo CD cluster secrets or other live inputs that
are not fully enumerated in this repository. Use checked-in selectors, list patches, and
overlays to identify what Git proves. If they do not enumerate the complete target set, say
that total fleet coverage cannot be proven from Git alone; do not silently treat discovered
cluster directories as exhaustive.

During migrations, distinguish a version present in a legacy environment definition from a
version proven to be consumed by a currently generated Application. Exclusion patches can
replace a standalone deployment with an operator-managed one. Describe the former as a
definition unless ApplicationSet evidence connects it to current targets.

Useful discovery commands include:

```bash
git grep -n -e '<component>' -e 'sourceRoot' -e 'clusterDir' -e 'values.ring' origin/main -- argo-cd-apps
git grep -n -e 'nameNormalized: <cluster>' -e 'values.clusterDir: <cluster>' origin/main -- argo-cd-apps
git ls-tree -r --name-only origin/main -- components/<component>
git grep -n -e 'newTag:' -e 'newName:' -e 'digest:' -e 'image:' -e 'helmCharts:' -e 'version:' -e '?ref=' origin/main -- components/<component>
```

Broad grep results are discovery leads. Confirm exact YAML fields and list elements before
using them as membership or inheritance evidence.

## Determine the effective pin

Follow `resources`, `components`, patches, image transforms, generators, and Helm values
from the exact cluster path. A nearby grep hit is a candidate, not necessarily effective
state. Cluster overlays can override environment or ring bases, and checked-in generated
YAML matters only when the active kustomization references it.

Render with:

```bash
kustomize build --enable-helm <exact-source-path>
```

Render when inheritance is non-trivial, an override is possible, or the question asks for
an exact effective result. Remote bases and Helm dependencies may require network access.
If rendering is blocked, report the traced definition and the unresolved uncertainty.

A component can comprise several first-party images or refs. Identify the primary workload
and include sidecars, monitors, dashboards, or dependencies only when they affect the
question. Do not assume correlated-looking pins have the same source without evidence.

## Resolve a pin to source

A direct Git or Kustomize ref may identify the source repository and commit. When an image's
source is not proven, load the separately available `working-with-provenance` skill and use
it to resolve the image to its repository and commit. A tag that resembles a Git SHA is not
proof of provenance.

Before comparing source revisions:

1. Confirm that both revisions belong to the same source repository.
2. Resolve abbreviated commits unambiguously and fetch enough source history.
3. Use `git merge-base --is-ancestor <older> <newer>` rather than comparing SHA strings,
   timestamps, or tags.
4. Record command exit codes and test both directions when explaining which revision
   predates the other.

If revisions are not linearly related, report that and use an appropriate merge base or
range rather than inventing an ordering.

## Compare results

Resolve and, when needed, render every requested cluster independently. Report a compact
matrix when comparing several targets. Cite paths at `origin/main`, and cite the line or
field that establishes each pin or inheritance edge. Classify absence carefully:

- **not targeted:** ApplicationSet evidence excludes the cluster;
- **definition not found:** no applicable definition was located;
- **unresolved:** migration, ownership, rendering, or unusual layout prevents a conclusion.

`hack/new-cluster/templates/` describes bootstrap state for future clusters. Consult it
only for bootstrap questions or explicit rollout-consistency checks; it is not the current
desired state of existing Argo CD-managed clusters.
