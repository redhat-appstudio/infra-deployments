#!/bin/bash

echo
echo "Disable all default sources for OperatrHub:"
kubectl patch operatorhub.config.openshift.io/cluster -p='{"spec":{"disableAllDefaultSources":true}}' --type=merge

echo 
echo "Complete."
