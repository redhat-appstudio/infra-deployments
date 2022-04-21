#!/bin/bash
# This script sets up the pre-requisites for running the chains demos on a
# Red Hat-based OS (Fedora, RHEL).

# Install the tekton cli
VERSION=v0.23.1
BINARY=Linux_x86_64
curl -LO https://github.com/tektoncd/cli/releases/download/${VERSION}/tkn_${VERSION}_${BINARY}.tar.gz
sudo tar xvzf tkn_${VERSION}_${BINARY}.tar.gz -C /usr/local/bin/ tkn

# Install go
sudo dnf install golang

# Install docker
sudo dnf install moby-engine

# Install skopeo
sudo dnf install skopeo

# Install rekor cli
go install github.com/sigstore/rekor/cmd/rekor-cli@latest

# Install cosign
go install github.com/sigstore/cosign/cmd/cosign@latest