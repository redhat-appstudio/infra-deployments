#!/bin/bash
# This script sets up the pre-requisites for running the chains demos on a
# Mac-based OS.

brew update

# Install the tekton cli
brew install tektoncd-cli

# Install go
brew install go

# Install docker
brew install docker

# Install skopeo
brew install skopeo

# Install rekor cli
go install -v github.com/sigstore/rekor/cmd/rekor-cli@latest

# Install cosign
brew install cosign