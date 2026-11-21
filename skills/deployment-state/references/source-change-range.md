# Source Change Range

Use this workflow after identifying a rollout's old and new deployment pins. If those pins
are not known yet, first use [rollout-inventory.md](rollout-inventory.md).

For each selected component:

1. Resolve the old and new pins to source repository commits using
   [repository-resolution.md](repository-resolution.md).
2. Verify repository identity and ancestry. If the revisions are not linearly related, use
   an appropriate merge base or range and make the relationship explicit.
3. List the source commits and associated pull requests between the old and new revisions.
4. Cite stable source commit and pull request URLs.

Keep the result as a concise change inventory unless interpretation is requested. For bug-fix
interpretation, continue with [interpret-bug-fixes.md](interpret-bug-fixes.md).
