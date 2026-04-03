## 1. Go Code Changes

- [x] 1.1 In `postCIComment` (`cmd/render-diff/ci.go`), read `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` from environment and construct the run URL (or empty string if either is missing)
- [x] 1.2 Update `buildCommentBody` signature to accept a `runURL string` parameter
- [x] 1.3 In `buildCommentBody`, use `runURL` for the workflow summary link when non-empty, falling back to `../actions` when empty
- [x] 1.4 Update the `runOutputMode` call site that invokes `buildCommentBody` to pass through the run URL

## 2. Workflow Changes

- [x] 2.1 In `.github/workflows/pr-render-diff.yaml`, add `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` to the render-diff step's `env` block

## 3. Tests

- [x] 3.1 Add a test for `buildCommentBody` with a run URL provided — verify the direct link appears in the output
- [x] 3.2 Add a test for `buildCommentBody` with an empty run URL — verify the `../actions` fallback link appears
