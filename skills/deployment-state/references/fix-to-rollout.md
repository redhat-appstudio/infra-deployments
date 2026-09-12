# Fix to Rollout

Use this workflow to determine which desired targets include a known component source fix
and when each target's desired state first included it.

1. Resolve each requested cluster's effective deployed source revision at `origin/main`
   using [repository-resolution.md](repository-resolution.md).
2. Use that reference's source-resolution and ancestry procedure to test whether the
   deployed revision contains the fix.
3. Classify each target as includes fix, predates fix, not targeted, or unresolved.
4. For included targets, identify the `origin/main` deployment commit that first changed
   that target's effective pin to the fix or a descendant. Its merge/commit time is the
   desired-state rollout time.
5. Account for inheritance: one base change may affect several clusters, while ring bases
   and cluster overrides can produce different times.

## Operator-managed operands

For an operand embedded in an umbrella operator, first-inclusion timing may require walking
historical operator image promotions backward, resolving each relevant image to its source,
and testing the embedded operand revision for ancestry with the fix.

An efficient boundary search is:

1. Extract the sequence of operator image pins and their infra-deployments commits for the
   relevant ring path.
2. Deduplicate pins and provenance-resolve each image only once.
3. Inspect the embedded operand revision and test it against the fix.
4. Bracket the last excluding and first including operator images, using binary search when
   the promotion sequence is long.
5. Map the first including image back to the deployment commit and desired-state merge time.

Delegate rings to sub-agents when the sequence is large and cache resolved image-to-source
mappings. If provenance or source history is unavailable, stop with an explicit unresolved
boundary. Use the first image proven to include the fix; the current operator image's
promotion time alone does not establish first inclusion.
