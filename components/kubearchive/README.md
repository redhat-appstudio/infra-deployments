# KubeArchive README

## Release Retention and gracePeriodDays

KubeArchive is configured to automatically delete Releases based on their retention period. The retention period can be customized using the `gracePeriodDays` field in the Release spec. KubeArchive is an exception to the typical [component layout standard](../../docs/ring-deployments/directory-layout.md) because its resources are released as a single YAML manifest named `kubearchive.yaml`. 

### How It Works

- **Default behavior**: If a Release does not have `gracePeriodDays` specified, it will be deleted 5 days after creation.
- **Custom retention**: If a Release has `spec.gracePeriodDays` set, that value will be used instead.
- **Maximum cap**: To prevent abuse, there is a maximum retention period of 30 days. Even if `gracePeriodDays` is set higher than 30, the Release will be deleted after 30 days.

### Examples

- Release with no `gracePeriodDays`: deleted after 5 days
- Release with `gracePeriodDays: 10`: deleted after 10 days
- Release with `gracePeriodDays: 45`: deleted after 30 days (capped at maximum)

### Important notes

> Deleted Releases are archived in KubeArchive and can still be accessed through the KubeArchive API. The deletion from the cluster is automatic, but the historical data is preserved.

> Because `kubearchive.yaml` is an upstream release artifact, it is located in the `base/` directory of the component and copied to each ring's `base/base-snapshot/` directory. Resources in it **must** be patched and not modified directly as a new upgrade will overwrite any modifications with a brand new file.

## Promoting Changes

*This section is meant to be referenced once the component's migration is complete and it is onboarded to Kargo*

Kargo takes care of promoting upgrades across rings by watching for changes to the `components/kubearchive/base/` directory and promoting the changes to each ring's `base/base-snapshot/` directory starting with ring 0 (development). In other words, to trigger any promotions across rings, **a PR with changes to the `components/kubearchive/base/` directory must be created and merged.**

If Kargo **is not** functioning properly, a user will need to **manually execute** Kargo's workflow to promote the changes across the rings. Start with the **lowest** ring possible.

1. Checkout a new branch of infra-deployments.
2. Delete the `rings/ring-#/base/base-snapshot/` directory.
3. Copy the `base/` directory from the commit with the desired changes into the `rings/ring-#/base/` directory and rename it `base-snapshot.
4. Commit these changes and open a PR.

### Upgrading

Upgrading is a special type of promotion where the old `kubearchive.yaml` release artifact is replaced by then desired version's `kubearchive.yaml` release artifact.

> These steps must be executed, regardless of Kargo's functionality status.

1. Checkout a new branch of infra-deployments. 
2. Delete the current `base/kubearchive.yaml` file and run `curl -Lo components/kubearchive/base/kubearchive.yaml https://github.com/kubearchive/kubearchive/releases/download/<NEW_VERSION>/kubearchive.yaml`. *Make sure to review the changes inside this new file before committing and merging anything.*
3. Update the `version` field in the `base/VERSION.yaml` file.
4. Commit these changes and open a PR.

If Kargo **is not** functioning properly, a user will need to **manually execute** Kargo's workflow to promote the upgrade across the rings. Start with the **lowest** ring possible.

1. Checkout a new branch of infra-deployments.
2. Delete the current `rings/ring-#/base/base-snapshot/kubearchive.yaml` file and run `curl -Lo components/kubearchive/rings/ring-#/base/base-snapshot/kubearchive.yaml https://github.com/kubearchive/kubearchive/releases/download/<NEW_VERSION>/kubearchive.yaml`.
3. Change the vacuum image versions in the `CronJob` resource files in the `rings/ring-#/base/vacuum/` folder.
4. Update the `Job` name suffix in the `rings/ring-#/base/patches/schema-migration-job-name-patch.yaml` file to match the release (e.g. `kubearchive-schema-migration-v14-v1.23.0`).
5. Update the `version` field in the `rings/ring-#/base/VERSION.yaml` file.

If the upgrade *requires a schema migration:*
6. Bump `MIGRATION_VERSION` in both `ConfigMap`s in `rings/ring-#/base/kustomization.yaml`:
  * kubearchive-schema-version (used by the schema migration `Job`)
  * kubearchive-deployment-schema-version (used by api-server and sink)
7. Commit these changes and open a PR.
8. Once the PR is merged, repeat these steps for the next ring.

ArgoCD sync waves handle the ordering: the migration Job (wave -1) runs before deployments (wave 0) pick up the new version.

### Schema Version ConfigMaps

There are two `ConfigMap`s for `MIGRATION_VERSION`:

* **kubearchive-schema-version** (sync-wave -2) — used by the schema migration Job.
* **kubearchive-deployment-schema-version** (no sync-wave) — used by api-server
  and sink deployments. Protects running deployments from restarting during
  migration if the `ConfigMap` hash changes.

If the release introduces a breaking schema change that requires deployments
to be upgraded between two migration phases, split the work into two promotions:
the first bumping the 'kubearchive-schema-version' and Job name suffix, the second bumping the 'kubearchive-deployment-schema-version'.

### Versioned Job Name

A versioned Job name suffix, `kubearchive-schema-migration-v<MIGRATION_VERSION>-v<KUBEARCHIVE_VERSION>`, is used so that ArgoCD resyncs do not restart the immutable migration Job. When changing either version, update the value in the `rings/ring-#/base/patches/schema-migration-job-name-patch.yaml` file to match.
