#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

KEEP=$1 
if [ -z "$KEEP" ] 
then
      KEEP=5  
fi  
tkn pr delete --keep $KEEP -f
$SCRIPTDIR/utils/cleanup-pvc.sh  
ISADMIN=$(oc whoami)
if [ $ISADMIN = "kubeadmin" ]; then 
oc adm prune images --confirm --registry-url  default-route-openshift-image-registry.apps-crc.testing
fi 

