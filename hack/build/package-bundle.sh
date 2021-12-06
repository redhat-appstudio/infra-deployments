#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$SCRIPTDIR/../..
TEMPLATES=$ROOT/components/build/build-templates 

BUNDLE=quay.io/redhat-appstudio/build-templates-bundle:v0.1

echo 
echo "Building $BUNDLE"
echo "Change this bundle name to your own for dev testing"
echo 

echo "This is packaging all pipelines in build templates into bundle"
echo "The namespace field is being removed."
PARAMS=""
BUILDDIR=$(mktemp -d) 
for i in $TEMPLATES/*.yaml ; do
    KIND=$(yq e '.kind' $i) 
    if [ $KIND = "Pipeline" ]; then 
        echo "Found Pipeline in: $i"
        filtered=$BUILDDIR/$(basename $i) 
        yq e 'del(.metadata.namespace)' $i > $filtered
        PARAMS="$PARAMS -f $filtered "
    fi 
done 
tkn bundle push $BUNDLE $PARAMS  
echo  
 


  