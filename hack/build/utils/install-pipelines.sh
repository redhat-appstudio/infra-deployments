#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# install each pipeline found in the templates 
# Used to validate all pipelines can be installed.
# otherwise, use build.sh which will auto-install the pipeline computed for the repo
# deprecated when bundles are used
 oc get pipelines -n build-templates -o yaml | \
  yq e '.items[].metadata.name' - | \
  xargs -n 1 $SCRIPTDIR/install-single-pipeline.sh