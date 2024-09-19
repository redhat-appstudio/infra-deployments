#!/bin/bash
set -eo pipefail

main() {
  echo "Setting secrets for EaaS (Environment as a Service)"

  local aws_credentials_path=${1:?"AWS credentials path was not provided"}
  local oidc_provider_s3_region=${2:?"OIDC provider S3 region was not provided"}
  local oidc_provider_s3_bucket=${3:?"OIDC provider S3 bucket was not provided"}
  local pull_secret_path=${4:?"OpenShift pull secret path was not provided"}

  create_namespace "local-cluster"
  create_namespace "clusters"
  create_oidc_provider_s3_secret \
    $aws_credentials_path \
    $oidc_provider_s3_region \
    $oidc_provider_s3_bucket
  create_hypershift_credentials \
    $aws_credentials_path \
    $pull_secret_path
}

create_namespace() {
  echo "Creating namespace '$1'"
  kubectl create namespace $1 -o yaml --save-config=true --dry-run=client | kubectl apply -f -
}

create_oidc_provider_s3_secret() {
  echo "Creating hypershift OIDC provider S3 secret"
  kubectl create secret generic hypershift-operator-oidc-provider-s3-credentials \
    --from-file=credentials=$1 \
    --from-literal=region=$2 \
    --from-literal=bucket=$3 \
    --save-config=true \
    --dry-run=client \
    -n local-cluster \
    -o yaml \
    | kubectl apply -f -
}

create_hypershift_credentials() {
  echo "Creating hypershift secret"
  kubectl create secret generic hypershift \
    --from-file=aws-credentials=$1 \
    --from-file=pull-secret=$2 \
    --from-literal=ssh-privatekey="not yet implemented" \
    --from-literal=ssh-publickey="not yet implemented" \
    --save-config=true \
    --dry-run=client \
    -n clusters \
    -o json \
    | jq '.metadata.labels |= {"hypershift.openshift.io/safe-to-delete-with-cluster": "false"}' \
    | kubectl apply -f -
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
