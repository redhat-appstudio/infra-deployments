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
    # Set current namespace pipeline serviceaccount which is used by buildah
    rm "$auth"
    oc registry login --registry=docker.io --auth-basic="$docker_io_auth" --to="$auth"
    oc create secret docker-registry docker-io-pull --from-file=.dockerconfigjson="$auth" -o yaml --dry-run=client | oc apply -f-
    oc create serviceaccount appstudio-pipeline -o yaml --dry-run=client | oc apply -f-
    oc secrets link appstudio-pipeline docker-io-pull
    rm "$auth"
}


if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
