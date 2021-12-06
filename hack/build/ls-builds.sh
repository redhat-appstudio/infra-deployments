
#!/bin/bash 

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

oc get pipelineruns --no-headers -o custom-columns=":metadata.name" | \
xargs -n 1 -I {}  $SCRIPTDIR/ls-build.sh {}