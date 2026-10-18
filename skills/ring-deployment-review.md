---
name: ring-deployment-review
description: >
  Use when reviewing a ring deployment migration PR or local branch.
  Runs a structured checklist against the SOP and directory layout,
  produces a local or HTML report with pass/fail results and action items.
---

# Ring Deployment Review

How to review a ring deployment migration PR or branch.

For performing the migration itself, see `skills/ring-deployment-migration.md`.
For the full PR lifecycle, see `skills/pr-workflow.md`.

## When to Use

- Reviewing a migration PR (titled `(onboarding)` or restructuring a component for rings)
- Reviewing a local branch before opening a migration PR

## Key References

| Document | Path | Use For |
|----------|------|---------|
| SOP | [directory-structure-migration.md](https://gitlab.cee.redhat.com/konflux/docs/sop/-/blob/main/infra/ring-deployments/directory-structure-migration.md) — see `skills/ring-deployment-migration.md` for access options | Step-by-step process (Parts 1–6) |
| Directory layout | `docs/ring-deployments/directory-layout.md` | Canonical structure, tier definitions, file conventions |

## Output Format

**Before running any checks, you MUST ask the user which output format they prefer.** Do NOT skip this question. Do NOT default to one format.

1. **Local report** — printed to terminal
2. **HTML report** — generate a standalone dark-themed HTML report file with sections, status badges, and per-check results. Ask the user where to save it.

## Review Checklist

**MANDATORY: Start your report with Check 0.** Walk through EVERY SOP step for the relevant part (Part 1, Part 2, etc.) and state whether each step was completed correctly. List each SOP step number and what it does. For steps you can verify from code, report PASS/FAIL. For manual steps that require cluster access or post-merge actions (ArgoCD sync, testing in cluster), flag as "Ask the PR creator to confirm step N was completed."

**Context matters**: If reviewing a PR (not a local branch), steps about creating the branch and opening the PR automatically PASS—the PR exists. If reviewing a local branch before PR creation, those steps are N/A.

After completing Check 0, proceed with checks 1a-11 below.

Use these abbreviations when checking file naming (check 1a):

| Resource Kind | Abbreviation |
|--------------|-------------|
| Namespace | `ns` |
| ServiceAccount | `sa` |
| Role | `role` |
| ClusterRole | `cr` |
| RoleBinding | `rb` |
| ClusterRoleBinding | `crb` |
| ConfigMap | `cm` |
| Secret | `secret` |
| Service | `svc` |
| Deployment | `deploy` |
| ExternalSecret | `es` |
| ServiceMonitor | `sm` |
| PrometheusRule | `pr` |
| SecurityContextConstraints | `scc` |

| # | Check | How to Verify |
|---|-------|---------------|
| 0 | All SOP steps for the relevant part are satisfied | Walk through each SOP step for the part being reviewed. For manual steps (cluster access, ArgoCD sync), flag as "Ask the PR creator to confirm step N was completed" |
| 1a | File naming | List every resource file. Each MUST be `{resource-name}-{type-abbrev}.yaml` using the abbreviation table above. FAIL any file that doesn't match |
| 1b | One resource per file | Run `grep -c '^---$'` on every YAML file. Count > 0 = FAIL. That file has multiple documents and MUST be split |
| 1c | Patches in patches/ | All patch files MUST be in a `patches/` subdirectory and end with `-patch.yaml`. FAIL any patch outside patches/ |
| 1d | Directory organization | If applicable — if any directory has more than ~6 resource files, it should be organized into subdirectories |
| 1e | Tier structure and clusters | Verify tier structure and cluster assignments match `docs/ring-deployments/directory-layout.md` Sections 3-4 |
| 2 | Base is truly common | For the first PR of a migration, run `hack/test-base-common.sh <COMPONENT>-rd`. FAIL if base misses common resources (some omissions may be intentional — version-pinned upstream resources). Do NOT skip this for first PRs. Do NOT flag as manual — this is an automated check |
| 3 | base-snapshot matches base | Run `diff -r components/<COMPONENT>-rd/base components/<COMPONENT>-rd/rings/ring-N/base/base-snapshot` for each ring. Any diff = FAIL. Do NOT flag as manual — this is an automated check |
| 4 | Ring render matches legacy | Run `hack/test-ring-render.sh <COMPONENT>-rd all`. MUST pass. Render diffs are sometimes acceptable but MUST be documented with an explanation of WHY. Do NOT flag as manual — this is an automated check |
| 5 | AppSet configuration | Compare new AppSet against SOP template AND original. Verify template is the base with original's extras added (retry, ignoreDifferences, annotations, namespace, syncOptions) |
| 6 | Legacy dirs removed, legacy files kept | `ls components/<COMPONENT>-rd/` — should have `base/`, `rings/`, optionally `features/`, plus `OWNERS`, `README.md`, etc. No `development/`, `staging/`, `production/` |
| 7 | kube-linter annotations | If original had `commonAnnotations` with `ignore-check.kube-linter.io/`, verify they appear on every subdirectory `kustomization.yaml` that CI will lint independently (any kustomization.yaml with `generators:`) |
| 8 | delete-legacy entry | Run `grep -r '<COMPONENT>' argo-cd-apps/overlays/development/kustomization.yaml`. If found, check `delete-applications.yaml`. If NOT already deleted there, FAIL — must add `$patch: delete` in `delete-legacy-konflux-member-appsets.yaml` |
| 9 | No trailing `/` on directory refs | `grep '/$' components/<COMPONENT>-rd/**/kustomization.yaml` |
| 10 | hack/ and workflow refs | Run `grep -r '<COMPONENT>' hack/ .github/workflows/`. If no output, PASS. If matches found, list them and check if they need updating. Do NOT flag as manual — this is an automated check |
| 11 | empty-base exists | Verify `rings/empty-base/empty-base/kustomization.yaml` exists with `resources: []` |

## Report Content

The report MUST include these sections in order:

1. **Summary** — component name, SOP part, PR number or branch name
2. **SOP Step Verification (Check 0)** — list each SOP step with PASS/FAIL/manual flag status
3. **Technical Checks (1a-11)** — per-check results with pass/fail/flag status and details for failures
4. **Automated verification output** — test-ring-render results, test-base-common results, diff results
5. **Action items** — what needs to be fixed before merge (blocking issues first)
6. **Manual verification needed** — flagged items for PR creator to confirm

## Common Mistakes

| Mistake | Impact | Fix |
|---------|--------|-----|
| Multi-document YAML files (files with `---`) | Violates Section 5.2; review will request changes | Split into individual files with `{resource-name}-{type-abbrev}.yaml` naming |
| base-snapshot diverges from base/ | Kargo promotion will overwrite; renders won't match | Always `diff -r` and update base-snapshot after any base/ change |
| Missing kube-linter annotations in sub-kustomizations | CI kube-linter check fails | Add annotations to each subdirectory kustomization.yaml that has generators |
| Using SOP template values verbatim without checking original | Component may behave differently (e.g., different retry backoff) | Diff original AppSet, carry over retry/backoff/ignoreDifferences |
| Forgetting delete-legacy entry | Both old and new AppSets create Applications with same name template | Check if legacy exists in ../development graph |
| Keeping legacy env dirs in -rd | Confusing structure, potential path conflicts | Delete development/, staging/, production/ but keep OWNERS, README |
| Trailing `/` on directory references | Can cause kustomize build issues | Remove trailing slashes from `resources:` entries |
| Not running test-ring-render with `all` | May miss regressions in other environments | Always use `all` tier when both old and new directories coexist |
