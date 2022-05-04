#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/.. 

if [ -z "${MY_GIT_REPO_URL}" ]; then
    MY_GIT_REPO_URL=$(git ls-remote --get-url | sed 's|^git@github.com:|https://github.com/|')
fi
if [ -z "${MY_GIT_BRANCH}" ]; then
    MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi

#set the local cluster to point to the current git repo and branch and update the path to development
$ROOT/hack/util-update-app-of-apps.sh $MY_GIT_REPO_URL development $MY_GIT_BRANCH
# reset the default repos in the development directory to be the current git repo
# this needs to be pushed to your fork to be seen by argocd
$ROOT/hack/util-set-development-repos.sh $MY_GIT_REPO_URL development $MY_GIT_BRANCH

# set the API server which SPI uses to authenticate users to empty string (by default) so that multi-cluster
# setup is not needed
$ROOT/hack/util-set-spi-api-server.sh "$SPI_API_SERVER"

# set backend route for quality dashboard for currrent cluster
$ROOT/hack/util-set-quality-dashboard-backend-route.sh

if [ -n "$MY_GITHUB_ORG" ]; then
    $ROOT/hack/util-set-github-org $MY_GITHUB_ORG
fi
