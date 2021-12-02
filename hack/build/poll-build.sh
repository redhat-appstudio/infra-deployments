#!/bin/bash

# Auto-detect changes for a repo and run the build for it when detected 
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
      echo "LOG $LOG"
      cat $LOG
      echo "---------------"
   else
      echo "BUILD $GITREPO PNAME $PIPELINE_NAME" 
      $SCRIPTDIR/build.sh "$GITREPO" $PIPELINE_NAME | tee $LOG
      LAST_BUILT=$CURRENT
      
      BUILDNAME=$(cat $LOG | grep "pipelinerun.tekton.dev" | cut -d '/' -f 2 | cut -d ' ' -f 1)
      echo  $BUILDNAME
      echo "^^^^^^"
      RESULT=$(oc get pr $BUILDNAME -o jsonpath="{.metadata.annotations.build\.appstudio\.io/repo}")
      echo  $RESULT
      echo "^^^^^^"
      echo "Waiting for Build:"
         while : ; do
         IMAGE=$(oc get pr $BUILDNAME -o jsonpath="{.metadata.annotations.build\.appstudio\.io/image}")
         if [ -z "$IMAGE" ] 
         then
               echo -n . 
               sleep 5
         else 
               echo Image  ready 
               $SCRIPTDIR/deploy.sh $IMAGE | tee -a $LOG
               break
         fi
         done
   fi
   sleep 5
done    
