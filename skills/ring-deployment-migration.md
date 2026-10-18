---
name: ring-deployment-migration
description: >
  Use when migrating a component to ring deployment directory structure.
  Covers planning and autonomous execution of each SOP part.
  Supplements the SOP with file restructuring rules and validation.
---

# Ring Deployment Migration

How to migrate a component to ring deployment structure.

For the full PR lifecycle (What / Why / Validation), see `skills/pr-workflow.md`.
For reviewing a migration PR or branch, see `skills/ring-deployment-review.md`.

## When to Use

- Migrating a component from legacy `development/staging/production/` structure to ring deployments

## Key References

| Document | Path | Use For |
|----------|------|---------|
| SOP | [directory-structure-migration.md](https://gitlab.cee.redhat.com/konflux/docs/sop/-/blob/main/infra/ring-deployments/directory-structure-migration.md) — see [Accessing the SOP](#accessing-the-sop) below | Step-by-step process (Parts 1–6) |
| Directory layout | `docs/ring-deployments/directory-layout.md` | Canonical structure, tier definitions, file conventions |
| Ring render check | `hack/test-ring-render.sh` | Verify rendered output matches legacy |
| Base common check | `hack/test-base-common.sh` | Identify resources common to all legacy envs |

### Accessing the SOP

The SOP lives in a private GitLab repo ([konflux/docs/sop](https://gitlab.cee.redhat.com/konflux/docs/sop)). Ask the user how they prefer to access it:

1. **Local clone** — ask the user for the path to their local clone of the repo (e.g., `~/konflux/sop`). The SOP file is at `<repo-path>/infra/ring-deployments/directory-structure-migration.md`. If not cloned yet:
   ```bash
   git clone git@gitlab.cee.redhat.com:konflux/docs/sop.git
   ```

2. **glab CLI** — read via:
   ```bash
   glab api projects/konflux%2Fdocs%2Fsop/repository/files/infra%2Fring-deployments%2Fdirectory-structure-migration.md/raw?ref=main --hostname gitlab.cee.redhat.com
   ```
   If `glab` is not installed or not authenticated to `gitlab.cee.redhat.com`:
   ```bash
   # Install (Fedora/RHEL)
   sudo dnf install glab

   # Authenticate
   glab auth login --hostname gitlab.cee.redhat.com
   ```

---

## Migration Mode

### Which SOP Part?

If the user does not specify which part, ask. The parts are:

| Part | What | When |
|------|------|------|
| Part 1 | Development migration (ring-0) | First step; skip if component has no dev env |
| Part 2 | Staging migration (ring-1) | After Part 1 merges (or first step if no dev env) |
| Part 3 | Old staging deletion | After Part 2 merges |
| Part 4 | Production migration (rings 2–4) | After Part 3 merges |
| Part 5 | Old production deletion | After Part 4 merges |
| Part 6 | Old component removal | After Part 5 merges |

### Execution

Follow the SOP steps **sequentially** — do not skip or reorder steps within a part. Also follow `docs/ring-deployments/directory-layout.md` for canonical structure (Sections 3-4) and cluster/ring assignments.

**For Parts 1 or 2 (first PR for the component):** after SOP step 2 (create `-rd` copy) and step 3 (create `base/`), run `hack/test-base-common.sh <COMPONENT>-rd` to verify the base captures what is truly common across all legacy environments. Review findings — Helm-generated resources (Services, ServiceAccounts, ClusterRoles) stay in ring, not base; only manually authored resources that are identical across all envs belong in base. Clean up any prerequisites created solely for this check that don't belong in the current SOP step. Then continue with the remaining SOP steps.

**Every time you copy or create a YAML file during ANY SOP step, you MUST immediately:**

1. Run `grep -c '^---$'` on the file. If count > 0, the file has multiple resources — split it into separate files, one resource per file. Do NOT copy multi-document files as-is.

2. Rename each file to `{resource-name}-{type-abbreviation}.yaml` where `resource-name` is the `metadata.name`. Use these abbreviations:

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

   For custom resources without a standard abbreviation, use a short descriptive suffix.

   Examples:
   - `scc.yaml` with SCC named `kubearchive-logging-scc` → `kubearchive-logging-scc.yaml`
   - `sa.yaml` with 4 ServiceAccounts → split into `loki-sa.yaml`, `loki-canary-sa.yaml`, `metrics-reader-sa.yaml`, `loki-gateway-sa.yaml`
   - `rbac.yaml` with ClusterRole + ClusterRoleBinding + RoleBinding → split into `{name}-cr.yaml`, `{name}-crb.yaml`, `{name}-rb.yaml`

3. If the file is a patch, put it in a `patches/` subdirectory with `-patch.yaml` suffix.

4. Update `kustomization.yaml` to reference the new filenames.

When applicable — organize related files into subdirectories with their own `kustomization.yaml` (e.g., `loki/`, `rbac/`, `grafana/`, `vector/`, `monitoring/`). Do this when a directory has more than ~6 resource files, or when matching an existing subdirectory structure from `base/` so that ring directories extend the same folders (e.g., `base/konflux-rbac/` → `rings/ring-1/base/konflux-rbac/`).

**After completing the last SOP step, run these validation commands. Fix any failures before presenting results:**

1. Verify base-snapshot matches base:
   `diff -r components/<COMPONENT>-rd/base components/<COMPONENT>-rd/rings/ring-N/base/base-snapshot`

2. Verify renders match legacy (use `all` when applicable — render diffs are sometimes acceptable but document WHY):
   `hack/test-ring-render.sh <COMPONENT>-rd all`

3. Check common base resources (first PR only):
   `hack/test-base-common.sh <COMPONENT>-rd`

4. Build each kustomization.yaml independently:
   ```bash
   find components/<COMPONENT>-rd -name kustomization.yaml -exec dirname {} \; | \
     while read d; do echo "=== $d ==="; kustomize build --enable-helm "$d" > /dev/null && echo OK || echo FAIL; done
   ```

5. Build the appset overlay:
   `kustomize build --enable-helm argo-cd-apps/overlays/rd-dev/<COMPONENT>`

6. Search for path references that need updating:
   `grep -r "<COMPONENT>" hack/ .github/workflows/`

7. Run kube-linter on each kustomization build output (CI builds each independently):
   ```bash
   find components/<COMPONENT>-rd -name kustomization.yaml -exec dirname {} \; | \
     while read d; do echo "=== $d ==="; kustomize build --enable-helm "$d" 2>/dev/null | kube-linter lint - && echo OK || echo FAIL; done
   ```

### AppSet Configuration

Start from the **SOP template** for the ApplicationSet. Then compare against the **original** AppSet (usually in `argo-cd-apps/base/member/infra-deployments/<COMPONENT>/` or `argo-cd-apps/base/all-clusters/`) and add any settings the original has that the template lacks:

- `retry` values (limit, backoff duration/factor/maxDuration)
- `ignoreDifferences` entries
- `annotations` (sync-wave, refresh, compare-options)
- `namespace` in destination
- `syncOptions`

Do not blindly copy the original — use the SOP template as the base and supplement with the original's extras.

### PR Conventions

Follow `skills/pr-workflow.md` for the full PR lifecycle. Migration-specific additions:

- **PR title** — use the exact title format from the SOP's commit step for each part (e.g., `(onboarding) Create new development <COMPONENT_NAME> AppSet in rd-dev overlay` for Part 1)
- **Validation section** — include proof of verification steps performed:
  - `hack/test-ring-render.sh` results (pass/fail, and explanation if diffs exist)
  - `hack/test-base-common.sh` results (for first PR)
  - `kustomize build --enable-helm` passes for all affected overlays
  - `diff -r` base vs base-snapshot

For a thorough review, use `skills/ring-deployment-review.md`.
