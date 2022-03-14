#!/bin/bash
#This script sets up the pre-requisites for setting up app-studio in redhat based machines.
# mainly setting up pip3,yq, kubectl, kustomize
# Firstly it checks which type of system is running this
# Then it installs the dependencies one by one

sudo yum -y install pip3
sudo pip3 install --upgrade setuptools
sudo snap install yq
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
sudo cp kustomize /usr/bin/
sudo chmod ug+x /usr/bin/kustomize