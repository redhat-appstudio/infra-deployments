---
name: pr-workflow
description: >
  Use when opening, monitoring, or iterating on a pull request in infra-deployments,
  including PR body template, CI interpretation, production ring requirements,
  commit conventions, and environment-specific validation.
---

# PR Workflow

Full lifecycle reference for pull requests in the infra-deployments repository.

## Overview

infra-deployments is a GitOps monorepo deploying 50+ Kubernetes components via Kustomize and ArgoCD.
PRs always target the upstream repository (`redhat-appstudio/infra-deployments`), never the fork.

## When to Use

- About to open a PR or write a PR description
- Preparing a production change and unsure about ring requirements
- Promoting a change from dev/staging to production

## Branch Setup

If a dedicated branch already exists for this work, use it. Otherwise, create a branch from the latest main of `redhat-appstudio/infra-deployments`.

First, find which remote points to `redhat-appstudio/infra-deployments`:

```bash
git remote -v | grep redhat-appstudio/infra-deployments
```

Then fetch and branch from it:

```bash
git fetch <remote>
git checkout -b <branch-name> <remote>/main --no-track
```

Common setups: fork-based workflows typically name it `upstream`, while direct clones use `origin`.

Never branch from an old or diverged main.

## PR Body Template

Every PR follows this structure:

```markdown
## What

<concise list of what changed>

Clusters affected: <list clusters>

[KFLUXINFRA-1234](https://redhat.atlassian.net/browse/KFLUXINFRA-1234)

## Why

<motivation — why this change is needed>

## Validation

- `kustomize build --enable-helm` passes for all affected overlays
- <staging evidence, links to prior ring PRs, test results>

## Risk Assessment

**Risk Level:** Low / Medium / High / Very High
**What could go wrong:** <describe what breaks if this change is incorrect>
**Rollback:** Revert PR / <specific rollback steps>
<blast radius — how many clusters, which environments>
```

**Rules:**
- **What** — Concise change list, affected clusters, Jira link at the bottom. Don't explain why here.
- **Why** — Motivation only. Keep it brief.
- **Validation** — Proof, not explanation. kustomize build results, staging links, prior ring PRs.
- **Risk Assessment** — Required for production PRs. May be omitted for dev/staging.

## Commit Conventions

Prefix every commit with the Jira key:

```
KFLUXINFRA-1234 short description of the change
```

Trailers (at end of commit message body). Use the actual agent/tool identity:
- Interactive sessions (human + agent): `Assisted-by:` trailer
- Agentic workflow (autonomous): `Authored-by:` trailer

## Pre-Push Validation

Run `kustomize build --enable-helm` on each affected directory containing a `kustomization.yaml`. You can also run `render-diff` locally to see the full rendered manifest delta across affected components:

```bash
cd infra-tools && make build && ./bin/render-diff
```

Mention the results in the Validation section of the PR body.

## Key CI Checks

| Check | Triggers On | What It Does |
|-------|-------------|--------------|
| **Ring deployment enforcement** | `components/`, `argo-cd-apps/`, `configs/` | Enforces staging/prod separation — fails if staging and production changes are mixed in the same PR. Apply `skip-ring-deployment/hotfix` label to bypass for critical hotfixes. |
| **Render-diff** | All PRs | Posts a rendered Kubernetes manifest diff as a PR comment. |
| **Chainsaw E2E tests** | `components/kyverno/**`, `components/policies/**` | Runs integration tests in a Kind cluster. Path-triggered — runs on any PR changing these paths regardless of environment. Can also be run locally — set up a Kind cluster with `hack/chainsaw/chainsaw-prepare.sh`, then run `chainsaw test <path>`. |
| **Kyverno policy tests** | `components/kyverno/**`, `components/policies/**` | CLI-based Kyverno policy validation. |
| **Tekton-Kueue config tests** | `components/kueue/**` | Validates CEL expressions in Tekton-Kueue integration. |
| **Pipeline-service verify** | `components/pipeline-service/**` | Ensures kustomize output matches committed manifests. |

**E2E test caveats:**
- Chainsaw tests can be run locally — use `hack/chainsaw/chainsaw-prepare.sh` to set up a Kind cluster with Kyverno, then `chainsaw test <path>`.
- E2E tests frequently fail in CI due to intermittent infrastructure issues. If the PR looks correct and logs show no relevant errors, comment `/retest` to re-trigger.

## Production Ring Rollouts

Production changes must be split into 3 ring PRs covering subsets of clusters. Never apply a production change to all clusters in a single PR — unless it's a hotfix using the `skip-ring-deployment/hotfix` label.

- Each ring PR title includes the ring number:
  `KFLUXINFRA-1234: short description of the change (ring-1)`
- The **What** section lists the specific clusters included in this ring.
- Later ring PRs reference earlier ring PRs in **Validation** as evidence of success.
- This is a team convention — CI does not enforce ring splitting, but the ring enforcement check does prevent mixing staging and production changes.

## Production PR Requirements

- **Risk Assessment** section is mandatory (level, what could go wrong, rollback plan, blast radius).
- Follow the ring rollout pattern above.
- When updating component images, check if corresponding references exist in `hack/new-cluster/templates/` — if so, update them and include this in the PR's **What** section. New clusters are bootstrapped from these templates and won't get ArgoCD-synced versions.

## Promotion Flow

Changes flow: **development/staging** then **production**.

Validate changes in dev/staging before promoting to production. Production PRs should reference the staging PR or evidence in their Validation section.

## Interactive Sessions

In interactive sessions (human + agent), always confirm with the human before pushing and opening the PR. Show them the commit message, PR title, and PR body for approval first. Never push or create a PR without explicit approval.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Forgetting Risk Assessment on a production PR | Add the section — reviewers will block without it |
| Changing all production clusters in one PR | Split into 3 ring PRs |
| Not running `kustomize build` before pushing | Run it on all affected overlays, mention in Validation |
| Skipping `hack/new-cluster/templates/` sync | Update template image references when changing component images |
| Putting explanation in Validation instead of proof | Validation = evidence (build output, staging links). Why = explanation. |
| Branching from a stale main | Always fetch and reset from upstream before branching |
