#!/bin/bash
# This script sets up the pre-requisites for running the chains demos on a
# Debian-based OS (Debian, Ubuntu, etc).

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

# Install the tekton cli
exist_or_install "tkn version" "VERSION=0.23.1; BINARY=Linux_x86_64; curl -LO https://github.com/tektoncd/cli/releases/download/v${VERSION}/tkn_${VERSION}_${BINARY}.tar.gz && sudo tar xvzf tkn_${VERSION}_${BINARY}.tar.gz -C /usr/local/bin/ tkn"

sudo apt-get update -y

# Install go
exist_or_install "go version" "sudo apt-get install golang"

# Install docker
exist_or_install "docker version" "sudo apt-get install docker.io"

# Install skopeo
exist_or_install "skopeo -v" "sudo apt-get install skopeo"

# Install rekor cli
exist_or_install "rekor-cli version" "go install github.com/sigstore/rekor/cmd/rekor-cli@latest"

# Install cosign
exist_or_install "cosign version" "go install github.com/sigstore/cosign/cmd/cosign@latest"
