# KubeArchive README

## Release Retention and gracePeriodDays

KubeArchive is configured to automatically delete Releases based on their retention period. The retention period can be customized using the `gracePeriodDays` field in the Release spec.

### How it works

- **Default behavior**: If a Release does not have `gracePeriodDays` specified, it will be deleted 5 days after creation.
- **Custom retention**: If a Release has `spec.gracePeriodDays` set, that value will be used instead.
- **Maximum cap**: To prevent abuse, there is a maximum retention period of 30 days. Even if `gracePeriodDays` is set higher than 30, the Release will be deleted after 30 days.

### Examples

- Release with no `gracePeriodDays`: deleted after 5 days
- Release with `gracePeriodDays: 10`: deleted after 10 days
- Release with `gracePeriodDays: 45`: deleted after 30 days (capped at maximum)

### Important note

Deleted Releases are archived in KubeArchive and can still be accessed through the KubeArchive API. The deletion from the cluster is automatic, but the historical data is preserved.

## Upgrading

To upgrade start by upgrading development, which also upgrades staging:

1. Replace `development/kubearchive.yaml` with the manifest from the new release.
2. Update the vacuum image tags in `development/kustomization.yaml`.
3. If the new release includes a schema change:
   * Bump `MIGRATION_VERSION` in both configmaps in `development/kustomization.yaml`:
     `kubearchive-schema-version` (used by the migration Job) and
     `kubearchive-deployment-schema-version` (used by api-server and sink).
   * Update the Job name suffix in `development/kustomization.yaml` to match
     (e.g. `kubearchive-schema-migration-v14`).

All changes go in a single commit. ArgoCD sync waves handle the ordering:
the migration Job (wave -1) runs before deployments (wave 0) pick up the new
version.

If the release introduces a breaking schema change that requires deployments
to be upgraded between two migration phases, split the work into two commits:
first bump `kubearchive-schema-version` and the Job name suffix, then after
migration completes bump `kubearchive-deployment-schema-version`.

### Schema version configmaps

There are two configmaps for `MIGRATION_VERSION`:

* `kubearchive-schema-version` (sync-wave -2) — used by the migration Job.
* `kubearchive-deployment-schema-version` (no sync-wave) — used by api-server
  and sink deployments. Protects running deployments from restarting during
  migration if the configmap hash changes.

### Versioned Job name

A versioned Job name suffix (e.g. `kubearchive-schema-migration-v13`) is used
so that ArgoCD resyncs do not restart the immutable migration Job. When
changing `MIGRATION_VERSION`, update the suffix in
`development/kustomization.yaml` to match.

### Production

After the staging upgrade is successful, start upgrading production clusters.
Make sure to review the changes inside the KubeArchive YAML pulled from GitHub.
Some resources may change so some patches may not be useful/wrong after upgrading.

### Upgrade Script

There is a simple bash script you can use to upgrade, run it as follows:

```
cd infra-deployments/
bash components/kubearchive/upgrade.sh <current-version> <new-version> [<new-migration-version>]
```

Examples:

```bash
# No schema change:
bash components/kubearchive/upgrade.sh v1.21.3 v1.21.4

# With schema change (migration version bumps to 14):
bash components/kubearchive/upgrade.sh v1.21.4 v1.22.0 14
```

The script downloads the new manifests from the GitHub repository
and replaces the `<current-version>` string with the `<new-version>`
string on all `kustomization.yaml` files. When `<new-migration-version>`
is provided, it also updates `MIGRATION_VERSION` in both configmaps and
the Job name suffix in `development/kustomization.yaml`.
