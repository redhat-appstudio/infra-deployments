---
name: production-ring-rollout
description: >
  Use when promoting a change to production across the 3 manual rings of
  clusters (e.g. tekton-kueue/kueue config, or any per-cluster kustomize
  overlay change) — covers ring membership, the temporary-override
  mechanism, and final-ring consolidation.
---

# Production Ring Rollout (manual, per-cluster kustomize overlays)

This is the **current, battle-tested** mechanism for rolling a change out to
all production clusters gradually. It is distinct from the Kargo-based
automated ring system described in `docs/ring-deployments/` (`architecture.md`,
`directory-layout.md`) — that system is for onboarding **new** components
across dev/staging/N generic rings and is still under active development.
This skill covers promoting a change to an **existing** component (e.g.
`kueue`, `tekton-kueue`) across the 3 named production rings below, using
plain per-cluster Kustomize overlays and 3 sequential PRs. See
`pr-workflow.md` for the PR template, branch/commit conventions, and CI
checks — this file only covers the ring-specific mechanics.

## Ring membership

Confirmed from the git history of two rollouts (`KFLUXINFRA-3973` token
resource, `KFLUXINFRA-3980` build-platforms type guard — see Precedents
below). Treat this table as current unless you find evidence of a cluster
being added/removed (new cluster additions show up as new
`components/kueue/production/<cluster>/` directories not listed here):

| Ring | Clusters |
|------|----------|
| 1 | `stone-prod-p01`, `kflux-ocp-p01`, `kflux-prd-rh02` |
| 2 | `kflux-osp-p01`, `kflux-prd-rh03`, `stone-prod-p02` |
| 3 (final) | `kflux-fedora-01`, `kflux-rhel-p01`, `stone-prd-rh01` |

To re-derive or verify this if it's ever in doubt:
```bash
git log --oneline --all | grep -iE "ring.[0-9]|ring-[0-9]"
git show <ring-N-commit> --stat   # lists the clusters touched by that ring's PR
```
`KFLUXINFRA-3973` ring PRs (#12529 ring 1, #12646 ring 2, #12923 ring 3) are
the clearest reference — each commit message explicitly names its clusters.

## The mechanism

Never change `components/kueue/production/base/tekton-kueue/config.yaml` (or
any shared `base`/shared policy kustomization) in a non-final ring — it's
inherited by every cluster that doesn't have its own override, so editing it
early would push the change to all 9 clusters at once, defeating the point
of rings.

**Per-cluster state before you start** — check whether each ring's clusters
already have a permanent override or inherit base:
```bash
find components/kueue/production/<cluster> -name config.yaml
cat components/kueue/production/<cluster>/kustomization.yaml   # look for configMapGenerator
```
- If the cluster has its own `config.yaml` + `configMapGenerator: behavior:
  replace` already (a **permanent** override, usually because it needs
  cluster-specific logic) — edit that file directly.
- If it inherits base (no own `config.yaml`) — create a **temporary**
  override: copy base's current content, apply your change, add it as
  `config.yaml` next to the cluster's `kustomization.yaml`, and add:
  ```yaml
  configMapGenerator:
    - name: tekton-kueue-config
      namespace: tekton-kueue
      behavior: replace
      files:
        - config.yaml
  ```

**For a new policy/VAP** (not just a config.yaml value): add it under
`components/policies/production/policies/kueue/<name>/`, but reference it
**only from the affected ring's cluster kustomizations**
(`components/policies/production/<cluster>/kustomization.yaml`, add
`- ../policies/kueue/<name>/` alongside the existing `../policies/kueue/`
line) — not from the shared `policies/kueue/kustomization.yaml`, which
every cluster includes.

**Test script**: `hack/test-tekton-kueue-config.py` has `CONFIG_COMBINATIONS`
(maps a name to a config file + kustomization file) and `TEST_COMBINATIONS`
(maps a PipelineRun fixture to a config combination, with optional
`expected` override). Add one `CONFIG_COMBINATIONS` entry per new
temporary/permanent override, and at least one `TEST_COMBINATIONS` entry
per ring exercising the new behavior.

## Final ring: consolidate

The last ring both applies the fix to its own 3 clusters *and* removes the
scaffolding from the earlier rings:

1. Apply the change directly to `base/tekton-kueue/config.yaml` (or the
   shared policy kustomization) — every cluster that inherits base now gets
   it automatically, covering 2 of this ring's 3 clusters for free.
2. Apply the change to this ring's permanent-override cluster(s) directly.
3. For every **temporary** override created in earlier rings: diff it
   against base first (`diff components/.../config.yaml
   components/kueue/production/base/tekton-kueue/config.yaml`) — it should
   be byte-identical now that base has the fix. Only then delete the
   override's `config.yaml` and remove the `configMapGenerator` block from
   its `kustomization.yaml`.
4. Fold any per-ring policy directory into the shared
   `policies/kueue/kustomization.yaml`, and remove the explicit
   `../policies/kueue/<name>/` line from every cluster kustomization that
   had it (rings 1 and 2's clusters).
5. Update the test script: delete `CONFIG_COMBINATIONS`/`TEST_COMBINATIONS`
   entries for removed overrides, add a case against the base `production`
   config key instead.

## Verification (every ring)

Confirm isolation — only the target ring's clusters should change:
```bash
for c in <all-9-clusters>; do
  echo -n "$c: "
  kustomize build components/kueue/production/$c | grep -c "<marker unique to your change>"
done
```
Do the same for `components/policies/production/$c` if a policy was added.
Non-target clusters must show `0`; target clusters must show the same
non-zero count. Then run `python3 hack/test-tekton-kueue-config.py` (needs
podman) — expect `OK`.

## Precedents

- `KFLUXINFRA-3973` (kueue token resource): ring 1 #12529, ring 2 #12646,
  ring 3 #12923 (final-ring consolidation).
- `KFLUXINFRA-3980` (build-platforms CEL type guard): dev/staging #12893,
  ring 1 #13059, ring 2 #13096, ring 3 #13138 (final-ring consolidation,
  same pattern as above).
