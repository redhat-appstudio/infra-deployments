## Context

The render-diff PR comment currently links to the actions page using a relative URL (`../actions`). GitHub Actions provides `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` environment variables in every workflow run, which together with `GITHUB_REPOSITORY` (already used) can construct a direct link: `$SERVER_URL/$REPOSITORY/actions/runs/$RUN_ID`. This pattern is already used in the repo's `gemini-dispatch.yml` workflow.

The render-diff binary follows DD-4 (CI config via environment variables, not CLI flags), so the new values will be read from env vars.

## Goals / Non-Goals

**Goals:**
- Replace the generic `../actions` link with a direct workflow run URL in the PR comment
- Maintain graceful fallback when env vars are absent (local runs)

**Non-Goals:**
- Adding CLI flags for the run URL (per DD-4)
- Changing the CI summary output (it doesn't contain links)

## Decisions

### DD-1: Read GITHUB_RUN_ID and GITHUB_SERVER_URL from environment

**Decision**: Read `GITHUB_RUN_ID` and `GITHUB_SERVER_URL` in `postCIComment` (where the other CI env vars are already read) and pass the constructed URL to `buildCommentBody`.

**Alternatives considered**:
- **Construct the URL in the workflow YAML and pass as a single env var**: Simpler for the Go code, but breaks the pattern where the binary reads raw GitHub context vars and composes behavior itself. Also makes the workflow responsible for URL formatting.
- **Add a `--run-url` CLI flag**: Violates DD-4 (CI config via env vars). Would clutter `--help` for local users.

**Rationale**: Consistent with the existing pattern — `GITHUB_TOKEN`, `GITHUB_REPOSITORY`, and `PR_NUMBER` are already read as env vars in `postCIComment`. Adding two more follows the same pattern.

### DD-2: Fallback to relative link

**Decision**: When `GITHUB_RUN_ID` or `GITHUB_SERVER_URL` is missing, fall back to the current `../actions` relative link. This keeps local/test usage working without requiring these vars.

**Rationale**: The env vars are always present in GitHub Actions, but may be absent in local testing or alternative CI systems. The relative link is a reasonable degradation.

## Risks / Trade-offs

- **[None significant]** — `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` are standard GitHub Actions variables, always available in workflow runs. The fallback ensures no breakage if absent.
