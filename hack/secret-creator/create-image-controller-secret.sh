#!/bin/bash -e

main() {
    local organization=${1:?"Organization was not provided"}
    local quay_token=${2:?"Quay token was not provided"}

    echo "Creating a secret with a token for Image Controller"
    oc create namespace image-controller --dry-run=client -o yaml | oc apply -f -
    oc create -n image-controller secret generic quaytoken --from-literal=organization="$organization" --from-literal=quaytoken="$quay_token" --dry-run=client -o yaml | oc apply -f -
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
