# Dev backup overlay notes

This dev backup overlay assumes the MinIO Operator is already deployed on the cluster.

Current dependency:
- the MinIO Operator is deployed via `components/pipeline-service/development`

Until KONFLUX-12563 is completed, deploy/use the pipeline-service MinIO Operator first before applying this overlay.

Notes:
- This overlay includes a `Schedule` to preserve the same backup template / exclusions used by staging and production.
- In development, the `Schedule` is paused so it does not create recurring backups.
- The credentials in `backup-s3-credentials.yaml` are dev-only credentials for an ephemeral local / in-cluster MinIO bucket.
