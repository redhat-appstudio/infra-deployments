#!/bin/bash
#This script configures CRC for running app-studio in it
#It cofigures the CRC to have better memory and CPUs 
#It also configures CRC to have required addons.

#TODO: Check for CRC installation and binary location and set it to CRC-BINARY
CRCBINARY=$(readlink -f ~/.crc/bin/crc)

echo $CRCBINARY
#Get CRC VMs configured in local machine
$CRCBINARY setup
#Set memory and CPU allowance for CRC
$CRCBINARY config set memory 16384
$CRCBINARY config set cpus 6
#enable netmetrices addons, to make sure member cluster has proper resources
$CRCBINARY config set enable-cluster-monitoring true
#Delete existing CRC cluster to apply the config updates
$CRCBINARY delete

#Start CRC with modified configs
$CRCBINARY start

#Point local environment clients (kubectl and oc) to the CRC cluster
eval $($CRCBINARY oc-env)
kubectl config use-context crc-admin

#Reduce cpu resource request for each AppStudio Application
./hack/reduce-gitops-cpu-requests.sh