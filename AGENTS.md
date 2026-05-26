# infra-deployments

GitOps monorepo deploying 50+ Kubernetes components across multiple clusters via Kustomize and ArgoCD ApplicationSets.

## Quick Commands

| Action         | Command                                    |
|----------------|--------------------------------------------|
| Build overlay  | `kustomize build components/<name>/<env>/` |
| Lint YAML      | `yamllint .`                               |
| K8s lint       | `kube-linter lint <path>`                  |
| Chainsaw tests | `./hack/chainsaw/chainsaw-prepare.sh` and `chainsaw test <path to .chainsaw-test folder>` |
| infra-tools    | `cd infra-tools && make build test lint`   |

## Project Layout

- `components/<name>/{base,development,staging,production}/` — per-component Kustomize overlays; staging and production are often further split per-cluster
- `argo-cd-apps/overlays/` — maps to deployment targets (development, staging-downstream, production-downstream, etc.)
- `configs/` — cluster-level configurations (etcd-defrag, kubelet settings)
- `hack/` — deployment and utility scripts
- `infra-tools/` — Go CLI tools (env-detector, render-diff) with their own Makefile

## Key Conventions

- Prefer using scripts in `hack/` over manual steps when available
- Promotion order: development/staging → production; changes must be validated in dev/staging before promoting to production
- Production has per-cluster overlay directories; rollouts must be split into rings (subsets of clusters), not applied to all at once
- All changes via PR; CODEOWNERS approval required
- Production PRs must include `## Risk Assessment` (level, description, rollback plan) and `## Validation` (staging evidence if applicable)
- Commits - Jira ID at start (e.g., `KFLUXINFRA-1234 description`). Interactive sessions: Use the -s flag and `Assisted-by:` trailer. Agentic workflow: `Authored-by:` trailer. Include agent name and tool.

## Gotchas

- E2E tests are designed to validate in an isolated environment in GitHub Actions CI and should not be run locally
- E2E tests are conditional — they only run on dev/staging PRs when specific files change. Production PRs do not run E2E; rely on prior dev/staging validation
- E2E tests frequently fail due to intermittent infrastructure issues. If the PR looks correct and E2E logs show no relevant errors, comment `/retest` to re-trigger
- When updating component images, also update image references in `hack/new-cluster/templates/` as part of the production ring deployments — new clusters are bootstrapped from these and won't get ArgoCD-synced versions
