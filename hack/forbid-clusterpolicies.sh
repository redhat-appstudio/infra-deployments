#!/bin/env bash

set -e -o pipefail

# build manifests and filter ClusterPolicies for provided folder
folder="${1}"
manifests=$(kustomize build --enable-helm "${folder}" | yq -o=j)
policies=$(echo "${manifests}"| jq 'select(.kind=="ClusterPolicy")' | jq -s '.')

# calculate the number of policies in the given folder
num_policies=$(echo "${policies}" | jq 'length')

# print details of the identified ClusterPolicies
if [[ "${num_policies}" -gt 0 ]]; then
  names=$(echo "${policies}" | jq -c 'map(.metadata.name)')
  message="unallowed ClusterPolicies found"
  jq -n --arg m "${message}" --arg p "${folder}" --argjson n "${names}" '{"message": $m, "path": $p, "policies": $n}'
  exit 1
fi

# nothing found, we are good
exit 0
