#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"/..

# Print help message
function print_help() {
  echo "Usage: $0 [-c|--component SUBDIR] [-h|--help]"
  echo "  -c, --component SUBDIR    Path to a directory. Defaults to the current directory."
  echo "  -h, --help                Help message"
  echo
  echo "Example usage: \`$0 -c components/pipeline-service/production"
}

function parse_args() {
  COMPONENT="."
  while [[ $# -gt 0 ]]; do
    key=$1
    case $key in
    --component | -c)
      shift
      COMPONENT="$1"
      ;;
    -h | --help)
      print_help
      exit 0
      ;;
    *)
      echo "Unknown argument: $key" >&2
      exit 1
      ;;
    esac
    shift
  done
}

function main() {
  parse_args "$@"

  for DIR in $(find "$COMPONENT" -name resources); do
    TARGET=$(dirname "$DIR")/deploy.yaml
    echo "$DIR: $TARGET"
    kustomize build "$DIR" >"$TARGET"
  done
}

main "$@"
