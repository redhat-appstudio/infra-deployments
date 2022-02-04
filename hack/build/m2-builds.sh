#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )" 

# demo of using the build service to create a build for a suite of devfile repos
# Note: deploy is not part of the build service, it will be performed via Gitops service
# This demo uses a direct deploy to demonstrate the build works. 

DEPLOY=$1
if [ -z "$DEPLOY" ]
then
      echo "Building all Milestone 2 Repos. add '-deploy' to commandline to auto deploy"
else
    if [ $DEPLOY = "-deploy" ]; then 
        echo "Building and Deploying all Milestone 2 Repos." 
    else 
        echo "$DEPLOY invalid, ignored."
    fi 
fi

CMD=$SCRIPTDIR/build$DEPLOY.sh

if [ -z "${MY_GITHUB_USER}" ]; then
    MY_GITHUB_USER=$(git ls-remote --get-url | sed 's|^git@github.com:|https://github.com/|' | cut -d/ -f4)
fi


# devfile based builds 
$CMD https://github.com/devfile-samples/devfile-sample-java-springboot-basic
$CMD https://github.com/nodeshift-starters/devfile-sample
$CMD https://github.com/devfile-samples/devfile-sample-code-with-quarkus
$CMD https://github.com/devfile-samples/devfile-sample-python-basic 

# auto-detect base builds
$CMD https://github.com/${MY_GITHUB_USER}/single-container-app
$CMD https://github.com/${MY_GITHUB_USER}/single-nodejs-app
$CMD https://github.com/${MY_GITHUB_USER}/spring-petclinic  java-builder
 
echo "Run this to show running builds $SCRIPTDIR/ls-builds.sh"
