#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"


# Print the Pruning Configuration 
ISADMIN=$(oc whoami)
if [ "$ISADMIN" = "kubeadmin" ]; then 
KEEP=$( oc get TektonConfig config -n openshift-pipelines -o 'jsonpath={.spec.pruner.keep}')
echo
echo "PipelineRuns will be pruned to last $KEEP"
echo "Full Config:"
oc get TektonConfig config -n openshift-pipelines -o 'jsonpath={.spec.pruner}' | jq
fi 

# Run the pvc internal GC 
echo
$SCRIPTDIR/utils/cleanup-pvc.sh  
echo
if [ "$ISADMIN" = "kubeadmin" ]; then 
echo "Pruning Registry"
oc adm prune images --confirm --registry-url  default-route-openshift-image-registry.apps-crc.testing
fi 

