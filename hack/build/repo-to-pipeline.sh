#!/bin/bash

# Peek the git repo to determine which pipeline to use
# This is a temp script until HAS Component Detection Query is completed
# Returns "noop" if no pipeline cannot be determined
# A noop pipeline will run but not produce an image or build
# Note if a devfile is present, the build will use a devfile builder

GITREPO=$1 
if [ -z "$GITREPO" ]
then
      echo Missing parameter Git URL to Build
      exit -1 
fi

# repo is $1, file is $2, pipeline name is $2
# echo the name and return from the script 
repo_marker_to_pipeline () {
  REPO=$1
  FILE=$2 
  PIPELINE=$3
  USER=$(echo $REPO | cut -d '/' -f 4)
  PROJECT=$(echo $REPO | cut -d '/' -f 5) 
  for BRANCH in main master
  do    
    URL="https://raw.githubusercontent.com/$USER/$PROJECT/$BRANCH/$FILE" 
    if curl  -H "Cache-Control: no-cache" --output /dev/null --silent --head --fail "$URL"; then
      # echo "Marker  $URL  exists"
      echo $PIPELINE
      exit 0;  
    fi 
  done 
}   

# need to understand dev files mapping 
repo_marker_to_pipeline $GITREPO "noop"          "noop"  
repo_marker_to_pipeline $GITREPO "devfile.yaml"  "devfile-build" 
repo_marker_to_pipeline $GITREPO "Dockerfile"    "docker-build" 
repo_marker_to_pipeline $GITREPO "package.json"  "nodejs-builder"
repo_marker_to_pipeline $GITREPO "pom.xml"       "java-builder" 
echo "noop" 