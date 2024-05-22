#!/bin/bash
set -eo pipefail

main() {
  echo "Setting secrets for EaaS (Environment as a Service)"

  local aws_access_key_id=${1:?"AWS access key id was not provided"}
  local aws_secret_access_key=${2:?"AWS secret access key was not provided"}
  local oidc_provider_s3_region=${3:?"OIDC provider S3 region was not provided"}
  local oidc_provider_s3_bucket=${4:?"OIDC provider S3 bucket was not provided"}
  local pull_secret_path=${5:?"OpenShift pull secret path was not provided"}
  local base_domain=${6:?"Route53 base domain was not provided"}

  create_namespace "local-cluster"
  create_namespace "clusters"
  create_oidc_provider_s3_secret \
    $aws_access_key_id \
    $aws_secret_access_key \
    $oidc_provider_s3_region \
    $oidc_provider_s3_bucket
  create_hypershift_credentials \
    $aws_access_key_id \
    $aws_secret_access_key \
    $pull_secret_path \
    $base_domain
}

create_namespace() {
  echo "Creating namespace '$1'"
  kubectl create namespace $1 -o yaml --save-config=true --dry-run=client | kubectl apply -f -
}

create_oidc_provider_s3_secret() {
  echo "Creating hypershift OIDC provider S3 secret"
  local creds=$(mktemp)
  echo "[default]" >> $creds
  echo "aws_access_key_id=$1" >> $creds
  echo "aws_secret_access_key=$2" >> $creds
  kubectl create secret generic hypershift-operator-oidc-provider-s3-credentials \
    --from-file=credentials=$creds \
    --from-literal=region=$3 \
    --from-literal=bucket=$4 \
    --save-config=true \
    --dry-run=client \
    -n local-cluster \
    -o yaml | \
    kubectl apply -f -
  rm "$creds"
}

create_hypershift_credentials() {
  echo "Creating hypershift secret"
  kubectl create secret generic hypershift \
    --from-literal=aws_access_key_id=$1 \
    --from-literal=aws_secret_access_key=$2 \
    --from-file=pullSecret=$3 \
    --from-literal=baseDomain=$4 \
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
