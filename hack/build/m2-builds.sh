#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )" 

# demo of using the build service to create a build for a suite of devfile repos
# Note: deploy is not part of the build service, it will be performed via Gitops service
# This demo uses a direct deploy to demonstrate the build works. 

echo "Building all Milestone 2 Repos. add '-deploy' to commandline to auto deploy"
echo "Building all Milestone 2 Repos. add 'quay' to commandline to use quay repo"

for PARAM in "$@"
do  
    if [ $PARAM = "-deploy" ]; then 
        DEPLOY=-deploy
    fi  
    if [ $PARAM = "quay" ]; then 
        REPO=quay
    fi    
done

CMD=$SCRIPTDIR/build$DEPLOY.sh 

# devfile based builds 
$CMD https://github.com/devfile-samples/devfile-sample-java-springboot-basic auto $REPO
$CMD https://github.com/nodeshift-starters/devfile-sample auto $REPO
$CMD https://github.com/devfile-samples/devfile-sample-code-with-quarkus auto $REPO
$CMD https://github.com/devfile-samples/devfile-sample-python-basic auto $REPO

# auto-detect base builds
$CMD https://github.com/jduimovich/single-container-app auto $REPO
$CMD https://github.com/jduimovich/single-nodejs-app auto $REPO
$CMD https://github.com/jduimovich/spring-petclinic  java-builder $REPO
 
echo "Run this to show running builds $SCRIPTDIR/ls-builds.sh"