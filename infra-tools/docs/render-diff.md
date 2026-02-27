# render-diff

Computes the kustomize render delta for components affected by your
branch's changes. Shows exactly what will change in each environment
before merging.

## Building

```bash
cd infra-tools
make build
```

The binary is placed at `infra-tools/bin/render-diff`.

## Quick start

From anywhere in the repo:

```bash
./infra-tools/bin/render-diff
```

This auto-detects the repo root, computes the merge-base with main,
and prints colored unified diffs for all affected components.

## Flag reference

### Repository and ref selection

| Flag | Default | Description |
|------|---------|-------------|
| `--repo-root` | auto-detect | Path to the repository root. When omitted, detected via `git rev-parse --show-toplevel` from the current directory. Useful in CI where the checkout path is known. |
| `--base-ref` | merge-base with main | Git ref to compare against (branch, tag, or commit SHA). By default, computes `git merge-base HEAD main` so the diff reflects only your branch's changes. Use an explicit ref when comparing against a release branch or a specific commit. |
| `--overlays-dir` | `argo-cd-apps/overlays` | Path to the ArgoCD overlays directory, relative to repo root. Only change this if the repo uses a non-standard layout. |

### Output control

| Flag | Default | Description |
|------|---------|-------------|
| `--color` | `auto` | Color mode: `auto` (detect TTY), `always`, or `never`. Use `always` when piping to a pager that supports ANSI (e.g. `less -R`). |
| `--open` | off | Write base and head YAML into two temp directories and open them in `$DIFFTOOL` (or `git difftool --no-index --dir-diff`). Files are named after component and environment for easy identification. |
| `--output-dir` | — | Write per-component `.diff` files to this directory instead of stdout. Files are named like `components__foo__staging__staging.diff`. |
| `--output-mode` | `local` | Output format: `local` (unified diff to stdout), `ci-summary` (markdown for `GITHUB_STEP_SUMMARY`), `ci-comment` (PR comment markdown), `ci-artifact-dir` (raw `.diff` files to `--output-dir`). In CI, accepts comma-separated values to produce multiple outputs in a single run (e.g. `--output-mode=ci-summary,ci-comment,ci-artifact-dir`). |
| `--log-file` | — | Write DEBUG-level logs to this file. INFO-level messages always go to stderr. |
| `--version` | — | Print version and exit. |

### CI environment variables (used with `--output-mode ci-comment`)

The `ci-comment` mode reads GitHub configuration from environment
variables rather than CLI flags, keeping `--help` clean for local users.
If any variable is missing, the comment markdown is printed to stdout
instead of being posted to GitHub.

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | API token for authentication. |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format. |
| `PR_NUMBER` | Pull request number to comment on. |

## Local usage

### Colored diff to stdout (default)

```bash
./bin/render-diff
./bin/render-diff --color=always    # force color (e.g. when piping to less -R)
./bin/render-diff --color=never     # plain text
```

### Pipe to a diff viewer

```bash
./bin/render-diff | delta
./bin/render-diff | diffnav
```

### Open in a GUI diff tool (folder comparison)

```bash
DIFFTOOL=meld ./bin/render-diff --open
```

Creates two temp directories (base/ and head/) with YAML files named
after each component, then opens the diff tool for side-by-side folder
comparison. Supports any tool that accepts two directory arguments
(meld, Beyond Compare, kdiff3, etc.).

Without `$DIFFTOOL`, falls back to `git difftool --no-index --dir-diff`.

### Write .diff files to a directory

```bash
./bin/render-diff --output-dir ./my-diffs
ls ./my-diffs/
delta < ./my-diffs/components__foo__staging__staging.diff
```

### Comparing against a specific ref

```bash
./bin/render-diff --base-ref origin/release-1.0
./bin/render-diff --base-ref abc1234
```

By default, render-diff computes `git merge-base HEAD main` so the diff
only reflects your branch's changes — not everything that happened on
main since you branched. Use `--base-ref` to compare against a different
branch or commit, for example when working against a release branch
instead of main.

### Explicit repo root

```bash
./bin/render-diff --repo-root /path/to/infra-deployments
```

Normally auto-detected. Specify explicitly when running from outside
the repo tree or in CI where the checkout path is known.

## CI output modes

These modes are designed for CI but can be run locally to preview the
output. When run locally without the CI environment variables, all
output goes to stdout.

### Previewing CI output locally

You can preview what the CI workflow will produce:

```bash
./bin/render-diff --output-mode ci-summary     # preview the job summary markdown
./bin/render-diff --output-mode ci-comment      # preview the PR comment markdown
./bin/render-diff --output-mode ci-artifact-dir --output-dir /tmp/artifacts  # preview artifact files
```

### How the CI workflow uses it

The CI workflow combines all three modes in a single invocation using
comma-separated values and sets environment variables for GitHub API
access:

```bash
./bin/render-diff \
  --output-mode=ci-summary,ci-comment,ci-artifact-dir \
  --output-dir=../render-diff-output
```

This runs the engine once and produces all three outputs. The
`ci-comment` mode reads `GITHUB_TOKEN`, `GITHUB_REPOSITORY`, and
`PR_NUMBER` from the environment to post or update the PR comment.
When these variables are missing (i.e. local runs), it prints the
comment markdown to stdout instead.

### Output mode details

- **ci-summary** — markdown with collapsible `<details>` blocks per
  component, suitable for `$GITHUB_STEP_SUMMARY`. Diffs over 50KB per
  component are truncated with a pointer to the artifact.
- **ci-comment** — markdown table with component, environment, and +/-
  line counts. Posted to the PR using an HTML comment marker for
  idempotent updates.
- **ci-artifact-dir** — one `.diff` file per component/environment pair,
  written to `--output-dir`. Uploaded as GitHub Actions artifacts.

## Debug logging

```bash
./bin/render-diff --log-file /tmp/debug.log
cat /tmp/debug.log
```

Writes DEBUG-level messages (component matching details, build timings)
to the file while showing only INFO on stderr. Useful for diagnosing
why a component was or wasn't detected.

## Viewing .diff files

```bash
delta < file.diff          # syntax-highlighted
diffnav < file.diff        # interactive TUI with file tree
bat file.diff              # highlighted with bat
less file.diff             # plain viewer
```
