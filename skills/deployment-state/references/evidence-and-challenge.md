# Evidence and Challenger Pass

## Evidence standard

Every material conclusion must be traceable to evidence another agent can open or reproduce.
Prefer links or precise identifiers for:

- infra-deployments files and fields at the examined `origin/main` commit;
- ApplicationSet cluster membership and inheritance edges;
- deployment commits and pull requests with timestamps;
- source commits, ancestry results, source pull requests, and linked issues;
- provenance-derived repository and commit mappings.

For GitHub-hosted evidence, prefer stable blob, commit, pull request, and issue URLs pinned to the
examined revision over branch-relative links or local paths alone. Local paths remain useful
for reproducible commands, but a challenger should be able to follow the durable URL.

Record commands and exact revisions when a result depends on rendering, ancestry, or a
time range. A search-result snippet is a lead, not final evidence; follow its underlying
file, commit, pull request, issue, or attestation.

## When a challenger is required

Use an independent sub-agent for broad multi-cluster investigations, historical rollout
claims, source ancestry, provenance mappings, and interpreted bug-fix summaries. A narrow
current-state answer may instead be verified directly by the primary agent.

Give the challenger the user's question, the draft findings, and their evidence links or
identifiers. Ask it to independently:

1. follow every material evidence link or identifier;
2. verify cluster targeting and the effective inheritance path;
3. reproduce pin resolution, timestamps, ancestry, and source ranges where applicable;
4. identify counterexamples, missing clusters, overrides, non-linear history, and
   confirmed-versus-inferred wording errors;
5. mark each conclusion supported, contradicted, or unresolved.

The challenger must not accept the first pass's summary as evidence. It should inspect the
underlying artifacts. Revise or qualify contradicted and unresolved findings before answering.
If evidence cannot be followed because of access or network limits, disclose that limitation
instead of treating the finding as verified.

If no independent agent slot is available, do not claim the required challenge succeeded.
Either wait/delegate when possible or label the affected conclusions provisional and identify
the specific evidence that still needs independent verification.

## Avoid circular verification

When practical, have the challenger use an independent route: rendered output versus raw
inheritance tracing, source Git ancestry versus deployment tag assumptions, or pull request/issue
content versus commit-message interpretation. Independence matters more than repeating the
same grep command.
