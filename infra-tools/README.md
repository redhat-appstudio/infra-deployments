# infra-tools

Go-based tooling for the infra-deployments repository. These tools analyse
the ArgoCD kustomize structure to detect which environments, clusters, and
components are affected by a set of file changes.

## Tools

### env-detector

Detects which environments and clusters a PR affects by building ArgoCD
ApplicationSet overlays, resolving kustomize dependency trees, and matching
changed files. Used in CI to auto-label PRs.

```bash
# Dry-run (prints affected environments/clusters without calling GitHub)
go run ./cmd/env-detector --repo-root /path/to/infra-deployments --dry-run

# Full run (labels a PR)
go run ./cmd/env-detector \
  --repo-root /path/to/infra-deployments \
  --pr-number 123 \
  --github-token "$GITHUB_TOKEN" \
  --repo owner/repo
```

Key flags:
- `--base-ref` — git ref to compare against (default: `main`)
- `--overlays-dir` — path to ArgoCD overlays (default: `argo-cd-apps/overlays`)
- `--cluster-labels` — include `cluster/<name>` labels
- `--dry-run` — print results without calling GitHub
- `--log-file` — write debug logs to a file

### overlay-app-collision-checker

Detects ArgoCD ApplicationSets that would template colliding generated
Application names when their overlays are deployed together to the same
cluster / ArgoCD control-plane namespace (e.g. during e2e bootstrap, or when
one overlay imports another's ApplicationSets directly). Fails loudly at
PR/CI time instead of letting the collision surface later as a flaky ArgoCD
ownership conflict.

```bash
# Build the binary
cd infra-tools
make build

# Check all known co-deployed overlay groups
./bin/overlay-app-collision-checker

# Explicit repo root (default: auto-detect via git)
./bin/overlay-app-collision-checker --repo-root /path/to/infra-deployments
```

Key flags:
- `--repo-root` — path to the repository root (default: auto-detect via `git`)

The overlay groups checked are hardcoded in `internal/collision.DefaultGroups`;
add a new entry there whenever a new pair of overlays gets co-deployed to the
same cluster.

### render-diff

Computes and displays the kustomize render delta for components affected by
the current branch's changes. Shows what will actually change in each
environment when your PR merges.

```bash
# Build the binary
cd infra-tools
make build

# Diff against merge-base with main
./bin/render-diff

# Force colored output
./bin/render-diff --color always

# Write .diff files to a directory
./bin/render-diff --output-dir ./diffs

# Open all diffs in a visual diff tool (folder comparison)
./bin/render-diff --open

# Use a specific diff tool
DIFFTOOL=meld ./bin/render-diff --open

# Explicit base ref
./bin/render-diff --base-ref origin/main
```

Key flags:
- `--base-ref` — git ref to compare against (default: merge-base with `main`)
- `--color` — color output: `auto` (default), `always`, `never`
- `--open` — open diffs in `$DIFFTOOL` or `git difftool` (directory comparison mode)
- `--output-dir` — write per-component `.diff` files to a directory
- `--output-mode` — output format (comma-separated): `local` (default), `ci-summary`, `ci-comment`, `ci-artifact-dir`
- `--log-file` — write debug logs to a file
- `--version` — print version and exit

#### CI output modes

The `--output-mode` flag selects how output is formatted. Multiple modes can
be combined with commas (e.g., `--output-mode=ci-summary,ci-comment,ci-artifact-dir`).
When multiple modes are specified, each mode runs independently — if one fails,
the remaining modes still execute. The CI modes are used by the `pr-render-diff`
GitHub Actions workflow and are not intended for local use:

| Mode | Description |
|------|-------------|
| `local` | Progressive colored diffs to stdout (default) |
| `ci-summary` | Posts a summary on the Checks section of the PR (collapsible per-component diffs) |
| `ci-comment` | Posts a summary table as a PR comment via the GitHub API |
| `ci-artifact-dir` | Writes raw `.diff` files to `--output-dir` for upload as an artifact |

The `ci-comment` mode reads its configuration from environment variables
rather than CLI flags, so these details are not exposed to local users:

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | API token for authentication |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format |
| `PR_NUMBER` | Pull request number to comment on |

If any of these are missing, `ci-comment` falls back to printing the comment
markdown to stdout.

## Project structure

```
infra-tools/
  cmd/
    env-detector/                   CLI entry point for env-detector
    overlay-app-collision-checker/  CLI entry point for overlay-app-collision-checker
    render-diff/                    CLI entry point for render-diff
  internal/
    appset/              ArgoCD ApplicationSet YAML parser
    collision/           Detects colliding generated Application names across co-deployed overlays
    deptree/             Kustomize dependency tree resolver
    detector/            Core detection logic (overlay building, file matching)
    git/                 Git operations (diff, worktree, merge-base)
    github/              GitHub API client (PR labels, PR comments)
    kustomize/           Kustomize build wrapper
    renderdiff/          Render diff engine (parallel builds, unified diffs, YAML normalization)
  Makefile               Build, test, lint targets
```

The `internal/` packages are shared across these tools. The `detector` package
provides the detection pipeline that `env-detector` and `render-diff` build
on: it constructs ApplicationSet overlays, resolves kustomize dependency
trees, and matches changed files to affected components.

## Development

Prerequisites: [Go 1.24+](https://go.dev/dl/)

```bash
cd infra-tools

# Build all binaries (output to bin/)
make build

# Run tests
make test

# Run linter (downloads golangci-lint on first run)
make lint

# Fix lint issues automatically
make lint-fix

# Clean build artifacts
make clean
```

### Running tests

```bash
# All tests
go test ./...

# Specific package with verbose output
go test -v ./internal/renderdiff/

# With coverage
go test ./... -coverprofile cover.out
go tool cover -html cover.out
```

### Adding a new internal package

1. Create the package under `internal/`
2. Write tests alongside the code (`*_test.go`)
3. Import it from the relevant `cmd/` entry point
4. Run `make lint` to verify

### CI

The tools are tested by `.github/workflows/infra-tools-ci.yaml`, which
triggers on changes under `infra-tools/` and runs `make test` and `make lint`.

The `render-diff` tool also runs in CI via
`.github/workflows/pr-render-diff.yaml`, which posts a summary of kustomize
render changes as a PR comment.

The `overlay-app-collision-checker` tool runs in CI via
`.github/workflows/check-overlay-app-collisions.yaml`, which fails the PR if
any co-deployed overlays would template colliding generated Application names.
