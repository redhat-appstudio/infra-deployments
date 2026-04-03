# Spec: render-diff CI

## Overview

GitHub Actions integration that automatically shows kustomize render deltas
on every PR via three output tiers: PR comment, Job Summary, and
downloadable artifact.

## Requirements

### REQ-1: Three-tier CI output

The workflow produces three outputs for each PR:

1. **PR comment** (`ci-comment`) — summary table with per-component
   added/removed counts. Idempotent via HTML marker
   (`<!-- render-diff-comment -->`).
2. **Job Summary** (`ci-summary`) — collapsible `<details>` blocks with
   full diffs per component. Truncated at 50KB with a pointer to the
   artifact.
3. **Artifact** (`ci-artifact-dir`) — raw `.diff` files uploaded via
   `actions/upload-artifact`, always available even if other steps fail.

### REQ-2: CI config via environment variables

CI-specific configuration is read from environment variables, not CLI flags:

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | API token for authentication |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format |
| `PR_NUMBER` | Pull request number to comment on |

**Rationale**: The GitHub API posting logic lives in the Go binary (per chief
architect request), but these options should not appear in `--help` since
they are not intended for local use. Environment variables keep the CLI clean
while preserving the Go-native posting. If any variable is missing,
`ci-comment` falls back to printing markdown to stdout.

### REQ-3: Idempotent PR comments

The `CommentClient.UpsertComment()` method searches for an existing comment
containing the `<!-- render-diff-comment -->` HTML marker. If found, it
updates that comment; otherwise it creates a new one. This prevents duplicate
comments on force-pushes.

### REQ-4: Fork PR security

The workflow uses `pull_request_target` to get write access for PR comments.
The binary is built from the trusted base branch **before** switching to the
PR head. This prevents fork PRs from injecting code that runs with the
elevated `GITHUB_TOKEN`.

Steps:
1. Checkout base branch
2. Build binary from base (trusted code)
3. Fetch and checkout PR head
4. Run binary against PR head code

### REQ-5: Non-blocking failures

All render-diff steps use `continue-on-error: true`. The artifact upload
uses `if: always()`. A kustomize build failure for one component does not
block other components or the workflow.

### REQ-6: Debug logging

All CI steps write debug logs to `render-diff-debug.log` via `--log-file`.
This file is uploaded as part of the artifact for troubleshooting.

## Workflow File

`.github/workflows/pr-render-diff.yaml`

Triggers on `pull_request_target` events: `opened`, `synchronize`, `reopened`.

Permissions: `pull-requests: write`, `contents: read`.
