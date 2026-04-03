# Design: render-diff

## Design Decisions

### DD-1: Component-level diffing (Level B)

Diffs are computed at the component level (what gets deployed to clusters),
not at the ArgoCD overlay level. This shows developers the actual Kubernetes
manifests that will change, which is what matters for understanding impact.

### DD-2: Diff base strategy

Both local and CI use the merge-base with `main` (computed via
`git merge-base HEAD main`). This ensures the diff shows only the changes
introduced by the branch, not unrelated commits merged into main since the
branch was created.

Previously, CI passed `--base-ref=origin/$base_ref` which compared against
the tip of the target branch. This was incorrect when the target branch had
advanced: the diff would include changes already landed on main that weren't
on the PR branch, inflating the render delta.

### DD-3: Shared packages, separate binary

render-diff is a separate binary (`cmd/render-diff`) that shares internal
packages with env-detector (`detector`, `appset`, `git`, `github`, etc.).
This avoids code duplication while keeping the tools independently
deployable.

### DD-4: CI config via environment variables, not CLI flags

**Rationale**: The chief architect requested that GitHub API posting logic
live in the binary itself (not delegated to `gh` CLI in the workflow). However,
exposing `--github-token`, `--pr-number`, and `--github-repo` as CLI flags
was counterproductive:
- Developers should not post PR comments from local runs
- Seeing these flags in `--help` is confusing

**Decision**: CI-specific configuration is read from environment variables
(`GITHUB_TOKEN`, `GITHUB_REPOSITORY`, `PR_NUMBER`). The workflow sets these
via `env:` blocks. If any are missing, `ci-comment` mode falls back to
printing markdown to stdout. This keeps the `--help` output clean for local
users while keeping the posting logic in Go as requested.

### DD-5: YAML normalization for cleaner diffs

**Status: Implemented** in `internal/renderdiff/normalize.go`. Kustomize
output is a multi-document YAML stream where resource ordering can shift
between builds. This causes diff tools to show large block-level changes
when only a single field changed.

**Decision**: After building each side, the YAML is normalized by sorting
documents by `apiVersion/kind/namespace/name`. This is done in
`computeDiff()` so all output modes benefit. Normalization is best-effort
— if parsing fails, the original YAML is used unchanged.

### DD-6: Directory-based --open mode

**Rationale**: The initial implementation opened each diff sequentially
in the diff tool (one file pair at a time), which was impractical with many
components.

**Decision**: `--open` writes all base YAML files to a `base/` temp directory
and all head YAML files to a `head/` directory, then invokes the diff tool
once. This enables folder comparison in tools like Meld, showing all changed
components in a single unified view. For `$DIFFTOOL`, it passes the two
directories directly. For `git difftool`, it uses `--dir-diff`.

### DD-7: Diff filenames

**Status: Implemented** in `cmd/render-diff/files.go`. Diff files use
human-readable labels with `ClusterDir` priority when available
(e.g., `stone-prod-p02--production.diff`), falling back to path-based
slugs (e.g., `components__foo__staging__staging.diff`). Collision handling
appends a counter suffix (`-2`, `-3`, etc.) via `dedupeFileName()`.

### DD-8: CI security model

The workflow uses `pull_request_target` to get write access for PR comments.
The binary is built from the trusted base branch before switching to the PR
head for analysis. This prevents fork PRs from injecting malicious code that
runs with elevated `GITHUB_TOKEN` permissions.

### DD-9: No wrapper script

**Rationale**: An initial `hack/render-diff.sh` wrapper was explored to
auto-build the binary via Go's build cache. This was removed in favor of
running `make build` directly from `infra-tools/`, which is simpler and
consistent with the env-detector workflow.

### DD-10: No short CLI flags

**Rationale**: Adding short flags (`-o` for `--open`, etc.) would require
replacing Go's standard `flag` package with `pflag` or `cobra`. The tool has
few flags, the defaults handle the common case (no flags needed), and the
typing overhead is negligible. Not worth the added dependency.

## Architecture

```
cmd/render-diff/main.go
  ├── flag parsing + setup
  ├── detector.AffectedComponents()    ← reused from env-detector
  ├── renderdiff.Engine.Run/RunProgressive()
  └── output formatters (local, ci-summary, ci-comment, ci-artifact-dir)

internal/renderdiff/
  ├── diff.go          ComponentDiff, computeDiff (go-difflib)
  ├── engine.go        Engine, Run, RunProgressive, buildPair
  ├── normalize.go     NormalizeYAML (sort by GVK+ns+name)
  └── *_test.go

internal/github/
  └── comments.go      CommentClient, UpsertComment (idempotent via HTML marker)
```

## Open Decisions

### OD-1: Handling non-kustomize component paths

**Status**: Pending architect decision

**Context**: `AffectedComponents()` returns all component paths from
ApplicationSets, including directories that don't contain a
`kustomization.yaml`. These are plain YAML directories that ArgoCD deploys
as raw manifests. Currently, `buildPair` attempts `kustomize build` on them,
which fails, and the error is captured and shown in the output.

**Question**: Should we avoid sending these paths to the engine when we know
the build will fail? What's the right behaviour?

**Options**:

| Option | Approach | Pros | Cons |
|--------|----------|------|------|
| **A** (current) | Try to build, capture and display the error | Transparent — reviewer sees all paths attempted | Noisy — "build error" for paths that were never going to work |
| **B** | Skip non-kustomize paths in the engine (check for `kustomization.yaml` before building, log at INFO level) | Clean output, no spurious errors | Changes to plain YAML directories are invisible in render-diff |
| **C** | Handle non-kustomize paths differently — concatenate raw YAML files instead of running `kustomize build`, then diff those | Most correct — these dirs *are* deployed, so their changes *should* appear | More implementation work; requires new `RepoBuilder` method (e.g., `ReadRawYAML`) |
| **D** | Filter upstream in `AffectedComponents()` — split return into kustomize vs plain YAML paths, only send kustomize ones to engine | Separation of concerns | Same visibility loss as B, pushes render-diff concerns into the detector |

**Recommendation**: Option C is the most correct long-term. Option B with
an INFO log is a pragmatic interim step that addresses the reviewer's concern
without significant effort.

## Risks

- **Kustomize build failures**: Build errors are non-fatal — the component is
  skipped with a warning, and the error is reported in CI output.
- **Large diffs**: CI summary truncates diffs over 50KB with a pointer to the
  downloadable artifact.
- **YAML normalization edge cases** (when implemented): Unparseable documents
  should be left in their original order. The sort should be stable so
  documents without metadata sort consistently.
