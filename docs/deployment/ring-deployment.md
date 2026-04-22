---
title: Ring Deployment Policy
---

## Overview

This repository enforces a ring deployment model where **staging is ring 0**.
Changes must be validated in staging before they can be promoted to production.
To enforce this, a CI check prevents PRs from modifying both staging and
production overlays at the same time.

## How it works

The `enforce-ring-deployment` GitHub Actions workflow runs on every PR that
touches files under `components/`, `argo-cd-apps/`, or `configs/`. It uses
the `env-detector` tool to:

1. Determine which environments (staging, production) are affected by the PR.
2. Classify changed files as belonging directly to a staging overlay, a
   production overlay, or neither (shared base files).
3. **Block** the PR if files under both staging and production directories are
   directly modified.
4. **Warn** (but allow) the PR if only shared base files are changed and both
   environments are indirectly affected through kustomize dependency trees.

### Directory classification

A file is classified as a direct staging or production change based on whether
its path passes through one of these directory names:

| Directory segment             | Environment |
|-------------------------------|-------------|
| `staging`                     | staging     |
| `staging-downstream`          | staging     |
| `konflux-public-staging`      | staging     |
| `production`                  | production  |
| `production-downstream`       | production  |
| `konflux-public-production`   | production  |

This applies under `components/` (including nested components like
`monitoring/grafana/staging/`), any subtree of `argo-cd-apps/` (e.g.
`overlays/`, `app-of-app-sets/`, `k-components/`), and `configs/`.

Files under `base/`, `development/`, or other directories that are not
environment-specific are not classified as direct staging or production changes.

## Workflow for making changes

### Staging-only changes (normal path)

Modify files under `components/<service>/staging/` (and/or related staging
overlay directories). The CI check passes and the PR can be merged after
review.

### Production promotion

After changes are validated in staging, create a **separate PR** that modifies
the corresponding `production/` overlay. The CI check passes because only
production files are touched.

### Shared base changes

If you modify files under `components/<service>/base/` that are referenced by
both staging and production overlays, the CI check will pass with a **warning**
explaining that both environments are affected. The preferred approach is to rework the change into a staging-only one and once that's verified, opening a second PR that reverts the staging-only change and makes the change in the shared base files.

### What to do if the check fails

If the CI check blocks your PR, split it into two PRs:

1. **First PR**: Contains only the staging overlay changes. Merge and validate.
2. **Second PR**: Contains only the production overlay changes. Create this
   after staging validation confirms the changes work correctly.
