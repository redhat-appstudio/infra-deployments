#!/bin/bash

set -euo pipefail

declare -r ROOT="${BASH_SOURCE[0]%/*}"

update_configmap_from_clusterqueue() {
  ALLOWED_RESOURCES=$(kustomize build "${1}" | \
    yq 'select(.kind=="ClusterQueue")' | \
    yq '[.spec.resourceGroups[].coveredResources[] | "kueue.konflux-ci.dev/requests-" + . ] ' -o json -I=0) \
      yq -i '.data.allowed-annotations = env(ALLOWED_RESOURCES)' "${2}"
}

main() {
    # Define input-output file pairs
    local -A queue_configs=(
        ["components/kueue/staging/stone-stage-p01"]="components/policies/staging/stone-stage-p01/kueue/deny-unallowed-annotations/deny-unallowed-annotations-configmap.yaml"
        ["components/kueue/staging/stone-stg-rh01"]="components/policies/staging/stone-stg-rh01/kueue/deny-unallowed-annotations/deny-unallowed-annotations-configmap.yaml"
    )

    for input_file in "${!queue_configs[@]}"; do
        local output_file="${queue_configs[$input_file]}"
        echo "Generating queue config: $input_file -> $output_file"

        update_configmap_from_clusterqueue "${input_file}" "${output_file}"
    done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    (
        cd "$ROOT/../.."
        main
    )
fi
