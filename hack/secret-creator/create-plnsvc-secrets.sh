#!/bin/bash -e

main() {
    echo "Setting secrets for pipeline-service"
    create_namespace
    create_db_secret
    create_s3_secret
    create_db_cert_secret_and_configmap
}

create_namespace() {
    if kubectl get namespace tekton-results &>/dev/null; then
        echo "tekton-results namespace already exists, skipping creation"
        return
    fi
    kubectl create namespace tekton-results -o yaml --dry-run=client | kubectl apply -f-
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
    if kubectl get secret -n tekton-results tekton-results-s3 &>/dev/null; then
        echo "S3 secret already exists, skipping creation"
        return
    fi
    USER=minio
    PASS="$(openssl rand -base64 20)"
    kubectl create secret generic -n tekton-results tekton-results-s3 \
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
    export MINIO_STORAGE_CLASS_STANDARD="EC:2"
    export MINIO_BROWSER="on"
EOF
}

create_db_cert_secret_and_configmap() {
    echo "Creating Postgres TLS certs" >&2
    if kubectl get secret -n tekton-results postgresql-tls &>/dev/null; then
        echo "Postgres DB cert secret already exists, skipping creation"
        return
    fi
    mkdir -p .tmp/tekton-results
    openssl req -new -nodes -text \
        -keyout ".tmp/tekton-results/ca.key" \
        -out ".tmp/tekton-results/ca.csr" \
        -subj "/CN=cluster.local" \
        > /dev/null
    chmod og-rwx ".tmp/tekton-results/ca.key"
    openssl x509 -req -days 9999 -text -extensions v3_ca \
        -signkey ".tmp/tekton-results/ca.key" \
        -in ".tmp/tekton-results/ca.csr" \
        -extfile "/etc/ssl/openssl.cnf" \
        -out ".tmp/tekton-results/ca.crt" \
        > /dev/null
    openssl req -new -nodes -text \
        -subj "/CN=postgres-postgresql.tekton-results.svc.cluster.local" \
        -addext "subjectAltName=DNS:postgres-postgresql.tekton-results.svc.cluster.local" \
        -out ".tmp/tekton-results/tls.csr" \
        -keyout ".tmp/tekton-results/tls.key" \
        > /dev/null
    chmod og-rwx ".tmp/tekton-results/tls.key"
    openssl x509 -req -text -days 9999 -CAcreateserial \
        -extfile <(printf "subjectAltName=DNS:postgres-postgresql.tekton-results.svc.cluster.local") \
        -in ".tmp/tekton-results/tls.csr" \
        -CA ".tmp/tekton-results/ca.crt" \
        -CAkey ".tmp/tekton-results/ca.key" \
        -out ".tmp/tekton-results/tls.crt" \
        > /dev/null
    cat ".tmp/tekton-results/ca.crt" > ".tmp/tekton-results/tekton-results-db-ca.pem"
    kubectl create secret generic -n tekton-results postgresql-tls \
        --from-file=.tmp/tekton-results/ca.crt \
        --from-file=.tmp/tekton-results/tls.crt \
        --from-file=.tmp/tekton-results/tls.key
    kubectl create configmap -n tekton-results rds-root-crt \
        --from-file=.tmp/tekton-results/tekton-results-db-ca.pem
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
