# Mimir PoC object storage

SeaweedFS runs as a single `weed mini` pod behind the
`mimir-poc-seaweedfs:8333` Service. Its `mimir-tsdb` bucket is created at
startup. The pod stores data on its own 100 GiB `gp3` claim.

`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are credentials for this
SeaweedFS S3 endpoint, not for AWS. Both SeaweedFS and Mimir read them from
the same `mimir-poc-s3` Kubernetes Secret. Before syncing this overlay, an
operator with Secret permissions must create it in `appstudio-mimir-poc`.
The key ID can be `mimir-poc`; the secret key should be a random value. Do not
put the secret key in Git. The Secret must exist before either workload rolls
out.

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
