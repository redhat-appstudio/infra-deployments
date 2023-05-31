#!/bin/bash -e

main() {
    local github_token=${1:?"Github token was not provided"}
    local tokens_list=${2}

    oc create namespace application-service --dry-run=client -o yaml | oc apply -f -
    # if not multiple tokens provided then use legacy one.
    if [[ -z "${tokens_list}" ]]; then
        echo "Creating a has secret from legacy token"
        oc create -n application-service secret generic has-github-token --from-literal=token="$github_token" --dry-run=client -o yaml | oc apply -f -
    else
        echo "Creating a secret with multiple tokens from Github"
        oc create -n application-service secret generic has-github-token --from-literal=tokens="$tokens_list" --dry-run=client -o yaml | oc apply -f -
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
### DO NOT MERGE TESTING CI