#!/bin/bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

PIPELINERUN_NAME=$1

# Preserve sanity while hacking
set -ue

if [[ -z $PIPELINERUN_NAME ]]; then
  # Use the most recently created pipelinerun
  # (Fixme: Would be better to exclude running pipelines)
  PIPELINERUN_NAME=$(
    kubectl get pipelinerun -o name --sort-by=.metadata.creationTimestamp |
      tail -1 | cut -d/ -f2 )
fi

TASKRUN_NAMES=$(
  kubectl get pipelinerun/$PIPELINERUN_NAME -o yaml |
    yq e '.status.taskRuns | keys | .[]' - )

for name in $TASKRUN_NAMES; do
  echo -n "$name 🔗 "
  $SCRIPTDIR/check-taskrun.sh $name --quiet
done
