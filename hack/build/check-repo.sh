#!/bin/bash

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
if [ -z "$1" ]
    then
      echo "Need parameter with git repo url"
      exit -1 
fi
echo  "$1   -> $($SCRIPTDIR/repo-to-pipeline.sh $1)" 
 