#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

GITREPO=$1 
if [ -z "$GITREPO" ]
then
      echo Missing parameter Git URL to Build
      exit -1
else 
      echo Git URL to Build $GITREPO
fi

APPNAME=$(basename $GITREPO)
TAG=$(date +"%Y-%m-%d-%H%M%S") 
NS=$(oc config view --minify -o "jsonpath={..namespace}")
IMG=image-registry.openshift-image-registry.svc:5000/$NS/$APPNAME

echo "Building $GITREPO in build-$TAG"
echo "Project Namespace: " $NS
echo "Project Image: " $IMG

yq -M e ".spec.params[0].value=\"$GITREPO\"" $SCRIPTDIR/default-build.yaml  | \
  yq -M e ".spec.params[1].value=\"$IMG\"" - | \
  yq -M e ".metadata.name=\"build-$TAG\"" - | \
  oc apply -f -

