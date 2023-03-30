#!/bin/bash -e

declare -r ROOT="${BASH_SOURCE[0]%/*}"

main() {
    load_global_vars
    "${ROOT}/secret-creator/create-quality-dashboard-secrets.sh"
    "${ROOT}/secret-creator/create-pact-broker-secret.sh" "${BROKER_USERNAME:-undefined}" "${BROKER_PASSWORD:-undefined}"
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
