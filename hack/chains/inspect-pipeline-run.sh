#!/bin/bash
#
# Take a look at taskrun annotations for a particular pipelinerun
# Useful to help debug and troubleshoot chains
#

source $(dirname $0)/_helpers.sh
set -u

# Use a specific pipelinerun if provided, otherwise use the latest
PR_NAME=${1:-$( tkn pipelinerun describe --last -o name )}
PR_NAME=pr/$( trim-name $PR_NAME )

TR_NAMES=$(
  kubectl get $PR_NAME -o yaml | yq e '.status.taskRuns | keys | .[]' - )

for tr in $TR_NAMES; do
  title $tr
  kubectl get tr/$tr -o yaml | yq e '.metadata.annotations' -
done
