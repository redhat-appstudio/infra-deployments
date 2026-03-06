#!/bin/bash -e

main() {
    echo "Setting secrets for pipeline-service"
    create_namespace tekton-results
    create_namespace tekton-logging
    create_namespace product-kubearchive-logging
    create_db_secret
    create_s3_secret tekton-results tekton-results-s3
    create_s3_secret tekton-logging tekton-results-s3
    create_db_cert_secret_and_configmap
}

create_namespace() {
    if kubectl get namespace $1 &>/dev/null; then
        echo "$1 namespace already exists, skipping creation"
        return
    fi
    kubectl create namespace $1 -o yaml --dry-run=client | kubectl apply -f-
}

create_db_secret() {
    echo "Creating DB secret" >&2
    if kubectl get secret -n tekton-results tekton-results-database &>/dev/null; then
        echo "DB secret already exists, skipping creation"
        return
    fi
    kubectl create secret generic -n tekton-results tekton-results-database \
      --from-literal=db.user=tekton \
      --from-literal=db.password="$(openssl rand -base64 20)" \
      --from-literal=db.host="postgres-postgresql.tekton-results.svc.cluster.local" \
      --from-literal=db.name="tekton_results"
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
      --from-literal=endpoint='https://minio.tekton-results.svc.cluster.local'

    echo "Creating MinIO config" >&2
    if kubectl get secret -n tekton-results minio-storage-configuration &>/dev/null; then
        echo "MinIO config already exists, skipping creation"
        return
    fi
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: minio-storage-configuration
  namespace: tekton-results
type: Opaque
stringData:
  config.env: |-
    export MINIO_ROOT_USER="$USER"
    export MINIO_ROOT_PASSWORD="$PASS"
    export MINIO_STORAGE_CLASS_STANDARD="EC:1"
    export MINIO_BROWSER="on"
EOF
}

create_db_cert_secret_and_configmap() {
    echo "Creating Postgres TLS certs" >&2
    if kubectl get secret -n tekton-results postgresql-tls &>/dev/null; then
        echo "Postgres DB cert secret already exists, skipping creation"
        return
    fi

    local cert_dir=".tmp/tekton-results"
    local pg_cn="postgres-postgresql.tekton-results.svc.cluster.local"
    mkdir -p "$cert_dir"

    # CA: self-signed cert (works on both macOS and Linux; avoids v3_ca / extfile issues)
    openssl req -new -x509 -nodes -days 9999 \
        -keyout "$cert_dir/ca.key" \
        -out "$cert_dir/ca.crt" \
        -subj "/CN=cluster.local" \
        2>/dev/null
    chmod og-rwx "$cert_dir/ca.key"

    # Server CSR
    openssl req -new -nodes \
        -subj "/CN=$pg_cn" \
        -addext "subjectAltName=DNS:$pg_cn" \
        -keyout "$cert_dir/tls.key" \
        -out "$cert_dir/tls.csr" \
        2>/dev/null
    chmod og-rwx "$cert_dir/tls.key"

    # Sign server cert with CA
    openssl x509 -req -days 9999 -CAcreateserial \
        -extfile <(printf "subjectAltName=DNS:%s" "$pg_cn") \
        -in "$cert_dir/tls.csr" \
        -CA "$cert_dir/ca.crt" \
        -CAkey "$cert_dir/ca.key" \
        -out "$cert_dir/tls.crt" \
        2>/dev/null

    cp "$cert_dir/ca.crt" "$cert_dir/tekton-results-db-ca.pem"

    kubectl create secret generic -n tekton-results postgresql-tls \
        --from-file="$cert_dir/ca.crt" \
        --from-file="$cert_dir/tls.crt" \
        --from-file="$cert_dir/tls.key"
    kubectl create configmap -n tekton-results rds-root-crt \
        --from-file="$cert_dir/tekton-results-db-ca.pem"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
