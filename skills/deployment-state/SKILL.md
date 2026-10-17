---
name: deployment-state
description: >
  Use when determining deployment state of konflux clusters, comparing
  component versions across environments or clusters, checking whether a source
  fix has rolled out, or tracing a rollout back to source changes and bug fixes.
---

# Deployment State

Determine what `origin/main` says should be deployed. This is a read-only investigation
skill: do not change deployment configuration.

This repository represents the **desired Git state**, not the live Argo CD sync
state or Kubernetes runtime state.

## Repository prerequisite

Work from a local clone so repository-wide search, history, and rendering are available.
If `infra-deployments` is not already present, clone
`https://github.com/redhat-appstudio/infra-deployments.git` into a suitable workspace or
temporary directory and enter it. Reuse an existing clone when available; do not assume
the skill was invoked from inside one. Then fetch `origin main` before investigating.
Do not clone over an existing path or alter a user's working tree.

Start every investigation by reading [references/repository-resolution.md](references/repository-resolution.md).

## Scope and confidence

- State the exact commits you examined. Do not switch branches or alter the
  working tree.
- Treat the merge time of a deployment change on `origin/main` as the desired-state
  rollout time. Do not imply that Argo CD finished reconciling at that time.
- If the question requires proof of live state, say that live Argo CD or Kubernetes
  confirmation is needed. Do not perform it as part of this skill.
- Prefer the smallest investigation that answers the question. Versions may include
  image tags or digests, Git/Kustomize refs, Helm chart versions, and multiple workload
  images, but do not load all of them into context unless they matter.

## Choose a workflow

- **Compare current desired state across clusters or environments:** resolve each exact
  deployment path, compare the effective pins, and cite their defining files. The
  repository-resolution reference is usually sufficient.
- **Known source fix -> rollout coverage and timing:** read
  [references/fix-to-rollout.md](references/fix-to-rollout.md).
- **Cluster rollout -> component/revision inventory:** read
  [references/rollout-inventory.md](references/rollout-inventory.md). Stop at the
  concise inventory unless more interpretation is requested.
- **Rollout -> source commits or pull requests:** read
  [references/source-change-range.md](references/source-change-range.md).
- **Rollout -> interpreted bug fixes:** read
  [references/interpret-bug-fixes.md](references/interpret-bug-fixes.md). Distinguish
  linked/confirmed fixes from inferred bug-related changes.
- **When and why clusters diverged:** read
  [references/divergence-history.md](references/divergence-history.md).

When an image must be mapped to its source repository and commit, use the separately
available `working-with-provenance` skill. Do not reproduce or improvise its attestation
commands here. A tag that resembles a Git SHA is not proof of provenance.

## Scale and verification

For a broad cluster/time-window request, first return or establish a concise inventory
of affected components and revisions. Investigate source history only for the relevant
components. Use sub-agents when multi-component history or source interpretation would
substantially bloat context.

Broad multi-cluster results, historical conclusions, ancestry checks, provenance-derived
mappings, and interpreted bug-fix findings require the challenger pass in
[references/evidence-and-challenge.md](references/evidence-and-challenge.md). A simple
current-state lookup can be self-verified against cited repository evidence.

## Answer contract

Include only relevant fields, but make these facts unambiguous:

- `origin/main` commit examined
- component and requested cluster/environment
- effective desired version or source revision, with evidence paths
- whether the result describes a stored definition or a Git-proven actively targeted path
- desired-state merge time when the question is historical
- ancestry result when checking whether a fix is included
- confirmed versus inferred source-level fixes
- exclusions, unresolved mappings, or ambiguity
- stable, revision-pinned evidence links for historical or challenged conclusions
- the desired-state/live-state confidence boundary
- use full github.com urls so that users can navigate to your evidence to understand

Never translate “definition not found” directly into “not deployed.” Distinguish a cluster
that is explicitly not targeted, a missing definition, and an unresolved or unusual layout.
