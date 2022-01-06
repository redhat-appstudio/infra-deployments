#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# install the a local namespace bundle to contain oci bundle.

CM=$(mktemp)
cat > $CM <<OCILOCATION
apiVersion: v1
kind: ConfigMap
metadata:
  name: build-pipelines-defaults 
data: 
  default_build_bundle: "Your Bundle Here" 
OCILOCATION

yq -M e ".data.default_build_bundle=\"$1\"" $CM | oc apply -f -

