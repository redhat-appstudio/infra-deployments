#! /bin/bash
#
# A wrapper for calling yamllint on yaml files in git repos
# This script uses the .yamllint file located in the same directory for
# configuration.

set +e

ROOT=$(git rev-parse --show-toplevel)

# Get the directory the script resides in
SCRIPT_DIR="$(
  cd -- "$(dirname "$0")" >/dev/null 2>&1
  pwd -P
)"

# Ensure we have the yamllint config
if ! command -v yamllint &>/dev/null; then
  echo "The yamllint command is not installed. Please install and try again."
  echo "See https://yamllint.readthedocs.io/en/stable/quickstart.html#installing-yamllint for details"
  exit 1
fi

# Ensure we have our .yamllint config file
if [[ ! -f $SCRIPT_DIR/.yamllint ]]; then
  echo "Local config file not found, using default ruleset"
fi

# If we have a path to check, set that our CHECK_PATH var
while getopts "hp:" OPTIONS; do
  case "${OPTIONS}" in
  p)
    CHECK_PATH="$ROOT/${OPTARG}"
    ;;
  h)
    echo "Example usage"
    echo ""
    echo "To validate all modified or untracked YAML files:"
    echo "  check-yaml.sh"
    echo ""
    echo "To validate a specific YAML file:"
    echo "  check-yaml.sh -p path/to/yaml/file"
    echo ""
    echo "For help:"
    echo "  check-yaml.sh -h"
    echo ""
    exit 0
    ;;
  *)
    echo "Unknown flag: $OPTARG"
    exit 1
    ;;
  esac
done

# Initialize our ERRORS var to false, indicating no failures to start
ERRORS=false

# A function to read all modified or untracked YAML files in the repo
# into an array which is used in the validation portion.
function get_yaml_files() {
  YAML_FILES=($(git ls-files --modified --other --full-name '*.yaml') )
  for i in "${!YAML_FILES[@]}"; do
    YAML_FILES[$i]="$ROOT/${YAML_FILES[$i]}"
  done
}

# A function to wrap the call of yamllint with our config file
function validate_yaml() {
  yamllint -c "$SCRIPT_DIR"/.yamllint "$1"
}

# If we have a specific path to check, we put that in the # YAML_FILES
# array. If we don't have a specific path, we call our get_yaml_files
# function which finds all modified or other state YAML files in our 
# repo and puts them in the YAML_FILES array
if [[ $CHECK_PATH ]]; then
  YAML_FILES=("$CHECK_PATH")
else
  get_yaml_files
fi

# Cycle through our YAML_FILES array, passing each to our validate_yaml
# function. if the return code of that call is 0, the file passed
# validation, otherwise it failed and we set our ERRORS var to true.
if [ ${#YAML_FILES[@]} != 0 ]; then
  for i in "${!YAML_FILES[@]}"; do
    if ! validate_yaml "${YAML_FILES[$i]}"; then
      ERRORS=true
    fi
  done
  # If we had errors, we indicate that we had YAML validation failures,
  # otherwise we report success, but note that warnings may have occured.
  if $ERRORS; then
    printf "\nYAML Validation failed\n"
  else
    printf "\nYAML validation passed, warnings may have occured.\n"
  fi
else
  echo "No YAML files to validate. Have a good day"
fi
