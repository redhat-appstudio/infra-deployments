#!/bin/bash

source $(dirname $0)/_helpers.sh
set -u

# Use a specific pipelinerun if provided, otherwise use the latest
PIPELINERUN_NAME=${1:-$( tkn pipelinerun describe --last -o name )}
PIPELINERUN_NAME=pipelinerun/$( trim-name $PIPELINERUN_NAME )

TASKRUN_NAMES=$(
  kubectl get $PIPELINERUN_NAME -o yaml | yq e '.status.taskRuns | keys | .[]' - )

for name in $TASKRUN_NAMES; do
  echo -n "$name 🔗 "
  $SCRIPTDIR/check-taskrun.sh $name --quiet
done
