#!/bin/bash -e

declare -r ROOT="${BASH_SOURCE[0]%/*}"

main() {
    load_global_vars
    "${ROOT}/secret-creator/create-eaas-secrets.sh" \
      "$EAAS_HYPERSHIFT_AWS_CREDENTIALS_PATH" \
      "$EAAS_HYPERSHIFT_OIDC_PROVIDER_S3_REGION" \
      "$EAAS_HYPERSHIFT_OIDC_PROVIDER_S3_BUCKET" \
      "$EAAS_HYPERSHIFT_PULL_SECRET_PATH"
}

load_global_vars() {
    local vars_file="$ROOT/preview.env"

    if [[ -f "$vars_file" ]]; then
        source "$vars_file"
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
