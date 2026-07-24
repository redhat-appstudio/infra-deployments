---
name: ci-troubleshooting
description: >
  Use when a CI check fails on a PR in infra-deployments and you need to
  understand what failed, how to read the logs, and how to fix it.
---

# CI Troubleshooting

## Overview

How to investigate and fix CI failures on infra-deployments PRs.

## When to Use

- A CI check failed on your PR
- You need to understand what a CI comment or status means
- You want to re-trigger a flaky test

## Prerequisites

Verify `gh` CLI is installed and authenticated:

```bash
gh auth status
```

All CI investigation commands below depend on it.

## Reading CI Logs

### GitHub Actions checks

```bash
gh pr checks <PR-number> --repo redhat-appstudio/infra-deployments
```

To investigate a failed check:

```bash
gh run view <run-id> --repo redhat-appstudio/infra-deployments
gh run view <run-id> --repo redhat-appstudio/infra-deployments --log-failed
```

### Prow checks (E2E tests)

Prow check statuses include a `target_url` with the build ID and job name. To find them:

```bash
gh api repos/redhat-appstudio/infra-deployments/commits/<sha>/statuses \
  --jq '.[] | select(.state == "failure") | "\(.context) — \(.target_url)"'
```

If a Prow job failed, fetch the log directly:

```
https://prow.ci.openshift.org/log?container=test&id=<build-id>&job=<job-name>
```

## Re-triggering failed checks

Comment on the PR to retry flaky CI without write access:

- **`/rerun`** — re-runs failed or cancelled **GitHub Actions** checks (yamllint, chainsaw, kube-linter, render-diff, etc.). Available to repository members and collaborators. `/rerun` can appear anywhere in the comment (e.g. `please /rerun`).

- **`/retest`** — re-triggers **Prow / OpenShift CI** E2E checks (`ci/prow/appstudio-e2e-tests`, etc.). Handled by the OpenShift CI bot.

## Common Failures

### yamllint

YAML formatting errors (trailing whitespace, wrong indentation, missing newline at end of file). The CI logs usually show the exact file and line — fix those directly. If the logs aren't clear, run `yamllint .` locally.

### Render-diff

Not a pass/fail check — it posts a PR comment showing the rendered Kubernetes manifest diff. Review it to verify your kustomize changes produce the expected output.

If it shows a "merge conflicts" error, rebase your PR.

### Ring enforcement

Blocks PRs that modify both staging and production files in the same PR.

Fix: split into separate staging and production PRs. Hotfixes that must ship fast still need separate staging vs production PRs if both envs change — CI blocks mixing them.

### Chainsaw tests

Path-triggered on `components/kyverno/**` and `components/policies/**` changes. Often flaky due to infrastructure issues.

If logs show no relevant errors and the PR looks correct, comment `/rerun` on the PR. Maintainers with write access can also rerun manually with `gh run rerun <run-id> --repo redhat-appstudio/infra-deployments --failed`. If the logs don't help identify the issue, run locally with `hack/chainsaw/chainsaw-prepare.sh` to set up a Kind cluster, then `chainsaw test <path>`.

### kube-linter

Scans Kubernetes manifests for security and best practice violations. Check the logs for specific rule violations.

### Prow E2E tests (dev PRs only)

`ci/prow/appstudio-e2e-tests` and `ci/prow/appstudio-operator-overlay-e2e-tests` run on OpenShift CI. These only trigger on dev PRs, not staging or production.

If failed, fetch the log and check whether the failure is an intermittent infrastructure issue (cluster provisioning timeout, image pull failure, TLS handshake errors) or a real test failure. If intermittent, `/retest`.

