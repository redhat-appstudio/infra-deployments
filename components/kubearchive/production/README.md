# Production Overlays for KubeArchive

Each overlay is independent of each other, so each cluster can update at its own pace.
The `base` folder contain an `ExternalSecret` that gets patched up on each
production overlay.

*NOTE*: there is duplication for patches, but there is no solution available as far
as we know unless Infra Deployments support `--load-restrictor LoadRestrictionsNone`
via ArgoCD.
