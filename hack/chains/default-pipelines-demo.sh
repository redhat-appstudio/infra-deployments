#!/bin/bash

source $(dirname $0)/_helpers.sh
set -ue

APP_URL=${1:-https://github.com/simonbaird/single-nodejs-app}

# Assume you have the repo checked out locally in this location
# with origin pointing at your fork and with main as the default branch
# (Use awk to extract the most likely directory name from the url)
#
APP_LOCAL_DIR=$ROOT/../$( echo "$APP_URL" | awk -F/ '{print $NF}' )

# The pipeline will skip the build if the sha was already built
# so make sure we have a fresh sha.
#
title "Push a fresh sha to ensure a rebuild occurs"
( cd $APP_LOCAL_DIR && git commit --amend --no-edit && git push -f origin main )

# Make sure we have the chains hinting in the cluster tasks
# (Remove later when build-pipelines-defaults is updated in components/build/kustomization)
#
title "Install latest pipeline bundle"
$ROOT/hack/build/utils/install-pipelines.sh \
  quay.io/redhat-appstudio/build-templates-bundle:b6d060ca46c0976251a8af6cc41c7dcb39d28da0

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
