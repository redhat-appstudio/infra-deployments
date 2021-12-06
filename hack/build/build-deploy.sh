#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
  
LOG=$(mktemp)
$SCRIPTDIR/build.sh $1 $2| tee $LOG 
BUILDNAME=$(cat $LOG | grep "pipelinerun.tekton.dev" | cut -d '/' -f 2 | cut -d ' ' -f 1)
echo  
echo  "Waiting for build  $BUILDNAME:"  
while : ; do 
     # oc get pr $BUILDNAME -o jsonpath='{.status.conditions[0]}' | jq  
      STATUS=$(oc get pr $BUILDNAME -o jsonpath='{.status.conditions[0].status}')  
      if [ "$STATUS" = "False" ]; then 
            echo "Build Error Occurred  "
            $SCRIPTDIR/ls-build.sh $BUILDNAME 
            break;
      fi  
      if [ "$STATUS" = "True" ]; then 
            echo "Build Success"
            IMAGE=$(oc get pr $BUILDNAME -o jsonpath="{.metadata.annotations.build\.appstudio\.openshift\.io/image}")
            if [ -z "$IMAGE" ] 
            then
                  echo "Error missing image "
                  exit 1
            else 
                  echo "Build Completed" 
                  $SCRIPTDIR/ls-build.sh $BUILDNAME 
                  $SCRIPTDIR/deploy-this-build.sh $BUILDNAME
                  exit 1
            fi
      else 
            $SCRIPTDIR/ls-build.sh $BUILDNAME 
            echo -n .
            sleep 5
      fi
done