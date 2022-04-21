#!/bin/bash
# This script sets up the pre-requisites for running the chains demos on a
# Debian-based OS (Debian, Ubuntu, etc).

# Install the tekton cli
VERSION=v0.23.1
BINARY=Linux_x86_64
curl -LO https://github.com/tektoncd/cli/releases/download/${VERSION}/tkn_${VERSION}_${BINARY}.tar.gz
sudo tar xvzf tkn_${VERSION}_${BINARY}.tar.gz -C /usr/local/bin/ tkn

sudo apt-get update -y

# Install go
sudo apt-get install golang

# Install docker
sudo apt-get install docker.io

# Install skopeo
sudo apt-get install skopeo

# Install rekor cli
go install -v github.com/sigstore/rekor/cmd/rekor-cli@latest

# Install cosign
go install github.com/sigstore/cosign/cmd/cosign@latest