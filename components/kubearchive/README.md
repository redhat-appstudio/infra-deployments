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

To upgrade start by upgrading development, which also upgrades staging. First replace the current `kubearchive.yaml` manifest
with the one from the new version, then the change the `kustomization.yaml` file so the diff looks like this:

```diff
diff --git a/components/kubearchive/development/kustomization.yaml b/components/kubearchive/development/kustomization.yaml
index 98bf1d721..ee330ff5a 100644
--- a/components/kubearchive/development/kustomization.yaml
+++ b/components/kubearchive/development/kustomization.yaml
@@ -56,7 +56,7 @@ patches:
               spec:
                 containers:
                   - name: vacuum
-                    image: quay.io/kubearchive/vacuum:v1.14.0
+                    image: quay.io/kubearchive/vacuum:v1.15.0
   - patch: |-
       apiVersion: batch/v1
       kind: CronJob
@@ -69,7 +69,7 @@ patches:
               spec:
                 containers:
                   - name: vacuum
-                    image: quay.io/kubearchive/vacuum:v1.14.0
+                    image: quay.io/kubearchive/vacuum:v1.15.0
   - patch: |-
       apiVersion: batch/v1
       kind: CronJob
@@ -82,7 +82,7 @@ patches:
               spec:
                 containers:
                   - name: vacuum
-                    image: quay.io/kubearchive/vacuum:v1.14.0
+                    image: quay.io/kubearchive/vacuum:v1.15.0
   - patch: |-
       apiVersion: batch/v1
       kind: Job
@@ -102,7 +102,7 @@ patches:
               - name: migration
                 env:
                   - name: KUBEARCHIVE_VERSION
-                    value: v1.14.0
+                    value: v1.15.0
   # These patches add an annotation so an OpenShift service
   # creates the TLS secrets instead of Cert Manager
   - patch: |-
```

So the version should change at:

* Patches that change the KubeArchive vacuum image for vacuum CronJobs.
* Environment variable that is used to pull the KubeArchive repository on the database migration Job.

Then after the upgrade is successful, you can start upgrading production clusters.
Make sure to review the changes inside the KubeArchive YAML pulled from GitHub. Some
resources may change so some patches may not be useful/wrong after upgrading.

### Upgrade Script

There is a simple bash script you can use to upgrade, run it as follows:

```
cd infra-deployments/
bash components/kubearchive/upgrade.sh <current-version> <new-version>
# For example: bash components/kubearchive/upgrade.sh v1.14.0 v1.15.0
```

This script downloads the new manifests from the GitHub repository
and replaces (using `sed`) the `<current-version>` string with the
`<new-version>` string on all `kustomization.yaml` files.
