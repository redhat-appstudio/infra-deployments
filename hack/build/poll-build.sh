#!/bin/bash

# Auto-detect changes for a repo and run the build-deploy for it when detected 
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

GITREPO=$1 
if [ -z "$GITREPO" ]
then
      echo Missing parameter Git URL to Build
      exit -1 
fi
PIPELINE_NAME=$2    
LOG=$(mktemp)
echo "No Builds Yet" >$LOG
LAST_BUILT=$(git ls-remote  $GITREPO HEAD |  cut -f 1) 
while : ; do 
  clear 
  CURRENT=$(git ls-remote  $GITREPO HEAD |  cut -f 1)
  echo "$GITREPO"
  echo "Last $LAST_BUILT current $CURRENT"
   if [ $LAST_BUILT = "$CURRENT" ]; then 
      echo Poll Nothing to Build $(date +"%Y-%m-%d-%H%M%S")
      echo 
      echo "Last Build Triggered:"
      echo "---------------" 
      cat $LOG
      echo "---------------"
   else
      echo "Build and Deploy Repo" 
      $SCRIPTDIR/build-deploy.sh "$GITREPO" $PIPELINE_NAME  | tee $LOG
   fi
   sleep 10
done    
