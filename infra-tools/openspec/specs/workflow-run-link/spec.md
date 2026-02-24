## ADDED Requirements

### Requirement: PR comment links directly to the workflow run
The PR comment SHALL include a direct URL to the specific GitHub Actions workflow run instead of a generic link to the actions list.

#### Scenario: CI environment with run ID available
- **WHEN** render-diff runs in `ci-comment` mode and `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` environment variables are set
- **THEN** the PR comment SHALL contain a link in the format `$GITHUB_SERVER_URL/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID`

#### Scenario: CI environment without run ID
- **WHEN** render-diff runs in `ci-comment` mode and `GITHUB_SERVER_URL` or `GITHUB_RUN_ID` is not set
- **THEN** the PR comment SHALL fall back to the relative `../actions` link

#### Scenario: No diffs detected
- **WHEN** render-diff runs in `ci-comment` mode and there are no render differences
- **THEN** no workflow link SHALL be shown (current behavior — comment body is "No render differences detected.")

### Requirement: Workflow passes run context to render-diff
The PR Render Diff workflow SHALL pass `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` as environment variables to the render-diff binary.

#### Scenario: Workflow environment variables
- **WHEN** the `pr-render-diff` workflow runs the render-diff step
- **THEN** `GITHUB_SERVER_URL` and `GITHUB_RUN_ID` SHALL be set in the step's `env` block alongside the existing `GITHUB_TOKEN`, `GITHUB_REPOSITORY`, and `PR_NUMBER` variables
