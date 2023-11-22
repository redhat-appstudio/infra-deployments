#!/usr/bin/env bash

#quit if exit status of any cmd is a non-zero value
set -o errexit
set -o nounset
set -o pipefail

usage() {
    echo "
Usage: 
    ${0##*/} ./update-argocd-app-revision.sh [options]
Replace the argocd application revision.
Mandatory arguments:
    --branch
        Pull request branch
    --filepath
        path to the file which has contains applicationset e.g pipeline-service: pipeline-service-stone-stg-m01
    
Optional arguments:
    -h, --help
        Display this message.
Example:
    ${0##*/} ./update-argocd-app-revision.sh --branch test-branch
" >&2

}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
    -b | --branch)
      shift
      export BRANCH="$1"
      ;;
    -p | --filepath)
      shift
      export FILEPATH="$1"
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 1
      ;;
    esac
      shift
  done
}

prechecks() {
    if [[ -z "$BRANCH" ]]; then
      printf "PR Branch is not set\n\n"
      usage
      exit 1
    fi
    if [[ -z "$FILEPATH" ]]; then
      printf "path of applicationset is not set\n\n"
      usage
      exit 1
    fi
}

main() {
    parse_args "$@"
    prechecks

    echo "Disable auto sync for all-application-sets"
    if ! oc patch applications.argoproj.io/all-application-sets -n openshift-gitops --type json --patch='[ { "op": "remove", "path": "/spec/syncPolicy/automated" } ]'; then
      echo "Failed to disable auto sync."
      exit 1
    fi
    # Read each line from the component file, which has all the application details from sync app task e.g {pipeline-service: pipeline-service-stone-stg-m01}
    while IFS= read -r line
    do
    component="$line"
    component=$(echo -e "$component" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
    
    # Set the delimiter to separate component-app and appset name
    IFS=":"
    read -ra app <<< "$component"
    echo "Replace revision for changed components ${app[0]} application set"
    if ! oc patch applicationsets.argoproj.io/${app[1]} -n openshift-gitops --type json --patch="[ { \"op\": \"replace\", \"path\": \"/spec/template/spec/source/targetRevision\", \"value\": \"$BRANCH\" } ]"; then
      echo "Error: Failed to replace revision for ${app[0]} application set."
      exit 1
    fi

    done < "$FILEPATH"
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
  main "$@"
fi
