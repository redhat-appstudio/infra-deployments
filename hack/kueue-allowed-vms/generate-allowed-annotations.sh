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
  # Updates the shared chainsaw test PipelineRun fixtures in a ring's base
  # kueue/deny-unallowed-annotations/.chainsaw-test/resources/ directory.
  #
  # Arguments:
  #   $1 - kueue ClusterQueue overlay directory (source of allowed-resources)
  #   $2 - exact path to the .chainsaw-test/resources/ directory to update
  local input_folder="${1}"
  local chainsaw_resources_dir="${2%/}"

  # calculates the allowed resources
  ALLOWED_RESOURCES=$(allowed_resources "${input_folder}")

  # updates the test PipelineRun with cluster specific allowed annotations
  local allowed_plr_filepath="${chainsaw_resources_dir}/pipelinerun_with_allowed_annotations.yaml"
  echo "Generating Test PipelineRun: ${input_folder} -> ${allowed_plr_filepath}"
  ALLOWED_RESOURCES=${ALLOWED_RESOURCES} \
    yq -P -i '.metadata.annotations = (
      strenv(ALLOWED_RESOURCES) | from_json |
      map({"key": ., "value": "1"}) | from_entries)' "${allowed_plr_filepath}"

  # updates the test PipelineRun with cluster specific allowed and unallowed annotations
  local unallowed_plr_filepath="${chainsaw_resources_dir}/pipelinerun_with_allowed_and_unallowed_annotations.yaml"
  echo "Generating Test PipelineRun: ${input_folder} -> ${unallowed_plr_filepath}"
  ALLOWED_RESOURCES=${ALLOWED_RESOURCES} \
    yq -P -i '.metadata.annotations = (
      strenv(ALLOWED_RESOURCES) | from_json | . += "kueue.konflux-ci.dev/requests-unallowed-resource" |
      map({"key": ., "value": "1"}) |
      from_entries)' "${unallowed_plr_filepath}"
}

main() {
  CONFIGMAP_FILENAME='deny-unallowed-annotations-configmap.yaml'

  # Per-cluster ConfigMap destinations.
  # Key: kueue ClusterQueue overlay dir
  # Value: cluster-specific policies-rd kueue-config dir containing the ConfigMap
  local -A configmap_configs=(
      ["components/kueue/rings/ring-1/stone-stage-p01"]="components/policies-rd/rings/ring-1/stone-stage-p01/kueue-config/"
      ["components/kueue/rings/ring-1/stone-stg-rh01"]="components/policies-rd/rings/ring-1/stone-stg-rh01/kueue-config/"
  )

  # Ring-level chainsaw test fixture destinations.
  # Each ring's shared .chainsaw-test/resources/ dir is updated from a single
  # representative cluster.  Use a separate map so the destinations are
  # independently configured and not inferred from the ConfigMap output path.
  # Key: representative kueue ClusterQueue overlay dir
  # Value: exact path to the ring's .chainsaw-test/resources/ dir
  local -A test_fixture_configs=(
      ["components/kueue/rings/ring-1/stone-stg-rh01"]="components/policies-rd/rings/ring-1/base/kueue/deny-unallowed-annotations/.chainsaw-test/resources/"
  )

  # Update per-cluster ConfigMaps
  for input_folder in "${!configmap_configs[@]}"; do
    local output_folder="${configmap_configs[$input_folder]%/}"
    local output_cm_file="${output_folder}/${CONFIGMAP_FILENAME}"
    echo "Generating ConfigMap: $input_folder -> $output_cm_file"
    update_configmap_from_clusterqueue "${input_folder}" "${output_cm_file}"
  done

  # Update ring-level chainsaw test fixtures
  for input_folder in "${!test_fixture_configs[@]}"; do
    local chainsaw_resources_dir="${test_fixture_configs[$input_folder]}"
    echo "Generating Test PipelineRuns: ${input_folder} -> ${chainsaw_resources_dir}"
    update_test_resources_from_clusterqueue "${input_folder}" "${chainsaw_resources_dir}"
  done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  (
      cd "$ROOT/../.."
      main
  )
fi
