# Production Overlays for KubeArchive

Each overlay is independent of each other, so each cluster can update at its own pace.
The `base` folder contain an `ExternalSecret` that gets patched up on each
production overlay.

*NOTE*: there is duplication for patches, but there is no solution available as far
as we know unless Infra Deployments support `--load-restrictor LoadRestrictionsNone`
via ArgoCD.


## DB Secret Paths

For new clusters the database is created automatically with a Konflux automation and the secret
is stored in a different place. Older clusters had the database created using `app-interface`.

Old clusters (`app-interface`)

* stone-stg-rh01
* stone-stage-p01
* stone-prd-rh01
* kflux-prd-rh02
* stone-prod-p01
* stone-prod-p02
* kflux-ocp-p01

New clusters (Konflux Automation):

* kflux-prd-rh03
* kflux-rhel-p01
* kflux-osp-p01

### app-interface databases

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

### Konflux Automation

The databases created using the new Konflux Automation use `ExternalSecret` with a Vault instance. The
`secretStoreRef` should be `apprse-stonesoup-vault` and the `key` contains the name of the cluster.
This is an example using the `kflux-prd-rh03` cluster:

```yaml
--
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  annotations:
    argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true
    argocd.argoproj.io/sync-wave: "-1"
  name: database-secret
  namespace: product-kubearchive
spec:
  dataFrom:
  - extract:
      key: production/platform/terraform/generated/kflux-prd-rh03/kubearchive-database
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: appsre-stonesoup-vault
  target:
    creationPolicy: Owner
    deletionPolicy: Delete
    name: kubearchive-database-credentials
    template:
      data:
        DATABASE_DB: '{{ index . "db.name" }}'
        DATABASE_KIND: postgresql
        DATABASE_PASSWORD: '{{ index . "db.password" }}'
        DATABASE_PORT: "5432"
        DATABASE_URL: '{{ index . "db.host" }}'
        DATABASE_USER: '{{ index . "db.user" }}'
```

To check if the database is created, ask [#forum-konflux-infrastructure](https://redhat.enterprise.slack.com/archives/C04F4NE15U1).
However you can assume the database and its secret were created successfuly. If something goes wrong
with the `ExternalSecret` contact the infrastructure team.
