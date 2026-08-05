#!/bin/bash -e

main() {
    create_namespace openshift-pipelines
    echo "Setting secrets for pipeline-service"
    create_namespace tekton-logging
    create_namespace product-kubearchive-logging
    create_s3_secret openshift-pipelines tekton-results-s3
    create_s3_secret tekton-logging tekton-results-s3
}

create_namespace() {
    if kubectl get namespace $1 &>/dev/null; then
        echo "$1 namespace already exists, skipping creation"
        return
    fi
    kubectl create namespace $1 -o yaml --dry-run=client | kubectl apply -f-
}

create_s3_secret() {
    echo "Creating S3 secret" >&2
    if kubectl get secret -n $1 $2 &>/dev/null; then
        echo "S3 secret already exists, skipping creation"
        return
    fi
    USER=minio
    PASS="$(openssl rand -base64 20)"
    kubectl create secret generic -n $1 $2 \
      --from-literal=aws_access_key_id="$USER" \
      --from-literal=aws_secret_access_key="$PASS" \
      --from-literal=aws_region='not-applicable' \
      --from-literal=bucket=tekton-results \
      --from-literal=endpoint='https://minio.openshift-pipelines.svc.cluster.local'

    echo "Creating MinIO config" >&2
    if kubectl get secret -n openshift-pipelines minio-storage-configuration &>/dev/null; then
        echo "MinIO config already exists, skipping creation"
        return
    fi
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: minio-storage-configuration
  namespace: openshift-pipelines
type: Opaque
stringData:
  config.env: |-
    export MINIO_ROOT_USER="$USER"
    export MINIO_ROOT_PASSWORD="$PASS"
    export MINIO_STORAGE_CLASS_STANDARD="EC:1"
    export MINIO_BROWSER="on"
EOF
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
