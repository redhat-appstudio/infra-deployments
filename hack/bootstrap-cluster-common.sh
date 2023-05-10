#!/bin/bash -e


declare -r ROOT="${BASH_SOURCE[0]%/*}"


main() {
    load_global_vars
    [[ -z "$DOCKER_IO_AUTH" ]] || \
        "${ROOT}/secret-creator/set-docker-hub-pull-secret.sh" "$DOCKER_IO_AUTH"
    "${ROOT}/secret-creator/create-dora-metrics-exporter-secrets.sh" \
        "$GITHUB_TOKEN"
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
