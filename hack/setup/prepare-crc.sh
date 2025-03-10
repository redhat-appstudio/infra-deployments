#!/bin/bash
#This script configures CRC for running app-studio in it
#It cofigures the CRC to have better memory and CPUs 
#It also configures CRC to have required addons.
#It takes few parameters
#    --delete-cluster : if this is set CRC cluster is automatically deleted before start
#    --memory, -m: sets memory allowance(MB) for CRC, default is 16384 MB
#    --cpu, -c: sets CPU allowance for CRC, default is 6
#    --help, -h: Print the help with options

DELETE_CLUSTER=0
MEMORY=16384
CPUS=6
ROOT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
#echo $ROOT_DIR
CRCBINARY=$(readlink -f ~/.crc/bin/crc)
if [[ -z "$CRCBINARY" ]]; then
  CRCBINARY="crc"
fi
#echo $CRCBINARY

while [ True ]; do
if [ "$1" = "--delete-cluster" ];then
    DELETE_CLUSTER=1
    shift 1
elif [ "$1" = "--memory" -o "$1" = "-m" ]; then
    MEMORY=$2
    shift 2
elif [ "$1" = "--cpu" -o "$1" = "-c" ]; then
    CPUS=$2
    shift 2
elif [ "$1" = "--help" -o "$1" = "-h" ]; then
    echo "
    Usage: $0 [OPTIONS]
    --delete-cluster : if this is set CRC cluster is automatically deleted before start
    --memory, -m: sets memory allowance(MB) for CRC, default is 16384 MB
    --cpu, -c: sets CPU allowance for CRC, default is 6
    --help, -h: Print the help with options
    "
    shift 1
    exit 0
else
    break
fi
done

#Get CRC VMs configured in local machine
"$CRCBINARY" setup
#Set memory and CPU allowance for CRC
echo "Memory allowance for CRC is set to $MEMORY MB"
echo "CPU allowance for CRC is set to $CPUS"
"$CRCBINARY" config set memory $MEMORY
"$CRCBINARY" config set cpus $CPUS
#enable netmetrices addons, to make sure member cluster has proper resources
"$CRCBINARY" config set enable-cluster-monitoring true
#Delete existing CRC cluster to apply the config updates
if [ $DELETE_CLUSTER -eq 1 ]; then
    echo "Force delete for CRC is set, deleting the existing CRC cluster"
    "$CRCBINARY" delete --force
else
    "$CRCBINARY" delete
fi

#TODO: Check the return value of delete command and go forward accordingly
#And inform the users for the next steps in case user choose not to delete the 
#existing cluster

#Start CRC with modified configs
"$CRCBINARY" start

#Point local environment clients (kubectl and oc) to the CRC cluster
eval $("$CRCBINARY" oc-env)
kubectl config use-context crc-admin

# Label CRC node with `topology.kubernetes.io/zone` so some topologySpreadConstraints
# do not complain about the label not existing (for example pipeline-service)
kubectl label nodes crc topology.kubernetes.io/zone=crc

#Reduce cpu resource request for each AppStudio Application
#TODO: Check when to run the reduce gitops cpu requests
#TODO: $ROOT_DIR/../../hack/reduce-gitops-cpu-requests.sh
