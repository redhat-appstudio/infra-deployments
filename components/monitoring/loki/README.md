# Loki

In-cluster log storage for Konflux, deployed via the Loki Operator.

The `ClusterLogForwarder` in `components/monitoring/logging` writes into the `LokiStack` defined here. The `development/` overlay backs it with an in-cluster MinIO instance, while `staging/` uses AWS S3.

## Prerequisite

The `logging-loki-s3` secret must be created by hand before the first deploy. It holds S3 credentials, so it is never committed to git.

On **development** the same secret serves two purposes: MinIO reads it as its own root credentials, and the `LokiStack` reads it as its S3 credentials. If it is missing, the MinIO pod will not start and the `LokiStack` will not become ready.

It must live in the `openshift-logging` namespace and contain exactly these four keys:

- `endpoint` - `http://minio.openshift-logging.svc:9000`, the MinIO `api` port
- `bucketnames` - `loki`, created by the `minio-create-bucket` job
- `access_key_id` - generated, also used as MinIO's `MINIO_ROOT_USER`
- `access_key_secret` - generated, also used as MinIO's `MINIO_ROOT_PASSWORD`

> The key is `access_key_secret`, **not** `secret_access_key`. The Loki Operator rejects the latter in static credential mode.

Generate it with random credentials:

```sh
oc create secret generic logging-loki-s3 \
  --namespace openshift-logging \
  --from-literal=endpoint="http://minio.openshift-logging.svc:9000" \
  --from-literal=bucketnames="loki" \
  --from-literal=access_key_id="$(openssl rand -hex 16)" \
  --from-literal=access_key_secret="$(openssl rand -hex 32)"
```

On **staging and production** the bucket and IAM role are provisioned by Terraform and the `LokiStack` authenticates via `credentialMode: token`, so no static keys are used.

## Deploying to Development

The namespace lives in the logging component, so apply that first. The `LokiStack` CR cannot be applied until the operator has registered its CRD, which means a manual install needs two passes:
```sh
# 1. Creates the openshift-logging namespace
kustomize build components/monitoring/logging/development | oc apply -f -

# 2. Create the secret (see above) before anything reads it

# 3. First pass: installs the operator; the LokiStack apply fails here
kustomize build components/monitoring/loki/development | oc apply -f -

# 4. Wait for the operator to register the CRD
oc wait --for=condition=Established \
  crd/lokistacks.loki.grafana.com --timeout=300s

# 5. Second pass: creates the LokiStack
kustomize build components/monitoring/loki/development | oc apply -f -
```

ArgoCD does not need the two passes: the ApplicationSet sets `retry: {limit: 50, backoff: {duration: 15s}}`, so the `LokiStack` is reapplied once the CRD is registered.

Verify:

```sh
oc get lokistack logging-loki -n openshift-logging
oc get pods -n openshift-logging -l app.kubernetes.io/instance=logging-loki
```
