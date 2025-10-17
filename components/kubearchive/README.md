# KubeArchive README

## Upgrading

To upgrade start by upgrading development, which also upgrades staging. The diff should
look like this:

```diff
diff --git a/components/kubearchive/development/kustomization.yaml b/components/kubearchive/development/kustomization.yaml
index b7d11eb00..8a5a0c9b1 100644
--- a/components/kubearchive/development/kustomization.yaml
+++ b/components/kubearchive/development/kustomization.yaml
@@ -8,7 +8,7 @@ resources:
   - release-vacuum.yaml
   - kubearchive-config.yaml
   - pipelines-vacuum.yaml
-  - https://github.com/kubearchive/kubearchive/releases/download/v1.7.0/kubearchive.yaml?timeout=90
+  - https://github.com/kubearchive/kubearchive/releases/download/v1.8.0/kubearchive.yaml?timeout=90
 
 namespace: product-kubearchive
 secretGenerator:
@@ -56,7 +56,7 @@ patches:
               spec:
                 containers:
                   - name: vacuum
-                    image: quay.io/kubearchive/vacuum:v1.7.0
+                    image: quay.io/kubearchive/vacuum:v1.8.0
   - patch: |-
       apiVersion: batch/v1
       kind: CronJob
@@ -69,7 +69,7 @@ patches:
               spec:
                 containers:
                   - name: vacuum
-                    image: quay.io/kubearchive/vacuum:v1.7.0
+                    image: quay.io/kubearchive/vacuum:v1.8.0
   - patch: |-
       apiVersion: batch/v1
       kind: CronJob
@@ -82,7 +82,7 @@ patches:
               spec:
                 containers:
                   - name: vacuum
-                    image: quay.io/kubearchive/vacuum:v1.7.0
+                    image: quay.io/kubearchive/vacuum:v1.8.0
   - patch: |-
       apiVersion: batch/v1
       kind: Job
@@ -95,7 +95,7 @@ patches:
               - name: migration
                 env:
                   - name: KUBEARCHIVE_VERSION
-                    value: v1.7.0
+                    value: v1.8.0
   # These patches add an annotation so an OpenShift service
   # creates the TLS secrets instead of Cert Manager
   - patch: |-
```

So the version should change at:

* URL that pulls KubeArchive release files.
* Patches that change the KubeArchive vacuum image for vacuum CronJobs.
* Environment variable that is used to pull the KubeArchive repository
on the database migration Job.

Then after the upgrade is successful, you can start upgrading production clusters.
Make sure to review the changes inside the KubeArchive YAML pulled from GitHub. Some
resources may change so some patches may not be useful/wrong after upgrading.
