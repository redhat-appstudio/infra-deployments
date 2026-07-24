---
name: risk-assessment
description: >
  Use when drafting or reviewing the Risk Assessment section of an
  infra-deployments PR — especially production, ring, or production-downstream
  changes. Provides level heuristics, blast-radius rules, and rollback guidance
  matched to this repo's PR conventions. Fullsend coder/review agents discover
  this via .agents/skills (and .claude/skills symlink).
---

# Risk Assessment

How to write or review `## Risk Assessment` for infra-deployments PRs.

For the full PR lifecycle (What / Why / Validation / rings), see `skills/pr-workflow.md`.
This skill owns **Risk Assessment quality only**.

## Agent obligations (Fullsend)

| Role | Obligation |
|------|------------|
| **coder** | When creating or updating a PR, include a `## Risk Assessment` section whenever **When to Use** applies. Use the required fields and rubric below. Use PR diff paths and Validation evidence already in context. |
| **review** | Check that production / ring / production-downstream PRs have an adequate Risk Assessment; request changes using **Reviewer Checks** when missing or weak. |
| **triage** | No requirement to write Risk Assessment on issues. |

## When to Use

- Drafting a PR that touches:
  - `components/**/production/**` or `production-downstream`
  - ring layouts: `components/*/rings/ring-N/` (Kargo) or `components/*/production/rings/` (legacy)
  - `argo-cd-apps/overlays/` targets such as `production-downstream`, `rd-production`, `konflux-public-production`
  - prod-affecting `configs/`
- Reviewing whether a production Risk Assessment is adequate
- Promoting staging → production and need a risk section

## When to Skip

| Case | Action |
|------|--------|
| Pure development / staging-only | Optional — omit or keep brief |
| Automated Kargo promotions (`chore(ring-N): promote …`, auto-merged, "Do not commit directly", often `automated-promotion`) | Skip |
| Docs-only / `skills/`-only / infra-tools tests with no cluster deploy | Optional |

## Required Fields

```markdown
## Risk Assessment

**Risk Level:** Low / Medium / High / Very High
**What could go wrong:** <concrete failure mode if this change is wrong>
**Rollback:** <how to undo — usually Revert PR + Argo sync; note exceptions>
**Blast radius:** <environments, rings, and named clusters>
```

`**Description:**` (seen in some templates) means the same as **What could go wrong** — prefer the latter for new PRs. Merged PRs sometimes omit blast radius; **new** PRs should include it.

## Procedure

1. **Classify environment** from changed paths (`development` / `staging` / `production` / `production-downstream` / ring).
2. **List blast radius** — name clusters from overlay dirs. Say "this ring only" or "remaining clusters after canary/prior ring" if applicable.
3. **Draft failure mode** (one or two sentences tied to *this* diff) **then pick Risk Level** from the rubric. Do not default to Low.
4. **Write Rollback** — GitOps default is revert; call out exceptions below.
5. If promoting to prod and Validation lacks staging/prior-ring links, note that when finishing the PR body (Validation is owned by `pr-workflow.md`).

If level is unclear, ask **one** clarifying question rather than inventing severity.

**Ring policy (team convention — CI does not enforce intra-prod ring splits; CI only blocks mixing staging + production in one PR):**

- Prefer gradual rollout: canary/prior-ring PR, then remaining clusters — or component ring layouts where they exist.
- **First** full rollout of a **critical** component to all prod clusters with **no** prior canary/ring evidence → recommend splitting; if the author proceeds, treat as **High** or **Very High**.
- **Remaining** clusters after a successful canary/prior ring (with staging proof) may stay **Low** / **Medium** per the rubric — multi-cluster alone does not force Very High.

## Risk Level Rubric

| Level | Typical changes | Examples |
|-------|-----------------|----------|
| **Low** | Additive allowlists/labels; resource tweaks with staging proof; chart/image bumps already validated in staging (+ prior ring/canary); single optional component | LabelKeep allowlist; exporter memory; caching-helm after staging + canary |
| **Medium** | Broader prod config; RBAC expansion; monitoring/pipeline config that can drop metrics or delay builds; first prod roll of a non-trivial feature; multi-cluster promote with thin soak | New exporter on many clusters; widened maintainer RBAC |
| **High** | Auth / SSO / Dex; network policies; Kyverno/admission; CRD or operator upgrades; pipeline-service / build-critical paths; etcd/kubelet/`configs/`; deleting prod resources | Dex client changes; ClusterPolicy |
| **Very High** | First all-prod critical change with no canary/prior ring; irreversible migration; platform-wide outage potential | Critical component to every `production/<cluster>` with no prior prod evidence |

**Adjustments:**
- Staging evidence + prior ring/canary success → may lower Medium → Low for routine promotes.
- No staging validation on a production change → raise at least one level.
- Hotfix during an outage may stay Low/Medium if blast radius is small and revert is safe — still name a real failure mode.

## Blast Radius

Derive from paths, not guesses:

- `components/<name>/production/<cluster>/` → list those clusters
- `components/<name>/production/base/` → all clusters consuming that base (say so)
- `components/<name>/rings/ring-N/` or `production/rings/` → "ring-N only" + cluster list if known
- `production-downstream` / `rd-production` / `konflux-public-production` → name targets from overlay paths

## Rollback

| Situation | Text |
|-----------|------|
| Normal GitOps change | `Revert PR` (ArgoCD resyncs previous desired state) |
| Chart/image bump | `Revert PR` to previous version; name the old version if known |
| CRDs / operators | Note CRD version may remain after revert; add manual steps if any |
| ExternalSecrets / one-way deletes | Revert may not restore secret data or deleted objects — say so |
| Kargo-managed ring promotions | Prefer promoting a prior known-good Freight (see `docs/ring-deployments/architecture.md`); "Revert PR" alone may not apply |

## Example (fill level from the rubric)

```markdown
## Risk Assessment

**Risk Level:** <Low / Medium / High / Very High — from rubric>
**What could go wrong:** Chart upgrade includes changes beyond the intended fix and could affect proxy/caching behavior on the listed clusters.
**Rollback:** Revert PR to restore previous version (0.1.1688).
**Blast radius:** 8 production clusters: kflux-fedora-01, kflux-osp-p01, … (remaining after prior canary: kflux-ocp-p01).
```

Vary failure mode and level for config, RBAC/auth, or hotfixes — do not copy Low from examples.

## Reviewer Checks

Request changes when (enforcement is review-time, not guaranteed CI):

- Multi-cluster production PR omits blast radius (clusters or "base → all consumers")
- Level is Low with no concrete failure mode
- Production promote has empty Validation (no staging / prior-ring evidence)
- First all-prod critical change with no canary/prior ring — ask to split or raise to High/Very High

## Anti-Patterns

| Bad | Better |
|-----|--------|
| Low with no explanation | Name a real failure mode |
| "Could break production" | Say *what* (auth outage, 502s, metric loss, build queue) |
| `Rollback: N/A` | Almost always `Revert PR` (or Kargo prior Freight); document exceptions |
| Defaulting every change to Low | Use the rubric |
| Calling every multi-cluster PR Very High | Distinguish canary-then-remaining vs first full critical rollout |
| Skipping gradual rollout and only stamping Very High | Prefer canary/rings first when the component supports it |
