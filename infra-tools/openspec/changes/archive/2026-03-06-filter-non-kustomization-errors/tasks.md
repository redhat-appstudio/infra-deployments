## 1. Error Classification

- [x] 1.1 Add an `IsNotKustomizationError` helper function in `internal/renderdiff/` that takes an error string and returns true if it matches the krusty "missing kustomization file" pattern
- [x] 1.2 Add a `SkipOutput` boolean field to `ComponentDiff` in `internal/renderdiff/diff.go`
- [x] 1.3 In `RunProgressive` (`internal/renderdiff/engine.go`), when a build error is classified as non-kustomization, set `cd.SkipOutput = true` and do not append the component to `result.Diffs` — only log and continue

## 2. Output Filtering

- [x] 2.1 In `printComponentDiff` (`cmd/render-diff/output.go`), skip components where `SkipOutput` is true
- [x] 2.2 In `printSummary` (`cmd/render-diff/output.go`), skip components where `SkipOutput` is true and exclude them from the component count
- [x] 2.3 In `writeCISummary` (`cmd/render-diff/ci.go`), skip components where `SkipOutput` is true and exclude them from the header count
- [x] 2.4 In `buildCommentBody` (`cmd/render-diff/ci.go`), skip components where `SkipOutput` is true and exclude them from the table and total count

## 3. Tests

- [x] 3.1 Add unit tests for `IsNotKustomizationError` covering: the exact kustomize error message, a genuine build error, and edge cases (empty string, partial match)
- [x] 3.2 Update `TestEngine_BuildFailure_ReportsError` in `engine_test.go` to verify that non-kustomization errors set `SkipOutput = true` and are excluded from `DiffResult.Diffs`
- [x] 3.3 Add a test case verifying that genuine build errors still appear in `DiffResult.Diffs` with `SkipOutput = false`
