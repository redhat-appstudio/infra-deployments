#!/bin/bash
#
# A wrapper for hack/build/build.sh to help exercise
# chains for an example S2I nodejs pipeline build
#
# Example usage:
#  ./default-pipelines-demo.sh
#  ./default-pipelines-demo.sh https://github.com/someuser/some-app
#  NO_FORCE_REBUILD=1 ./default-pipelines-demo.sh https://github.com/someuser/some-app
#

source $(dirname $0)/_helpers.sh
set -ue

APP_URL=${1:-https://github.com/simonbaird/single-nodejs-app}
APP_NAME=$( basename $APP_URL )

# Set NO_FORCE_REBUILD if you don't want to nuke the imagestream before building
if [[ -z ${NO_FORCE_REBUILD:-} ]]; then
  title "Delete previous builds to force a rebuild"
  oc delete is $APP_NAME --ignore-not-found=true
fi

title "Run the build pipeline and wait for it to complete"
$ROOT/hack/build/build.sh $APP_URL

PR_NAME=$( tkn pr describe --last -o yaml | yq e .metadata.name - )

show-tasks() {
  tkn pr describe $PR_NAME | grep "^ $PR_NAME" --color=never
}

until oc wait --for=condition=Succeeded --timeout=15s pr $PR_NAME >/dev/null 2>&1 ; do
  show-tasks
  say "Waiting..."
done
show-tasks
say "Build $PR_NAME complete"

# A few extra seconds for chains finalizers to finish up
sleep 5

# Todo: the image is pushed to the internal repo so we would need
# to run cosign inside the cluster to verify the image.

# For now we'll just show some annotations
$SCRIPTDIR/inspect-pipeline-run.sh $PR_NAME
