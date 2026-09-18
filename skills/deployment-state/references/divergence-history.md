# Divergence History

Use this workflow to determine when and why desired versions differ across targets.

Resolve each target's current effective pin with
[repository-resolution.md](repository-resolution.md). Then use path history, blame, pickaxe
search, and individual deployment commits to find when the effective values diverged and
whether the cause was a ring promotion, cluster override, rollback or revert, or layout
migration.

Useful commands:

```bash
git log --oneline origin/main -- components/<component>
git log -p origin/main -- <exact-path>
git blame origin/main -- <file>
git log -S'<tag-or-sha>' --all -- components/<component>
git show --stat <deployment-commit>
git show <deployment-commit> -- <relevant-paths>
```

History explains how current desired state arose; current `origin/main` remains authoritative.
Call out reverts, partial rings, coupled refs, and migration commits that make a simple
“version changed once” narrative inaccurate.
