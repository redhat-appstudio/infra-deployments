#!/bin/bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Preserve sanity while hacking
set -u

# Use a specific pipelinerun if provided, otherwise use the latest
PIPELINERUN_NAME=${1:-$( tkn pipelinerun describe --last -o name )}
PIPELINERUN_NAME=pipelinerun/$( echo $PIPELINERUN_NAME | sed 's#.*/##' )

TASKRUN_NAMES=$(
  kubectl get $PIPELINERUN_NAME -o yaml | yq e '.status.taskRuns | keys | .[]' - )

for name in $TASKRUN_NAMES; do
  echo -n "$name 🔗 "
  $SCRIPTDIR/check-taskrun.sh $name --quiet
done
