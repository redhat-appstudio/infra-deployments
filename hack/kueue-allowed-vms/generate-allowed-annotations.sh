#!/bin/bash

set -euo pipefail

declare -r ROOT="${BASH_SOURCE[0]%/*}"

allowed_resources() {
  kustomize build "${1}" | \
    yq 'select(.kind == "ClusterQueue")' | \
    yq '[.spec.resourceGroups[].coveredResources[]]' | \
    yq '. |= del(.[] | select(. == "tekton.dev/pipelineruns"))' | \
    yq '[.[] | "kueue.konflux-ci.dev/requests-" + . ] | sort' -o=json -I=0
}

update_configmap_from_clusterqueue() {
  ALLOWED_RESOURCES=$(allowed_resources "${1}") \
    yq -i '.data.allowed-annotations = (env(ALLOWED_RESOURCES)| to_string)' "${2}"
}

update_test_resources_from_clusterqueue() {
  local input_folder="${1}"
  local output_folder="${2}"

  # calculates the allowed resources
  ALLOWED_RESOURCES=$(allowed_resources "${input_folder}")

  # updates the test PipelineRun with cluster specific allowed annotations
  TEST_ALLOWED_PLR_FILEPATH='.chainsaw-test/resources/pipelinerun_with_allowed_annotations.yaml'
  local allowed_plr_filepath="${output_folder%/}/${TEST_ALLOWED_PLR_FILEPATH}"
  echo "Generating Test PipelineRun: ${input_folder} -> ${allowed_plr_filepath}"
  ALLOWED_RESOURCES=${ALLOWED_RESOURCES} \
    yq -P -i '.metadata.annotations = (
      strenv(ALLOWED_RESOURCES) | from_json |
      map({"key": ., "value": "1"}) | from_entries)' "${allowed_plr_filepath}"

  # updates the test PipelineRun with cluster specific allowed and unallowed annotations
  TEST_UNALLOWED_PLR_FILEPATH='.chainsaw-test/resources/pipelinerun_with_allowed_and_unallowed_annotations.yaml'
  local unallowed_plr_filepath="${output_folder%/}/${TEST_UNALLOWED_PLR_FILEPATH}"
  echo "Generating Test PipelineRun: ${input_folder} -> ${unallowed_plr_filepath}"
  ALLOWED_RESOURCES=${ALLOWED_RESOURCES} \
    yq -P -i '.metadata.annotations = (
      strenv(ALLOWED_RESOURCES) | from_json | . += "kueue.konflux-ci.dev/requests-unallowed-resource" |
      map({"key": ., "value": "1"}) |
      from_entries)' "${unallowed_plr_filepath}"
}

main() {
  CONFIGMAP_FILENAME='deny-unallowed-annotations-configmap.yaml'

  # Define input-output file pairs
  local -A queue_configs=(
      ["components/kueue/staging/stone-stage-p01"]="components/policies/staging/stone-stage-p01/kueue/deny-unallowed-annotations/"
      ["components/kueue/staging/stone-stg-rh01"]="components/policies/staging/stone-stg-rh01/kueue/deny-unallowed-annotations/"
  )

  for input_folder in "${!queue_configs[@]}"; do
    echo "Generating Resources for ${input_folder}"
    local output_folder="${queue_configs[$input_folder]%/}"
    # Update ConfigMap
    local output_cm_file="${output_folder}/${CONFIGMAP_FILENAME}"
    echo "Generating ConfigMap:        $input_folder -> $output_cm_file"
    update_configmap_from_clusterqueue "${input_folder}" "${output_cm_file}"

    # Update Test Resources
    update_test_resources_from_clusterqueue "${input_folder}" "${output_folder}"
  done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  (
      cd "$ROOT/../.."
      main
  )
fi
