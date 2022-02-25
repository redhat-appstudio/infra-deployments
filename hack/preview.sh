#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [ -f $ROOT/hack/preview.env ]; then
    source $ROOT/hack/preview.env
fi

if [ -z "$MY_GIT_FORK_REMOTE" ]; then
    echo "Set MY_GIT_FORK_REMOTE environment to name of your fork remote"
    exit 1
fi

MY_GIT_REPO_URL=$(git ls-remote --get-url $MY_GIT_FORK_REMOTE | sed 's|^git@github.com:|https://github.com/|')
MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)


if echo "$MY_GIT_REPO_URL" | grep -q redhat-appstudio/infra-deployments; then
    echo "Use your fork repository for preview"
    exit 1
fi

if ! git diff --exit-code --quiet; then
    echo "Changes in working Git working tree, commit them or stash them"
    exit 1
fi

# Create preview branch for preview configuration
PREVIEW_BRANCH=preview-${MY_GIT_BRANCH}${TEST_BRANCH_ID+-$TEST_BRANCH_ID}
if git rev-parse --verify $PREVIEW_BRANCH; then
    git branch -D $PREVIEW_BRANCH
fi
git checkout -b $PREVIEW_BRANCH

# reset the default repos in the development directory to be the current git repo
# this needs to be pushed to your fork to be seen by argocd
$ROOT/hack/util-set-development-repos.sh $MY_GIT_REPO_URL development $PREVIEW_BRANCH

if [ -n "$MY_GITHUB_ORG" ]; then
    $ROOT/hack/util-set-github-org $MY_GITHUB_ORG
fi

if [ -n "$SHARED_SECRET" ] && [ -n "$SPI_TYPE" ] && [ -n "$SPI_CLIENT_ID" ] && [ -n "$SPI_CLIENT_SECRET" ]; then
    TMP_FILE=$(mktemp)
    ROUTE="https://$(oc get routes -n spi-system spi-oauth-route -o yaml -o jsonpath='{.spec.host}')"
    yq e ".sharedSecret=\"$SHARED_SECRET\"" $ROOT/components/spi/config.yaml | \
        yq e ".serviceProviders[0].type=\"$SPI_TYPE\"" - | \
        yq e ".serviceProviders[0].clientId=\"$SPI_CLIENT_ID\"" - | \
        yq e ".serviceProviders[0].clientSecret=\"$SPI_CLIENT_SECRET\"" - | \
        yq e ".baseUrl=\"$ROUTE\"" - > $TMP_FILE
    oc create -n spi-system secret generic oauth-config --from-file=config.yaml=$TMP_FILE --dry-run=client -o yaml | oc apply -f -
    echo "SPI configurared, set Authorization callback URL to $ROUTE"
    rm $TMP_FILE
fi

if ! git diff --exit-code --quiet; then
    git commit -a -m "Preview mode, do not merge into main"
    git push -f --set-upstream $MY_GIT_FORK_REMOTE $PREVIEW_BRANCH
fi

git checkout $MY_GIT_BRANCH

#set the local cluster to point to the current git repo and branch and update the path to development
$ROOT/hack/util-update-app-of-apps.sh $MY_GIT_REPO_URL development $PREVIEW_BRANCH
