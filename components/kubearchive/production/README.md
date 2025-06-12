# Production Overlays for KubeArchive

Each overlay is independent of each other, so each cluster can update at its own pace.
The `base` folder contain an `ExternalSecret` that gets patched up on each
production overlay.

*NOTE*: there is duplication for patches, but there is no solution available as far
as we know unless Infra Deployments support `--load-restrictor LoadRestrictionsNone`
via ArgoCD.


## DB Secret Paths

The paths to the DB secrets are built from
[App Interface Konflux Namespaces](https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/data/services/stonesoup/namespaces?ref_type=heads).
For example, the information to build the DB path for the cluster `stone-prod-p01` is defined on the file `stonesoup-prod-private-1.appsrep09ue1.yaml`.
Then with the information on that file build the DB path:

```text
integrations-output/external-resources/<AppSRE Cluster>/<Name>/<DB Identifier>-rds
```

* AppSRE Cluster: within the file, the id of the `cluster` property. In this example `appsrep09ue1`
* Name: within the file, the `name` property. In this example `stone-prod-p01`
* DB Identifier: within the file, the `externalResources[].identifier` property. In this example `stone-prod-p01-kube-archive`

So the path for `stone-prod-p01` is:

```text
integrations-output/external-resources/appsrep09ue1/stone-prod-p01/stone-prod-p01-kube-archive-rds
```
