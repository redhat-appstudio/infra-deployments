#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# This script represents the maintainance loop for build pipelines
# Periodically scrub builds, keeping at most 10

while true
do 
$SCRIPTDIR/prune-builds.sh 8 
sleep 60
done