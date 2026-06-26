# perf-team-prometheus-reader

Monitoring and observability resources for the Konflux Perf&Scale team. Deploys service accounts and RBAC for reading cluster Prometheus metrics, detecting OOMKills/crashloops, and reading events in tenant namespaces.

## Directory Layout

- `base/core/` — service accounts, RBAC grants, maintainer bindings (shared by development and staging)
- `base/tenants-rbac/` — event-reader roles for `konflux-perfscale-{1,2,3}-tenant` namespaces (shared by development and staging)
- `development/` — overlay for dev clusters, also defines tenant namespace resources
- `staging/base/` — overlay for staging clusters
- `production/base/` — standalone overlay with its own copies of all resources (does not reference shared `base/`)

## Promotion Workflow

Development and staging overlays reference the shared `base/` directory. Production has its own standalone copies of resources and does not inherit from `base/`.

To roll out a change:

1. **First PR** — modify files in `base/` (and `development/` or `staging/` if needed). This applies to development and staging clusters.
2. **Soak** — let the change run in staging to validate stability over time.
3. **Second PR** — copy the tested files into `production/base/`. Production PRs should be small, simple copy-pastes of what was already validated in staging to minimize risk of introducing new issues.

This separation is intentional. Do not refactor production to reference the shared `base/` — the isolation ensures production changes are always explicit and deliberate.

## Service Accounts

- **`perf-team-prometheus-reader-cluster-sa`** — reads monitoring data from cluster Prometheus. Bound to `cluster-monitoring-view` ClusterRole via `sa-read-permissions-openshift-monitoring` ClusterRoleBinding.

- **`perf-team-prometheus-reader-oomcrash-sa`** — used by [oomkill-and-crashloopbackoff-detector](https://github.com/konflux-ci/perfscale/tree/main/tools/oomkill-and-crashloopbackoff-detector) to monitor OOMKills and crashloops. Bound to `perf-team-prometheus-reader-oomcrash-role` ClusterRole (list namespaces, get/list events and pods, get pods/log and pods/status). Has a long-lived token Secret.

- **`konflux-bot-0`** (defined in tenants config, e.g. [here](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/6a23b46f6031be732474324f7da7ed0ae027739e/tenants-config/cluster/stone-prd-rh01/tenants/konflux-perfscale-1-tenant/konflux-bot-0.yaml)) — granted `perf-team-event-reader-role` (get/list/watch events) in each `konflux-perfscale-{1,2,3}-tenant` namespace via `tenants-rbac/`. This SA is used to run [Probe runs](https://docs.google.com/document/d/1lIWwBXeBxlyHkucNJu__NrAsPtRMgoPGjf_0EbHUhcs/edit?tab=t.0#heading=h.lxuqoglzqrgk) in those namespaces.

Members of the `konflux-performance` group can create tokens for both SAs via the `perf-team-sa-token-creator` Role. To become a member of that group for Red Hat clusters you need to be in [this](https://rover.redhat.com/groups/group/konflux-performance) Rover group, for Fedora cluster add yourself to [this](https://github.com/redhat-appstudio/internal-infra-deployments/blob/6802797bcb8b66968e363d5e8811ac6e1ff348e3/components/k8s-groups/production/fas/groups/konflux-performance.yaml) file.

## Production PR Template

Production PRs require `## Risk Assessment` and `## Validation` sections ([full docs](https://konflux-production-approval-tool-982507.pages.redhat.com/#/docs/developer-guide)). Use this template:

```
## Risk Assessment
**Risk Level:** Low
**Description:** <what changed and what it affects>
**Rollback:** Revert PR

## Validation
Tested on staging — no regressions observed.
Staging PR: <link to the staging PR>
```

Risk levels: Low / Medium / High / Very High. Given purpose of our component, we will be Low or Medium.
