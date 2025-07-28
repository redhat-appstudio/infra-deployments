#!/bin/env bash

set -e -o pipefail

# inputs
declare OUTPUT
folder="${1}"

error_handler() {
  local exit_code=$?
  local command_line="${BASH_COMMAND}"
  local line_number="${BASH_LINENO[0]}"

  if [[ "${OUTPUT}" == "GITHUB" ]]; then
    printf "::error title=Script crashed on line %s::executing '%s' for folder '%s' with exit code %s.\n" "${line_number}" "${command_line}" "${folder}" "${exit_code}"
  else
    printf "Script crashed on line %s executing '%s' for folder '%s' with exit code %s.\n" "${line_number}" "${command_line}" "${folder}" "${exit_code}"
  fi

  exit "${exit_code}"
}

# Trap the ERR signal and call the error_handler function
trap error_handler ERR

# build manifests and filter ClusterPolicies for provided folder
manifests=$(kustomize build --enable-helm "${folder}" | yq -o=j)
policies=$(echo "${manifests}" | jq 'select(.kind=="ClusterPolicy")' | jq -s '.')

# calculate the number of policies in the given folder
num_policies=$(echo "${policies}" | jq 'length')

# print details of the identified ClusterPolicies
if [[ "${num_policies}" -gt 0 ]]; then
  names=$(echo "${policies}" | jq -c 'map(.metadata.name)')
  message="unallowed ClusterPolicies found"

  if [[ "${OUTPUT}" == "GITHUB" ]]; then
    fmt_policies="$(echo "${names}" | jq 'join(", ")')"
    printf "::error title=%s::violated at '%s' by policies '%s'.\n" "${message}" "${folder}" "${fmt_policies}"
  else
    jq -n --arg m "${message}" --arg p "${folder}" --argjson n "${names}" '{"message": $m, "path": $p, "policies": $n}'
  fi
  exit 1
fi

# nothing found, we are good
exit 0
