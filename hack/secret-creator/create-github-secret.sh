#!/bin/bash -e

main() {
    local github_token=${1:?"Github token was not provided"}

    echo "Creating a secret with a token for Github"
    oc create namespace application-service --dry-run=client -o yaml | oc apply -f -
    oc create -n application-service secret generic has-github-token --from-literal=token="$github_token" --dry-run=client -o yaml | oc apply -f -
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
