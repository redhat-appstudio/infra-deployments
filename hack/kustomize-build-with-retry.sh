#!/bin/env bash

set -e -o pipefail

# Retry wrapper for flaky remote kustomize/helm fetches.
#
# Usage: kustomize-build-with-retry.sh <dir> [extra kustomize args...]
# Env:
#   KUSTOMIZE_RETRIES - max attempts (default 5)
#   OUTPUT=GITHUB     - emit GitHub Actions warnings on retry
#
# On success, kustomize stdout is written to this script's stdout (stderr is
# discarded). Callers that use -o rely on the output file; others can pipe
# stdout into yq/jq.

dir="${1:?directory required}"
shift

max_attempts="${KUSTOMIZE_RETRIES:-5}"
attempt=1
build_rc=0
kustomize_stdout=""
kustomize_stderr=""

while true; do
  kustomize_stderr=$(mktemp)
  set +e
  kustomize_stdout=$(kustomize build --enable-helm "${dir}" "$@" 2>"${kustomize_stderr}")
  build_rc=$?
  set -e

  if (( build_rc == 0 )); then
    rm -f "${kustomize_stderr}"
    printf '%s\n' "${kustomize_stdout}"
    exit 0
  fi

  if (( attempt >= max_attempts )); then
    printf "Error when running kustomize build for %s:\n" "${dir}" >&2
    cat "${kustomize_stderr}" >&2
    rm -f "${kustomize_stderr}"
    exit "${build_rc}"
  fi

  if [[ "${OUTPUT}" == "GITHUB" ]]; then
    printf "::warning::kustomize build for '%s' failed (attempt %s/%s), retrying in %ss...\n" \
      "${dir}" "${attempt}" "${max_attempts}" $(( attempt * 2 )) >&2
  else
    printf "kustomize build for '%s' failed (attempt %s/%s), retrying in %ss...\n" \
      "${dir}" "${attempt}" "${max_attempts}" $(( attempt * 2 )) >&2
  fi

  rm -f "${kustomize_stderr}"
  sleep $(( attempt * 2 ))
  attempt=$((attempt + 1))
done
