#!/bin/bash -e

declare -r ROOT="${BASH_SOURCE[0]%/*}"

main() {
    load_global_vars
    "${ROOT}/secret-creator/create-gitops-secrets.sh"
    [[ -z "$MY_GITHUB_TOKEN" ]] ||
        "${ROOT}/secret-creator/create-github-secret.sh" "$MY_GITHUB_TOKEN"
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
