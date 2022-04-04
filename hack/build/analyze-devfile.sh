#!/usr/bin/env bash

fetch_file () { 
  REPO=$1 
  FILE=$2
  OUT=$3
  USER=$(echo $REPO | cut -d '/' -f 4)
  PROJECT=$(echo $REPO | cut -d '/' -f 5)
  for BRANCH in main master
  do   
    URL="https://raw.githubusercontent.com/$USER/$PROJECT/$BRANCH/$FILE" 
    if curl  -H "Cache-Control: no-cache" --output $OUT --silent   --fail "$URL"; then
        return;  
    fi 
  done  
}   

DEVFILE=$(mktemp)
fetch_file $1 devfile.yaml $DEVFILE

OLOOP_BUILD=$(mktemp)
OLOOP_DEPLOY=$(mktemp)
yq e '.components | with_entries(select(.[].name=="outerloop-build"))' $DEVFILE |  yq e '.[]' - > $OLOOP_BUILD
yq e '.components | with_entries(select(.[].name=="outerloop-deploy"))' $DEVFILE | yq e '.[]' - > $OLOOP_DEPLOY

DEV_DOCKERFILE="$(yq e '.image.dockerfile.uri' $OLOOP_BUILD)"
DEV_DIRECTORY="$(yq e '.image.dockerfile.buildContext' $OLOOP_BUILD)"
DEV_DEPLOY_FILE="$(yq e '.kubernetes.uri' $OLOOP_DEPLOY)"

if [ -z "$DEV_DEPLOY_FILE" ] 
then
    echo "No deployment yaml in devfile, will use internal deployent yaml" 
else 
    DEPLOYFILE=$(mktemp)
    fetch_file $1 $DEV_DEPLOY_FILE $DEPLOYFILE  
    DEV_DEPLOY=$(cat $DEPLOYFILE | base64 -w 0)   
fi 
echo "Devfile Analysis:"
echo "Dockerfile: $DEV_DOCKERFILE"  
echo "BuildContext: $DEV_DIRECTORY"   
echo "Deploy: $DEV_DEPLOY_FILE" 

  
  