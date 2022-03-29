#!/bin/bash
#This script sets up the pre-requisites for setting up app-studio in redhat based machines.
# mainly setting up pip3,yq, kubectl, kustomize
# Firstly it checks which type of system is running this
# Then it installs the dependencies one by one
sudo apt-get update -y
VERSION=v4.23.1
BINARY=yq_linux_amd64
wget https://github.com/mikefarah/yq/releases/download/${VERSION}/${BINARY}.tar.gz -O - |tar xz && sudo mv ${BINARY} /usr/bin/yq
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin
sudo apt-get install jq
