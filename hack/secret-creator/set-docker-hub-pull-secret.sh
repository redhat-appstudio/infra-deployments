#!/bin/bash -e


main() {
    local docker_io_auth=${1:?"User name and password were not provided"}
    local auth

    echo "Configuring the cluster with a pull secret for Docker Hub"
    auth=$(mktemp)
    # Set global pull secret
    oc get secret/pull-secret -n openshift-config --template='{{index .data ".dockerconfigjson" | base64decode}}' > "$auth"
    oc registry login --registry=docker.io --auth-basic="$docker_io_auth" --to="$auth"
    oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson="$auth"
    # Set current namespace pipeline serviceaccounts which is used by buildah
    rm "$auth"
    oc registry login --registry=docker.io --auth-basic="$docker_io_auth" --to="$auth"
    oc create secret docker-registry docker-io-pull --from-file=.dockerconfigjson="$auth" -o yaml --dry-run=client | oc apply -f-
    for sa in $(oc get serviceaccounts -o custom-columns=":metadata.name" | grep '^build-pipeline-'); do
        oc secrets link $sa docker-io-pull
    done
    rm "$auth"
}


if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
