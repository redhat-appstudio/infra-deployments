# KubeArchive README

## Release Retention and gracePeriodDays

KubeArchive is configured to automatically delete Releases based on their retention period. The retention period can be customized using the `gracePeriodDays` field in the Release spec. KubeArchive is an exception to the typical [component layout standard](../../docs/ring-deployments/directory-layout.md) because it's resources are released as a single YAML manifest named `kubearchive.yaml`. 

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

Kargo *should* take care of promoting upgrades across rings. To start the upgrade promotion process, delete the current `base/kubearchive.yaml` file and run `curl -Lo components/kubearchive/base/kubearchive.yaml https://github.com/kubearchive/kubearchive/releases/download/<NEW_VERSION>/kubearchive.yaml` with the <NEW_VERSION>. Commit and merge this change.

If Kargo *is not* functioning properly:

1. Checkout a new branch of infra-deployments.
2. Delete the current `rings/ring-#/base/kubearchive.yaml` file and run `curl -Lo components/kubearchive/rings/ring-#/base/kubearchive.yaml https://github.com/kubearchive/kubearchive/releases/download/<NEW_VERSION>/kubearchive.yaml` with the <NEW_VERSION>.
3. Change the vacuum image versions in the `CronJob` resource files in the `rings/ring-#/base/vacuum/` folder.

*If the upgrade requires a schema migration:*

1. Checkout the branch in the PR for promoting this upgrade to ring #.
2. Bump `MIGRATION_VERSION` in both `ConfigMap`s in `rings/ring-#/base/kustomization.yaml`:
  * kubearchive-schema-version (used by the schema migration `Job`)
  * kubearchive-deployment-schema-version (used by api-server and sink)
  * Update the `Job` name suffix in the `rings/ring-#/base/patches/schema-migration-job-name-patch.yaml` file to match the release (e.g. `kubearchive-schema-migration-v14-v1.23.0`).
3. Commit and push these changes.

ArgoCD sync waves handle the ordering: the migration Job (wave -1) runs before deployments (wave 0) pick up the new version.

### Schema version configmaps

There are two `ConfigMap`s for `MIGRATION_VERSION`:

* `kubearchive-schema-version` (sync-wave -2) — used by the schema migration Job.
* `kubearchive-deployment-schema-version` (no sync-wave) — used by api-server
  and sink deployments. Protects running deployments from restarting during
  migration if the `ConfigMap` hash changes.

### Versioned Job name

A versioned Job name suffix (e.g. `kubearchive-schema-migration-v13-v1.20.0`) is used
so that ArgoCD resyncs do not restart the immutable migration Job. When changing `MIGRATION_VERSION`, update the suffix in the `rings/ring-#/base/patches/schema-migration-job-name-patch.yaml` file to match.
