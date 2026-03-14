# Tasks: render-diff

This documents what was implemented.

## Group 1: Detection pipeline extension
- [x] Add `AffectedComponents()` method to `Detector`
- [x] Add `matchAffectedComponents()` helper function
- [x] Add tests for AffectedComponents (DepTreeMatch, NoMatch, MultipleEnvs, PrefixFallback)

## Group 2: Render diff engine
- [x] Create `internal/renderdiff/diff.go` — ComponentDiff, computeDiff, countStats
- [x] Create `internal/renderdiff/engine.go` — Engine, Run, RunProgressive, buildPair
- [x] Create `internal/renderdiff/normalize.go` — NormalizeYAML (sort by GVK+ns+name)
- [x] Create tests for engine (NormalDiff, NewComponent, RemovedComponent, BuildFailure, NoChange, Progressive)
- [ ] Create tests for normalize

## Group 3: CLI entry point
- [x] Create `cmd/render-diff/main.go` with flag parsing and setup
- [x] Implement `runLocal()` with progressive output and color support
- [x] Implement `runCISummary()` with collapsible markdown
- [x] Implement `runCIComment()` with env var config (GITHUB_TOKEN, GITHUB_REPOSITORY, PR_NUMBER)
- [x] Implement `runCIArtifactDir()` for raw .diff file export
- [x] Implement `openInDiffTool()` with directory-based folder comparison
- [x] Implement diff filenames using path slugs (components__foo__staging__staging.diff)
- [x] Add human-readable labels (ClusterDir priority) and collision handling (dedupeFileName)

## Group 4: GitHub integration
- [x] Create `internal/github/comments.go` — CommentClient, UpsertComment
- [x] Implement idempotent comment updates via HTML marker
- [x] Create tests (CreatesNew, UpdatesExisting, IgnoresUnrelated)

## Group 5: Build system
- [x] Add `render-diff` to Makefile build target
- [x] Add `git.MergeBase()` function

## Group 6: CI workflow
- [x] Create `.github/workflows/pr-render-diff.yaml`
- [x] Implement pull_request_target security model (build from base, analyze head)
- [x] Configure three output tiers (comment, summary, artifact)
- [x] Use env vars for CI config instead of CLI flags

## Group 7: Documentation
- [x] Create `infra-tools/README.md` with developer guide
- [x] Document CI output modes and env vars
