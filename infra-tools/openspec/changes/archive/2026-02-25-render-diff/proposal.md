# Proposal: render-diff

## Why

Developers changing kustomize components in infra-deployments have no easy way
to see what their changes will actually produce in each environment. They must
mentally trace kustomize overlays, bases, and patches to predict the rendered
output. This leads to unexpected changes reaching staging or production.

## What Changes

A new CLI tool (`render-diff`) that builds kustomize overlays on both the
current branch and the merge-base, then computes unified diffs for every
affected component. It reuses the env-detector's component detection pipeline
and adds a parallel render engine on top.

## Capabilities

1. **render-diff-cli** — Local CLI with progressive colored output, directory
   diff tool integration, and per-component `.diff` file export.

2. **render-diff-ci** — CI integration via GitHub Actions that posts a PR
   comment summary, generates a Job Summary with collapsible diffs, and
   uploads raw `.diff` files as artifacts.

## Impact

- Developers can see the exact render delta before merging
- CI provides automatic visibility on every PR
- Shared packages with env-detector (detector, appset, git, github)
- New packages: `internal/renderdiff` (engine, diff, normalization)
