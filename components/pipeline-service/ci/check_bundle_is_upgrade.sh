#!/usr/bin/env bash
#
# Compare the changed pipelines bundle index images in two copies of the
# the repository: one at ./head and one at ./base.
# Emits any warnings to stdout and $GITHUB_OUTPUT.

# Easiest way to identify all nested deploy.yaml files
shopt -s globstar

function extract_index_catalog() {
    IMAGE_REF="${1}"
    CONTAINER=$(podman create "${IMAGE_REF}")
    tmpdir=$(mktemp -d)
    podman cp "${CONTAINER}:/configs/openshift-pipelines-operator-rh/catalog.json" "${tmpdir}"
    podman container rm "${CONTAINER}" >/dev/null
    cat "${tmpdir}/catalog.json"
}

export OUTPUT=""

for f in head/components/pipeline-service/**/deploy.yaml; do
    CONFIG=$(echo "${f}" | cut -d '/' -f2-)
    NEW_CONFIG="${f}"
    OLD_CONFIG="base/${CONFIG}"
    if [[ ! -e "${OLD_CONFIG}" ]]; then
        echo "No corresponding manifest found for ${CONFIG} in base revision. Either the cluster is new or this is a CI bug. Skipping..."
        continue
    fi

    OLD_INDEX_IMAGE=$(yq 'select(.kind == "CatalogSource") | .spec.image' "${OLD_CONFIG}")
    NEW_INDEX_IMAGE=$(yq 'select(.kind == "CatalogSource") | .spec.image' "${NEW_CONFIG}")

    if [[ "${OLD_INDEX_IMAGE}" == "${NEW_INDEX_IMAGE}" ]]; then
        echo "No change in index image for ${CONFIG}"
        continue
    fi


    OLD_CATALOG=$(extract_index_catalog "${OLD_INDEX_IMAGE}")
    NEW_CATALOG=$(extract_index_catalog "${NEW_INDEX_IMAGE}")

    OLD_BUILD_VERSION=$(echo "${OLD_CATALOG}" | jq -r 'select(.name | startswith("openshift-pipelines-operator-rh.v5.0.5")) | .properties.[] | select(.type == "olm.package") | .value.version')
    NEW_BUILD_VERSION=$(echo "${NEW_CATALOG}" | jq -r 'select(.name | startswith("openshift-pipelines-operator-rh.v5.0.5")) | .properties.[] | select(.type == "olm.package") | .value.version')

    OLD_BUILD_ID=$(echo "${OLD_BUILD_VERSION}" | cut -d '-' -f2)
    NEW_BUILD_ID=$(echo "${NEW_BUILD_VERSION}" | cut -d '-' -f2)

    if [ "${OLD_BUILD_ID}" -ge "${NEW_BUILD_ID}" ]; then
        OUTPUT="${OUTPUT}:warning: New index image in ${CONFIG} uses a package version (${NEW_BUILD_VERSION}) which is not higher than currently applied package version (${OLD_BUILD_VERSION}). When applied, operator might not upgrade to new index\n"
        LINE_NUMBER=$(grep --line-number "image: ${NEW_INDEX_IMAGE}" "${NEW_CONFIG}" | cut -d ':' -f1)
        echo "::warning file=${CONFIG},line=${LINE_NUMBER}::Index references bundle version ${NEW_BUILD_VERSION} which is not higher than previous bundle version ${OLD_BUILD_VERSION}"
    fi

done

echo "comment=${OUTPUT}" | tee -a "${GITHUB_OUTPUT:-out}"

