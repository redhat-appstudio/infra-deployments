#!/bin/bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
PIPELINE_NAME=$2   
if [ -z "$PIPELINE_NAME" ]
then
      PIPELINE_NAME=auto
fi 
# build repo pipelinename quay/internal
$SCRIPTDIR/build.sh $1 $PIPELINE_NAME quay
 