#!/bin/bash
# This script sets up the pre-requisites for running the chains demos on a
# Mac-based OS.

exist_or_install () {
    local check_command=$1
    local install_command=$2
    
    eval $check_command
    local status=$?

    if [[ $status -eq 127 ]]
    then
        eval $install_command
    fi
} # end exist_or_install

brew update

# Install the tekton cli
exist_or_install "tkn version" "brew install tektoncd-cli"

# Install go (needed for installing rekor cli)
exist_or_install "go version" "brew install go"

# Install docker
exist_or_install "docker version" "brew install docker"

# Install skopeo
exist_or_install "skopeo -v" "brew install skopeo"

# Install rekor cli
exist_or_install "rekor-cli version" "go install github.com/sigstore/rekor/cmd/rekor-cli@latest"

# Install cosign
exist_or_install "cosign version" "brew install cosign"
