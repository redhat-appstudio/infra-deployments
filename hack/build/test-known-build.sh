#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )" 

# Run a set of builds which are known to work
# all can be deployed by deploy.sh 

$SCRIPTDIR/build.sh https://github.com/jduimovich/single-nodejs-app noop
$SCRIPTDIR/build.sh https://github.com/jduimovich/single-container-app
$SCRIPTDIR/build.sh https://github.com/jduimovich/single-nodejs-app
$SCRIPTDIR/build.sh https://github.com/jduimovich/spring-petclinic  java-builder


 