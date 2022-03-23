#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

#-----------------------------------------------------------
#
# For a particular pipeline run, download pipeline run data from
# multiple sources and lay it out on disk so it can be referenced
# in rego policies by opa.
#
# (WIP, experimental)
#
#-----------------------------------------------------------

DEFAULT_PR_NAME=$( tkn pr describe --last -o name )
PR_NAME=${1:-$DEFAULT_PR_NAME}
PR_NAME=$( echo $PR_NAME | sed 's#.*/##' )

DEFAULT_DATA_DIR=$(dirname $0)/data
DATA_DIR=${2:-$DEFAULT_DATA_DIR}

#-----------------------------------------------------------

pr-get-tr-names() {
  local pr=$1
  kubectl get pr/$pr -o json | jq -r '.status.taskRuns|keys|.[]'
}

tr-get-annotation() {
  local tr=$1
  local key=$2

  # escape dots
  key=$( echo "$key" | sed 's/\./\\\./g' )

  kubectl get tr/$tr -o jsonpath="{.metadata.annotations.$key}"
}

tr-get-result() {
  local tr=$1
  local key=$2
  kubectl get tr/$tr -o jsonpath="{.status.taskResults[?(@.name == \"$key\")].value}"
}

data-file() {
  local category=$1
  local kind=$2
  local name=$3

  local dir="$DATA_DIR/$category/$kind"
  local file="$dir/$name.yaml"

  mkdir -p $dir

  # Better not silently overwrite data
  [[ -f $file ]] && echo "Name clash for $file!" && exit 1

  echo $file
}

yq-inject-top-level-key() {
  local key="\"$1\""
  local no_quotes=${2:-}
  [[ -n $no_quotes ]] && key="$1"
  yq -P e "{ $key: . }" -
}

k8s-save-data() {
  local kind=$1
  local name=$2
  local name_space=${3:-}

  local name_space_opt=
  [[ -n $name_space ]] && name_space_opt="-n$name_space"

  local file=$( data-file k8s $kind $name )

  kubectl get \
    $name_space_opt $kind $name -o yaml |
      yq-inject-top-level-key $name > $file
}

rekor-entry-save-data() {
  local transparency_url=$1

  # Assume it ends with something like '?logIndex=1234'
  local log_index=$( echo "$transparency_url" | cut -d= -f2 )
  # ...and starts with something like 'https://host.tld/'
  local rekor_host=$( echo "$transparency_url" | cut -d/ -f3 )

  local file=$( data-file rekor $rekor_host/logIndex $log_index )

  rekor-cli get \
    --log-index $log_index --rekor_server "https://$rekor_host" --format json |
      yq-inject-top-level-key $log_index no_quotes > $file
}

rekor-digest-save-data() {
  local digest=$1
  local transparency_url=$2
  local rekor_host=$( echo "$transparency_url" | cut -d/ -f3 )

  local file=$( data-file rekor "$rekor_host/sha" "$digest" )

  local uuids=$( rekor-cli search --sha "$digest" --rekor_server "https://$rekor_host" 2>/dev/null )
  for uuid in $uuids; do
    # Could be multiple so append to the file
    rekor-cli get \
      --uuid $uuid --rekor_server "https://$rekor_host" --format json |
        yq-inject-top-level-key $uuid >> $file

  done
}

save-all-for-taskrun() {
  local tr=$1

  # The k8s object
  k8s-save-data TaskRun $tr

  # The transparency log entry found in the task
  local transparency_url=$( tr-get-annotation $tr 'chains.tekton.dev/transparency' )
  [[ -n $transparency_url ]] && rekor-entry-save-data "$transparency_url"

  # The transparency log entry for the image digest
  local image_digest=$( tr-get-result $tr IMAGE_DIGEST | cut -d: -f2 )
  # transparency_url is needed here only to find the rekor server
  [[ -n $image_digest ]] && rekor-digest-save-data "$image_digest" "$transparency_url"

  # Todo:
  # - What about tekton results?
  # - What about image data from the registry?
  # - Should we extract the attestations and bodies for easier access?

  # set -e is not all fun and games.. :)
  true
}

# Clean out old data
rm -rf $DATA_DIR

# Save new data
k8s-save-data ConfigMap chains-config tekton-chains
k8s-save-data PipelineRun $PR_NAME
for tr in $( pr-get-tr-names $PR_NAME ); do
  save-all-for-taskrun $tr
done

# Show what we created
find data -name '*.yaml'
