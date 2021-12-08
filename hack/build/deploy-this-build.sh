#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

BUILDNAME=$1 

echo
echo  "Deploying Results of  $BUILDNAME:"   
STATUS=$(oc get pr $BUILDNAME -o jsonpath="{.status.conditions[0].status}")
if [ $STATUS = "Unknown" ]; then 
      echo  "$BUILDNAME: is still running, cannot deploy this build"
      exit 0
fi
if [ $STATUS = "False" ]; then 
      echo  "$BUILDNAME: failed, cannot deploy this build."
      exit 0
fi    
IMAGE=$(oc get pr $BUILDNAME -o jsonpath="{.metadata.annotations.build\.appstudio\.openshift\.io/image}")
if [ -z "$IMAGE" ] 
then
      echo "Build $BUILDNAME: Not Ready "
      exit 1
fi 

DEPLOY=$(mktemp)  
oc get pr $BUILDNAME -o jsonpath="{.metadata.annotations.build\.appstudio\.openshift\.io/deploy}" |\
      base64 -d > $DEPLOY 

if [ -s $DEPLOY ]; then
      echo Using Deploy yaml from .build.appstudio.openshift.io/deploy 
else
      DEPLOY=""
fi
 
$SCRIPTDIR/deploy.sh $IMAGE $DEPLOY
rm -rf $DEPLOY
