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
  echo "Creating DB cert secret" >&2
  if kubectl get secret -n tekton_results postgresql-tls &>/dev/null; then
    echo "DB cert secret already exists, skipping creation"
    return
  fi
  mkdir -p .tmp/tekton-results
  openssl req -newkey rsa:4096 -nodes -text \
    -keyout ".tmp/tekton-results/root.key" \
    -out ".tmp/tekton-results/root.csr" \
    -subj "/CN=postgres-postgresql.tekton-results.svc.cluster.local" \
    -addext "subjectAltName=DNS:postgres-postgresql.tekton-results.svc.cluster.local" \
    > /dev/null 2>&1
  chmod og-rwx ".tmp/tekton-results/root.key"
  openssl x509 -req -days 7 -text \
    -signkey ".tmp/tekton-results/root.key" \
    -in ".tmp/tekton-results/root.csr" \
    -extfile "/etc/ssl/openssl.cnf" \
    -extensions v3_ca \
    -out ".tmp/tekton-results/ca.crt" \
    > /dev/null 2>&1
  openssl req -new -nodes -text \
    -out ".tmp/tekton-results/root.csr" \
    -keyout ".tmp/tekton-results/tls.key" \
    -subj "/CN=postgres-postgresql.tekton-results.svc.cluster.local" \
    -addext "subjectAltName=DNS:postgres-postgresql.tekton-results.svc.cluster.local" \
    > /dev/null 2>&1
  chmod og-rwx ".tmp/tekton-results/tls.key"
  openssl x509 -req -text -days 7 -CAcreateserial \
    -in ".tmp/tekton-results/root.csr" \
    -CA ".tmp/tekton-results/ca.crt" \
    -CAkey ".tmp/tekton-results/root.key" \
    -out ".tmp/tekton-results/tls.crt" \
    > /dev/null 2>&1
  cat ".tmp/tekton-results/ca.crt" ".tmp/tekton-results/tls.crt" > ".tmp/tekton-results/tekton-results-db-ca.pem"
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
