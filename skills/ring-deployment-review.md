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

**Part-specific check scope**: Part 3 and Part 5 only edit `delete-applications.yaml` files. Part 6 renames `<COMPONENT>-rd` → `<COMPONENT>` and deletes the legacy component. For these parts, most component structure checks (1a-1e, 2-4, 6, 8, 10) do not apply—focus on the SOP steps (Check 0) and the specific files changed.

After completing Check 0, proceed with checks 1a-10 below.

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
| 1a | File naming | List every resource file. Each MUST be `{resource-name}-{type-abbrev}.yaml` using the abbreviation table above (Secret and SCC are extensions beyond Section 5.1). If the resource name already ends with the abbreviation, do not repeat it (e.g., `kubearchive-logging-scc` → `kubearchive-logging-scc.yaml`, not `kubearchive-logging-scc-scc.yaml`). For custom resources without a standard abbreviation, a short descriptive suffix is acceptable. FAIL any file that doesn't match these rules |
| 1b | One resource per file | Run `grep -c '^kind:'` on hand-written resource files listed in `resources:`. Skip files in `templates/`, `charts/`, `.chainsaw-test/`, `patches/` subdirectories, and upstream manifests/CRDs (Section 5.2). Count > 1 = FAIL. That file has multiple documents and MUST be split |
| 1c | Patches in patches/ | All patch files MUST be in a `patches/` subdirectory and end with `-patch.yaml`. FAIL any patch outside patches/ |
| 1d | Directory organization | If applicable — if any directory has more than ~6 resource files, it should be organized into subdirectories |
| 1e | Tier structure, rings, and clusters | Verify tier structure (Sections 3-4) and cluster assignments match `docs/ring-deployments/directory-layout.md` Section 1 "Current Ring/Cluster Layout". Check expected rings exist for the part: Part 1 requires ring-0 (development), Part 2 requires ring-1 (staging), Part 4 requires rings 2/3/4 (production). Verify each ring has correct cluster directories per Section 1 (e.g., ring-1 should have kflux-stg-p01, kflux-stg-p02; ring-2 should have stone-prod-p01, kflux-prd-rh03, etc.) |
| 2 | Base is truly common | For the first PR of a migration, first verify the PR doesn't modify `hack/test-base-common.sh` (`git diff main...HEAD -- hack/test-base-common.sh`). If modified, flag as manual verification. If not modified, run `hack/test-base-common.sh <COMPONENT>-rd`. FAIL if base misses common resources (some omissions may be intentional — version-pinned upstream resources). Do NOT skip this for first PRs |
| 3 | base-snapshot matches base | Run `diff -r components/<COMPONENT>-rd/base components/<COMPONENT>-rd/rings/ring-N/base/base-snapshot` for each ring. Any diff = FAIL. Do NOT flag as manual — this is an automated check |
| 4 | Ring render matches legacy | First verify the PR doesn't modify `hack/test-ring-render.sh` (`git diff main...HEAD -- hack/test-ring-render.sh`). If modified, flag as manual verification. If not modified, run `hack/test-ring-render.sh <COMPONENT>-rd all`. MUST pass. Render diffs are sometimes acceptable but MUST be documented with an explanation of WHY |
| 5 | AppSet configuration | Compare new AppSet against SOP template AND original. Verify template is the base with original's extras added (retry, ignoreDifferences, annotations, namespace, syncOptions) |
| 6 | Legacy dirs removed, legacy files kept | `ls components/<COMPONENT>-rd/` — should have `base/`, `rings/`, optionally `features/`, plus `OWNERS`, `README.md`, etc. No `development/`, `staging/`, `production/` |
| 7 | delete-legacy entry | Run `grep -r '<COMPONENT>' argo-cd-apps/base/member/` or `kustomize build --enable-helm argo-cd-apps/overlays/development \| yq 'select(.kind=="ApplicationSet") \| .metadata.name'`. If component AppSet found, verify `$patch: delete` entry exists in `argo-cd-apps/overlays/rd-dev/delete-legacy-konflux-member-appsets.yaml` |
| 8 | No trailing `/` on directory refs | `find components/<COMPONENT>-rd -name kustomization.yaml -exec grep -Hn '^ *- .*/$' {} +`. Style preference (kustomize accepts both with and without trailing slash). Flag if found |
| 9 | hack/ and workflow refs | Run `grep -r '<COMPONENT>' hack/ .github/workflows/`. If no output, PASS. If matches found, list them and check if they need updating. Do NOT flag as manual — this is an automated check |
| 10 | empty-base exists | Verify `rings/empty-base/empty-base/kustomization.yaml` exists with `resources: []` |

## Report Content

The report MUST include these sections in order:

1. **Summary** — component name, SOP part, PR number or branch name
2. **SOP Step Verification (Check 0)** — list each SOP step with PASS/FAIL/manual flag status
3. **Technical Checks (1a-10)** — per-check results with pass/fail/flag status and details for failures
4. **Automated verification output** — test-ring-render results, test-base-common results, diff results
5. **Action items** — what needs to be fixed before merge (blocking issues first)
6. **Manual verification needed** — flagged items for PR creator to confirm

## Common Mistakes

| Mistake | Impact | Fix |
|---------|--------|-----|
| Multi-document YAML files (multiple `kind:` in hand-written resources) | Violates Section 5.2; review will request changes | Split into individual files with `{resource-name}-{type-abbrev}.yaml` naming. Exception: templates/, charts/, .chainsaw-test/, patches/, and upstream manifests |
| base-snapshot diverges from base/ | Kargo promotion will overwrite; renders won't match | Always `diff -r` and update base-snapshot after any base/ change |
| Using SOP template values verbatim without checking original | Component may behave differently (e.g., different retry backoff) | Diff original AppSet, carry over retry/backoff/ignoreDifferences |
| Forgetting delete-legacy entry | Both old and new AppSets create Applications with same name template | Check if legacy exists in ../development graph |
| Keeping legacy env dirs in -rd | Confusing structure, potential path conflicts | Delete development/, staging/, production/ but keep OWNERS, README |
| Trailing `/` on directory references | Style inconsistency (kustomize accepts both) | Remove trailing slashes from `resources:` entries for consistency |
| Not running test-ring-render with `all` | May miss regressions in other environments | Always use `all` tier when both old and new directories coexist |
