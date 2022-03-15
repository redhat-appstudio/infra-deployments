#!/bin/bash
#
# Based on https://github.com/tektoncd/chains/blob/main/docs/tutorials/getting-started-tutorial.md
#
# Here only the taskrun is being verified.
#

source $(dirname $0)/_helpers.sh
set -ue

title "Suggested config for this demo:"
$SCRIPTDIR/config.sh simple --dry-run

title "Current config:"
$SCRIPTDIR/config.sh

pause

title "Run a simple task and watch its logs"
kubectl create -f \
  https://raw.githubusercontent.com/tektoncd/chains/main/examples/taskruns/task-output-image.yaml
tkn tr logs --follow --last

##
## The task produces a fake manifest json file with a digest that is
## visible to chains if oci storage is enabled. I don't understand
## how or why...
##

title "Wait a few seconds for chains finalizers to complete"
sleep 10

# This will show details about the taskrun and use cosign to verify it
$SCRIPTDIR/check-taskrun.sh
