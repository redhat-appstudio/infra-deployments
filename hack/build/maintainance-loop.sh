#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# This script is representative of some of the maintainance needed on a workspace/pipelines
# the PVC is shared, so the cleanup-pvc will rm each directory which has no active pipeline runes
# the PipelineRuns pile up, so prune-pipelines will keep it trim
 

while true
do 
$SCRIPTDIR/prune-pipelines 20 
$SCRIPTDIR/cleanup-pvc.sh 
sleep 60
done