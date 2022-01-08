#!/bin/bash

# This script demonstrates of how a client of the Build Service 
# eg HAS Application can launch a Build
# This Build launch conforms the Build Contract from the Build Service
# see the spec <link> for more details
# 
# See templates/default-build.yaml for the sample build run


# To launch a build a client will need to pass parameters
# - spec.params[GITURL] the URL for REPO
# - spec.params[IMAGE] for the IMAGE
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

PIPELINE_RUN=$SCRIPTDIR/templates/default-build-bundle.yaml   

$SCRIPTDIR/utils/install-pvc.sh $PIPELINE_NAME
$SCRIPTDIR/utils/install-secrets.sh  

APPNAME=$(basename $GITREPO) 
IMAGE_FULL_TAG=$(git ls-remote $GITREPO HEAD)
IMAGE_SHORT_TAG=${IMAGE_FULL_TAG:position:7}
BUILD_TAG=$(date +"%Y-%m-%d-%H%M%S") 
NS=$(oc config view --minify -o "jsonpath={..namespace}")

# local bundle overide for dev purposes 
BUNDLE=$(oc get cm build-pipelines-defaults -o=jsonpath='{.data.default_build_bundle}' 2> /dev/null)
if [ -z "$BUNDLE" ]
then
      BUNDLE=$(oc get cm -n build-templates build-pipelines-defaults -o=jsonpath='{.data.default_build_bundle}')
      if [ -z "$BUNDLE" ]
      then
            BUNDLE=$(yq -M e ".spec.pipelineRef.bundle"  $PIPELINE_RUN)
            echo "Warning missing bundle name configmap in current namespace and build-templates, using default" 
      fi 
fi 

oc get secret quay-registry-secret  2> /dev/null  > /dev/null 
ERR=$?  
if (( $ERR == 0 )); then    
   echo "Using Secret: quay-registry-secret"
   export QUAY_PATCH="keep-secrets" 
else  
   echo "Warning - No Registry Secrets installed, only internal registry repos will work."
   export  QUAY_PATCH="registry-auth" 
fi 
oc get secret git-repo-secret   2> /dev/null  > /dev/null 
ERR=$?  
if (( $ERR == 0 )); then
   echo "Using Secret: git-repo-secret"
   export GIT_PATCH="keep-secrets" 
else 
   echo "Warning - No Git Secrets installed, only public git repos will work."
   export  GIT_PATCH="git-auth" 
fi   
 
IMG=quay.io/$MY_QUAY_USER/$APPNAME:$IMAGE_SHORT_TAG
echo
echo "Building $GITREPO"
echo "Build Name: build-$BUILD_TAG"
echo "Namespace: " $NS
echo "Image: " $IMG
echo "Bundle: " $BUNDLE
echo "Pipeline: " $PIPELINE_NAME  

PATCHQ=$(printf "del(.spec.workspaces[] | select (.name == \"%q\"))" "$QUAY_PATCH") 
PATCHR=$(printf "del(.spec.workspaces[] | select (.name == \"%q\"))" "$GIT_PATCH") 

APPLY=$(mktemp)
yq -M e ".spec.params[0].value=\"$GITREPO\"" $PIPELINE_RUN | \
  yq -M e ".spec.params[1].value=\"$IMG\"" - | \
  yq -M e ".metadata.name=\"$PIPELINE_NAME-$BUILD_TAG\"" - | \
  yq -M e ".spec.pipelineRef.name=\"$PIPELINE_NAME\"" - | \
  yq -M e ".spec.pipelineRef.bundle=\"$BUNDLE\"" - | \
  yq -M e ".spec.workspaces[0].subPath=\"pv-$PIPELINE_NAME-$BUILD_TAG\"" - | \
  yq -M e "$PATCHR" - | yq -M e "$PATCHQ" - > $APPLY

#cat $APPLY 
oc apply -f $APPLY
 
