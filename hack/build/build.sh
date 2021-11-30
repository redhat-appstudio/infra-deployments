#!/bin/bash

# This script is representative of how A HAS Application will launch a pipelinerun
# to perform a build. 
# See default-build.yaml for the sample pipeline run yaml.
# To launch a build HAS will need to pass parameters
# - spec.params[GITURL] the URL for REPO
# - pspec.params[IMAGE] for the IMAGE
# - add bindings for the SPI to ensure access to the GITURL and the IMAGE registry
# This script installs pipelines needed for the default build but this mechanism will change
# The type of build is currently fixed at docker
# An imported HAS Application will detect language and build type and map to a 
# pipeline name which will be provided by the Build Service

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

GITREPO=$1 
if [ -z "$GITREPO" ]
then
      echo Missing parameter Git URL to Build
      exit -1 
fi
PIPELINE_NAME=$2   
if [ -z "$PIPELINE_NAME" ]
then
      PIPELINE_NAME=$($SCRIPTDIR/repo-to-pipeline.sh $GITREPO) 
      if [ "$PIPELINE_NAME" == "none" ]
      then 
            echo Pipeline name  $PIPELINE_NAME - exiting build 
            exit 0
      else 
            echo Pipeline name computed from git repo $PIPELINE_NAME
      fi
fi 

PIPELINE_RUN=$SCRIPTDIR/default-build.yaml  
#PIPELINE_NAME=$(yq e '.spec.pipelineRef.name' $PIPELINE_RUN) 

$SCRIPTDIR/install-single-pipeline.sh $PIPELINE_NAME

oc get pipeline  $PIPELINE_NAME > /dev/null 2>&1
ERR=$? 
if (( $ERR != 0 )); then
  echo Error Pipeline $PIPELINE_NAME Missing - exiting 
  exit -1
fi 


APPNAME=$(basename $GITREPO) 
IMAGE_FULL_TAG=$(git ls-remote $GITREPO HEAD)
IMAGE_SHORT_TAG=${IMAGE_FULL_TAG:position:7}
BUILD_TAG=$(date +"%Y-%m-%d-%H%M%S") 
NS=$(oc config view --minify -o "jsonpath={..namespace}")
 
IMG=image-registry.openshift-image-registry.svc:5000/$NS/$APPNAME:$IMAGE_SHORT_TAG
echo
echo "Building $GITREPO"
echo "Build Name: build-$BUILD_TAG"
echo "Namespace: " $NS
echo "Image: " $IMG
echo "Pipeline: " $PIPELINE_NAME 

yq -M e ".spec.params[0].value=\"$GITREPO\"" $PIPELINE_RUN | \
  yq -M e ".spec.params[1].value=\"$IMG\"" - | \
  yq -M e ".metadata.name=\"$PIPELINE_NAME-$BUILD_TAG\"" - | \
  yq -M e ".spec.pipelineRef.name=\"$PIPELINE_NAME\"" - | \
  yq -M e ".spec.workspaces[0].subPath=\"$PIPELINE_NAME-$BUILD_TAG\"" - | \
  oc apply -f -

echo
