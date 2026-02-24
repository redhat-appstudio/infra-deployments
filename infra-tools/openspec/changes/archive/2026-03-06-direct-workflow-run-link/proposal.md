## Why

The render-diff PR comment includes a link to the workflow summary and downloadable artifact, but it uses a relative `../actions` URL that lands on the full actions list. For repos like infra-deployments with 55K+ workflow runs, finding the relevant run is impractical. Users must manually search for the correct "PR Render Diff" run among thousands.

## What Changes

- The PR comment link will point directly to the specific workflow run (e.g., `https://github.com/owner/repo/actions/runs/12345`) instead of the generic actions page
- The workflow will pass `GITHUB_RUN_ID` and `GITHUB_SERVER_URL` as environment variables to the render-diff binary
- `buildCommentBody` will read these env vars and construct the direct URL
- When the env vars are absent (local runs, fallback), the link degrades gracefully to the current relative URL

## Capabilities

### New Capabilities

- `workflow-run-link`: Construct a direct link to the GitHub Actions workflow run in PR comment output.

### Modified Capabilities

_(none)_

## Impact

- **Code**: `infra-tools/cmd/render-diff/ci.go` (`buildCommentBody` and `postCIComment`)
- **Workflow**: `.github/workflows/pr-render-diff.yaml` (add two env vars)
- **User experience**: One-click access to the workflow summary and artifact download instead of manual search
- **Risk**: Minimal — falls back to current behavior if env vars are missing
