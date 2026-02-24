## Context

The render-diff engine processes all component paths returned by `AffectedComponents()`, including directories that are plain YAML (deployed by ArgoCD as raw manifests) without a `kustomization.yaml`. When `kustomize build` runs on these directories, it fails with an error like `kustomize build <path>: <...> missing kustomization file`. The error is captured in `ComponentDiff.Error` and displayed prominently in all output modes — bold red BUILD ERROR in local output, table rows in PR comments, and collapsible sections in job summaries.

This was identified as OD-1 (Open Decision 1) in the original render-diff design. User feedback confirms that showing these errors is unhelpful since those folders were never meant to build with kustomize.

**Previous recommendation from OD-1**: Option C (handle non-kustomize paths with raw YAML concatenation) is most correct long-term. Option B (skip + log) is a pragmatic interim step. This change implements the interim step (Option B), filtering non-kustomization errors from output while preserving them in logs.

## Goals / Non-Goals

**Goals:**
- Stop showing non-kustomization directory errors in user-facing output (local, CI summary, CI comment)
- Preserve these errors in diagnostic logs (`--log-file` / stderr at DEBUG level)
- Continue showing genuine kustomize build errors (malformed kustomization, missing resources, etc.)

**Non-Goals:**
- Handling plain YAML directories as diffable content (Option C from OD-1 — deferred to a future change)
- Changing the detector or `AffectedComponents()` upstream filtering
- Modifying the kustomize build logic itself

## Decisions

### DD-1: Error classification via error string matching

**Decision**: Classify kustomize build errors by inspecting the error message for the kustomize "missing kustomization file" pattern. Non-kustomization errors are identified by checking if the error string contains the substring that krusty/kustomize emits when no `kustomization.yaml` is found in a directory.

**Alternatives considered**:
- **Pre-check for `kustomization.yaml` existence before building**: Cleaner but changes the `buildPair` contract. The `RepoBuilder` interface would need a new method (e.g., `FileExists`) or the engine would need filesystem awareness. This adds complexity beyond what's needed for filtering output.
- **Sentinel error type**: Wrapping the kustomize error in a typed error (e.g., `NotKustomizationError`). More Go-idiomatic, but the error originates from the krusty library, so we'd still need string matching at the boundary to classify it, then wrap — adding a layer without eliminating the string check.

**Rationale**: String matching is pragmatic and contained. The kustomize error message is stable (it's from the krusty library, not user-generated), and the classification is needed in exactly one place. If kustomize changes its error message in a future version, the worst case is that these errors start appearing in output again — a safe failure mode.

### DD-2: Classification in the engine, filtering in output

**Decision**: The engine (`RunProgressive`) classifies the error and stores the classification in `ComponentDiff`. Output formatters check the classification to decide whether to display the error.

- Add a `SkipOutput` boolean field to `ComponentDiff` (or similar marker)
- In `RunProgressive`, when a build error is a non-kustomization error, set `SkipOutput = true` and log at WARN level (as today — already goes to log file)
- In output formatters (`printComponentDiff`, `printSummary`, `writeCISummary`, `buildCommentBody`), skip components where `SkipOutput` is true
- The component count in summaries should exclude skipped components

**Rationale**: Keeps the engine responsible for understanding errors and output formatters responsible for display decisions. The engine already logs build errors via `slog.Warn`, so log-file visibility is preserved without additional work.

### DD-3: Exclude skipped components from DiffResult counts

**Decision**: Components with `SkipOutput = true` should not be counted in `DiffResult.Diffs` or included in summary totals. The engine collector goroutine should not append these to `result.Diffs`.

**Rationale**: If skipped components remain in `Diffs`, every output formatter must independently filter them and the "N components with differences" count becomes misleading. Filtering at the source is cleaner.

## Risks / Trade-offs

- **[String matching fragility]** → If kustomize changes its error message, non-kustomization errors will appear in output again. This is a safe degradation (reverts to current behavior). A test should pin the expected error substring.
- **[Silent failures]** → A genuine build error that happens to match the non-kustomization pattern would be hidden from output. This is mitigated by matching on the specific kustomize error message, not a generic pattern. The error still appears in the log file.
- **[Loss of visibility]** → Users won't see which directories were skipped unless they check logs. This is the intended behavior per user feedback — these directories are noise, not signal.
