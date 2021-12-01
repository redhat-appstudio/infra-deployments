#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$SCRIPTDIR/../..
TEMPLATES=$ROOT/components/build/build-templates 

echo "package all pipelines in build templates into bundle"
echo "The namespace field needs to be removed"
PARAMS=""
mkdir $SCRIPTDIR/tmp 
for i in $TEMPLATES/*.yaml ; do
    KIND=$(yq e '.kind' $i) 
    if [ $KIND = "Pipeline" ]; then 
        echo "Found Pipeline in: $i"
        filtered=$SCRIPTDIR/tmp/$(basename $i) 
        yq e 'del(.metadata.namespace)' $i > $filtered
        PARAMS="$PARAMS -f $filtered "
    fi 
done
tkn bundle push quay.io/jduimovich0/bundle:appstudio $PARAMS
rm -rf $SCRIPTDIR/tmp 