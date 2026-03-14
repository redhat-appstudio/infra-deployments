## Why

When `render-diff` processes affected components, some directories are plain YAML directories (deployed by ArgoCD as raw manifests) that don't contain a `kustomization.yaml`. Running `kustomize build` on these directories fails with "not a kustomization directory" errors. These errors are currently displayed prominently in both local output (bold red BUILD ERROR) and CI output (PR comments, job summaries), creating noise that confuses reviewers. Users have reported this is unhelpful since those folders were never meant to be built with kustomize.

## What Changes

- Non-kustomization build errors will be filtered out of user-facing output (local stdout, GitHub PR comments, and job summaries)
- Non-kustomization build errors will continue to be logged to the log file for diagnostic completeness
- Other genuine build errors (e.g. malformed kustomization, missing resources) will continue to be shown in output as before
- The engine will distinguish between "not a kustomization directory" errors and other kustomize build failures

## Capabilities

### New Capabilities

- `kustomization-error-classification`: Classify kustomize build errors to distinguish non-kustomization directory errors from genuine build failures, and filter non-kustomization errors from user-facing output while preserving them in logs.

### Modified Capabilities

_(none — no existing spec-level requirements are changing)_

## Impact

- **Code**: `infra-tools/internal/renderdiff/engine.go` (error classification in `buildPair`/`RunProgressive`), `infra-tools/cmd/render-diff/output.go` and `ci.go` (output filtering)
- **User experience**: Cleaner output — reviewers will no longer see irrelevant build errors for plain YAML directories
- **Observability**: No loss of information — errors remain in debug logs via `--log-file`
- **Risk**: Low — this is an output filtering change; the build and diff logic remain unchanged
