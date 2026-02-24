# Spec: render-diff CLI

## Overview

Local CLI tool that computes and displays kustomize render deltas for
components affected by the current branch's changes.

## Requirements

### REQ-1: Affected component detection

The tool reuses `detector.AffectedComponents()` to determine which component
paths are affected by the changed files, grouped by environment.

### REQ-2: Parallel kustomize builds

Components are built on both HEAD and the base ref in parallel using an
`errgroup` bounded by `runtime.NumCPU()`. Each build populates `HeadYAML`
and `BaseYAML` on a `ComponentDiff` struct.

### REQ-3: YAML normalization

**Status: Implemented** in `internal/renderdiff/normalize.go`. After
building, both sides are normalized by sorting the multi-document YAML
stream by `apiVersion/kind/namespace/name`. This minimizes diff noise from
resource reordering. Normalization is best-effort — unparseable documents
are left unchanged.

### REQ-4: Progressive stdout output

In default local mode, diffs are streamed to stdout as they complete via
`RunProgressive()` and a Go channel. Colored output uses ANSI codes (auto-
detected via `term.IsTerminal`, overridable with `--color`).

### REQ-5: Directory-based --open mode

`--open` writes all base YAML to a `base/` temp directory and all head YAML
to a `head/` directory, then invokes `$DIFFTOOL` (or `git difftool --dir-diff`)
once for a folder comparison. Files are named using path slugs
(e.g., `components__monitoring__prometheus__staging__base__staging.yaml`).

### REQ-6: Per-component .diff file export

`--output-dir` writes one `.diff` file per component. When `ClusterDir` is
available, it is used as the primary label (e.g.,
`stone-prod-p02--production.diff`); otherwise falls back to path-based slugs
(e.g., `components__foo__staging__staging.diff`). Collisions are handled by
appending a counter suffix (`-2`, `-3`, etc.).

### REQ-7: Build error handling

Build failures are non-fatal — the component is skipped with a `slog.Warn`.
Components with errors are included in results with an `Error` field so
formatters can report them.

### REQ-8: Summary statistics

After all diffs complete, a summary shows per-component added/removed counts
and aggregate totals.

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--repo-root` | auto-detect | Path to repository root |
| `--base-ref` | merge-base with main | Git ref to compare against |
| `--color` | `auto` | `auto`, `always`, `never` |
| `--open` | false | Open folder diff in `$DIFFTOOL` or `git difftool` |
| `--output-dir` | — | Write `.diff` files to directory |
| `--output-mode` | `local` | `local`, `ci-summary`, `ci-comment`, `ci-artifact-dir` (comma-separated for multiple) |
| `--log-file` | — | Write debug logs to file |
| `--version` | — | Print version and exit |
