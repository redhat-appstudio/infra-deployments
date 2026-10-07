# Mimir PoC object storage

SeaweedFS runs as a single `weed mini` pod behind the
`mimir-poc-seaweedfs:8333` Service. Its `mimir-tsdb` bucket is created at
startup. The pod stores data on its own 100 GiB `gp3` claim.

The existing External Secrets operator generates the `mimir-poc-s3` Secret
automatically. Its Password generator creates a random 48-character secret
key; the access key ID is `mimir-poc`. Both SeaweedFS and Mimir read these
credentials from the Secret. No AWS account, Vault entry, or manual Secret
creation is required, and credential values are never stored in Git.

The ExternalSecret runs in sync wave `-1` and uses `CreatedOnce` with an
immutable target Secret and orphan ownership. Reapplying or recreating the
ExternalSecret preserves the credential. Do not delete the generated Secret
while retaining the SeaweedFS volume: SeaweedFS persists its initial S3
identity, so changing credentials requires coordinating the server identity
and restarting all clients.

SeaweedFS cannot read MinIO's on-disk data format. The old `mimir-poc-minio`
PVC is omitted from the new manifests and will be pruned when Argo CD syncs
the staging cutover. The `gp3` StorageClass has a `Delete` reclaim policy, so
expect the old volume's data to be removed with that PVC. The sandbox
deployment and S3 write/read and Mimir query checks below were completed
first.

## Sandbox validation (2026-10-06)

Deployed the rendered overlay to the `w9wl8` sandbox cluster in the isolated
`mimir-seaweedfs-sandbox` namespace. The sandbox uses `gp3-csi` and smaller
PVCs to match its storage class and keep the test footprint small. All eight
Mimir and SeaweedFS pods became Ready with zero restarts, and all three PVCs
became Bound. The SeaweedFS startup log reported creation of `mimir-tsdb`.

An S3 smoke Job uploaded an object, downloaded and compared its contents, then
deleted it. A remote-write request to the Mimir distributor returned HTTP 200;
a query through the query frontend returned the sample value `123` for
`mimir_sandbox_probe{source="seaweedfs_test"}`.

## Automatic credential validation (2026-10-07)

The full `stone-stg-rh01` overlay renders with both credential resources, the
SeaweedFS endpoint, and no legacy MinIO resources. The Password and
ExternalSecret manifests validate against the CRD schemas read from staging.
The staging operator watches all namespaces, and its existing role permits
reading Password generators and creating and updating Secrets.

The sandbox workload test used a manually generated Secret with the same two
keys. Live validation of the External Secrets reconciliation remains pending:
the sandbox lacks those CRDs, and automatic approval review rejected their
installation as a cluster-wide change. No operator or CRDs were installed.
