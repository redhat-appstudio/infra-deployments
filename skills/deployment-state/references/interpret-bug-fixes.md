# Interpret Bug Fixes

First establish a verified source change range with
[source-change-range.md](source-change-range.md). Then inspect commit messages, diffs where
necessary, pull request descriptions, and linked issues.

Report separately:

- **Confirmed fixes:** commits, merged pull requests, or linked issues explicitly describe
  the change as a fix.
- **Likely fixes:** evidence supports a bug-fix interpretation but does not explicitly say so.
- **Other changes:** relevant features, refactors, dependency updates, or operational changes.

Explain the evidence briefly and link it. Never convert a plausible code interpretation into
a confirmed user-visible fix. This workflow is intentionally more expensive; do not run it
when the user only needs a known fix identifier checked. Delegate components to sub-agents
when the source range is broad.
