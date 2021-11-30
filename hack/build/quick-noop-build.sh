#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )" 
# this does a quick noop build
$SCRIPTDIR/build.sh https://github.com/jduimovich/single-container-app noop
 