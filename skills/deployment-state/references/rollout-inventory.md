# Rollout Inventory

Use this workflow for questions such as “what changed on cluster X this morning?”

1. Resolve current ApplicationSet targeting with
   [repository-resolution.md](repository-resolution.md) before searching history. If a
   component is explicitly not targeted, stop unless the user asks about former targeting.
2. Interpret the requested time window in the user's timezone and state the exact range.
3. Resolve which ApplicationSets and component paths target the cluster.
4. Inspect `origin/main` commits in the range and retain only changes affecting those
   effective paths, including inherited environment or ring bases.
5. Produce a concise inventory: component, old pin, new pin, desired-state merge time,
   deployment commit or pull request when available, and defining path.
6. Stop unless the user asks for source commits, pull requests, or interpreted fixes. For a
   large inventory, delegate components to sub-agents and merge only evidence-backed results.

Changes to bootstrap templates alone are not rollouts to existing clusters.
