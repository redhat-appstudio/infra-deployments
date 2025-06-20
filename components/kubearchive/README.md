# KubeArchive README

## Upgrading

To upgrade start by upgrading development, which also upgrades staging. The diff should
look like this:

```diff
diff --git a/components/kubearchive/development/kustomization.yaml b/components/kubearchive/development/kustomization.yaml
index aa2d0f98..982086c2 100644
--- a/components/kubearchive/development/kustomization.yaml
+++ b/components/kubearchive/development/kustomization.yaml
@@ -4,7 +4,7 @@ kind: Kustomization
 resources:
   - ../base
   - postgresql.yaml
-  - https://github.com/kubearchive/kubearchive/releases/download/v1.0.1/kubearchive.yaml?timeout=90
+  - https://github.com/kubearchive/kubearchive/releases/download/v1.1.0/kubearchive.yaml?timeout=90

 namespace: product-kubearchive
 secretGenerator:
@@ -36,7 +36,7 @@ patches:
               - name: migration
                 env:
                   - name: KUBEARCHIVE_VERSION
-                    value: v1.0.1
+                    value: v1.1.0
   # These patches add an annotation so an OpenShift service
   # creates the TLS secrets instead of Cert Manager
   - patch: |-
```

So you need to change the URL of the file and the KUBEARCHIVE_VERSION in the
migration Job.

Then after the upgrade is successful, you can start upgrading production clusters.
Make sure to review the changes inside the KubeArchive YAML pulled from GitHub. Some
resources may change so some patches may not be useful/wrong after upgrading.
