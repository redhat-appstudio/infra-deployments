---
name: risk-assessment
description: >
  Use when drafting or reviewing the Risk Assessment section of an
  infra-deployments PR — especially production or production-downstream
  changes, and when reviewing opaque component ref/image bumps.
  Provides level heuristics, blast-radius rules, and rollback guidance
  matched to this repo's PR conventions.
---

# Risk Assessment

How to write or review `## Risk Assessment` for infra-deployments PRs.

For the full PR lifecycle (What / Why / Validation), see `skills/pr-workflow.md`.
This skill owns **Risk Assessment quality only**.

## Author and reviewer obligations

| Role | Obligation |
|------|------------|
| **Author** | When creating or updating a PR, include a `## Risk Assessment` section whenever **When to Use** applies. Use the required fields and rubric below. Base the text on the PR diff paths and Validation evidence. |
| **Reviewer** | Check that production / production-downstream PRs have an adequate Risk Assessment; for component bumps, assess risk from the upstream changelog even when the section is optional (e.g. staging). |

## When to Use

- Drafting a PR that touches:
  - `components/**/production/**` or `production-downstream`
  - `argo-cd-apps/overlays/` targets such as `production-downstream`, `rd-production`, `konflux-public-production`
  - prod-affecting `configs/`
- Reviewing whether a production Risk Assessment is adequate
- Promoting staging → production and need a risk section
- Reviewing component ref/image bumps (bot or human) whose risk is in the upstream changelog

## When to Skip

| Case | Action |
|------|--------|
| Pure development / staging-only with no meaningful upstream delta (docs-only, CODEOWNERS-only) | Optional — omit or keep brief |
| Auto-merged PRs marked "Do not commit directly" (no human review expected) | Skip |
| Docs-only / `skills/`-only / infra-tools tests with no cluster deploy | Optional |

## Required Fields

```markdown
## Risk Assessment

**Risk Level:** Low / Medium / High / Very High
**What could go wrong:** <concrete failure mode if this change is wrong>
**Rollback:** <how to undo — usually Revert PR + Argo sync; note exceptions>
**Blast radius:** <environments and named clusters>
```

`**Description:**` (seen in some templates) means the same as **What could go wrong** — prefer the latter for new PRs. Merged PRs sometimes omit blast radius; **new** PRs should include it.

## Component ref / image bumps

Opaque bumps (e.g. `?ref=` / `newTag` SHA changes under `components/*/`) look tiny in the infra-deployments diff. **Risk is in the upstream changelog**, not the YAML line count.

1. Read Included PRs / Changelog on the PR body (or compare old→new upstream SHAs).
2. Rank the *highest*-risk upstream change using the rubric (not the average). Signals that raise level:
   - RBAC / ClusterRole / `bind` / impersonation
   - Auth, TLS/CA, secrets, admission, CRDs, operators
   - Build/pipeline-critical behavior changes
3. Routine dep bumps, CODEOWNERS, or ref-only churn → usually **Low** if nothing above appears.
4. Blast radius = overlays/envs in *this* PR (dev/staging soak is still worth stating).
5. Staging-only: a full `## Risk Assessment` section is optional, but the review comment should still call out Medium/High upstream signals so they are not missed before prod.

## Procedure

1. **Classify environment** from changed paths (`development` / `staging` / `production` / `production-downstream`, plus targets like `rd-production` / `konflux-public-production` when present).
2. **List blast radius** — name clusters from overlay dirs already in the diff.
3. **Draft failure mode** (one or two sentences tied to *this* diff — or upstream changelog for bumps) **then pick Risk Level** from the rubric. Do not default to Low.
4. **Write Rollback** — GitOps default is revert; call out exceptions below.
5. If promoting to prod and Validation lacks staging evidence, note that when finishing the PR body (Validation is owned by `pr-workflow.md`).

If level is unclear, ask **one** clarifying question rather than inventing severity.

## Risk Level Rubric

| Level | Typical changes | Examples |
|-------|-----------------|----------|
| **Low** | Additive allowlists/labels; resource tweaks with staging proof; chart/image bumps already validated in staging; single optional component | LabelKeep allowlist; exporter memory |
| **Medium** | Broader prod config; RBAC expansion; monitoring/pipeline config that can drop metrics or delay builds; first prod roll of a non-trivial feature; multi-cluster promote with thin soak; component bump whose changelog includes RBAC or config/security changes | New exporter on many clusters; widened maintainer RBAC; integration-service bump granting ClusterRole `bind` |
| **High** | Auth / SSO / Dex; network policies; Kyverno/admission; CRD or operator upgrades; pipeline-service / build-critical paths; etcd/kubelet/`configs/`; deleting prod resources | Dex client changes; ClusterPolicy |
| **Very High** | Irreversible migration; platform-wide outage potential; critical change across many prod clusters with no prior staging evidence | Critical component change with empty Validation |

**Adjustments:**
- Strong staging evidence → may lower Medium → Low for routine promotes.
- No staging validation on a production change → raise at least one level.
- Hotfix during an outage may stay Low/Medium if blast radius is small and revert is safe — still name a real failure mode.
- Multi-cluster alone does not force Very High — use the change type and Validation evidence.
- For ref/image bumps, level follows the **riskiest** upstream commit in the range, not "it's only a SHA bump."

## Blast Radius

Derive from paths in the PR, not guesses:

- `components/<name>/production/<cluster>/` → list those clusters
- `components/<name>/production/base/` → all clusters consuming that base (say so)
- `production-downstream` / `rd-production` / `konflux-public-production` → name targets from overlay paths

## Rollback

| Situation | Text |
|-----------|------|
| Normal GitOps change | `Revert PR` (ArgoCD resyncs previous desired state) |
| Chart/image bump | `Revert PR` to previous version; name the old version if known |
| CRDs / operators | Note CRD version may remain after revert; add manual steps if any |
| ExternalSecrets / one-way deletes | Revert may not restore secret data or deleted objects — say so |

## Example (fill level from the rubric)

```markdown
## Risk Assessment

**Risk Level:** <Low / Medium / High / Very High — from rubric>
**What could go wrong:** Chart upgrade includes changes beyond the intended fix and could affect proxy/caching behavior on the listed clusters.
**Rollback:** Revert PR to restore previous version (0.1.1688).
**Blast radius:** production clusters touched by this PR: kflux-fedora-01, kflux-osp-p01, stone-prod-p01.
```

Vary failure mode and level for config, RBAC/auth, or hotfixes — do not copy Low from examples.

## Reviewer Checks

Request changes when (enforcement is review-time, not guaranteed CI):

- Multi-cluster production PR omits blast radius (clusters or "base → all consumers")
- Level is Low with no concrete failure mode
- Production promote has empty Validation (no staging evidence)
- Component bump: Medium/High signals in the changelog (RBAC, auth/CA, CRDs) are ignored or treated as Low because the infra diff is small

## Anti-Patterns

| Bad | Better |
|-----|--------|
| Low with no explanation | Name a real failure mode |
| "Could break production" | Say *what* (auth outage, 502s, metric loss, build queue) |
| `Rollback: N/A` | Almost always `Revert PR`; document exceptions |
| Defaulting every change to Low | Use the rubric |
| Calling every multi-cluster PR Very High | Rank by change type + staging evidence, not cluster count alone |
| Treating a SHA-only bump as Low without reading the changelog | Rank from upstream commits (e.g. ClusterRole `bind` → at least Medium) |
