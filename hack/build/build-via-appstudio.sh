#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

COMPONENT=$1
PATH_TO_DOCKERFILE=$2
[ -z "$MY_GITHUB_TOKEN" ] && echo "error: MY_GITHUB_TOKEN env var is not exported" && exit 1

# Configure namespace
$SCRIPTDIR/setup-namespace.sh

oc delete --ignore-not-found -f $SCRIPTDIR/templates/application.yaml
oc create -f $SCRIPTDIR/templates/application.yaml

function create-secret {
  yq e "(.stringData.password=\"$MY_GITHUB_TOKEN\")" $SCRIPTDIR/templates/secret.yaml | oc apply -f-
}

function create-component {
  GIT_URL=$1
  PATH_TO_DOCKERFILE=$2
  REPO=$(echo $GIT_URL | grep -o '[^/]*$')
  NAME=${REPO%%.git}
  oc delete --ignore-not-found component $NAME
  [ -n "$SKIP_INITIAL_CHECKS" ] && ANNOTATE_SKIP_INITIAL_CHECKS='| (.metadata.annotations.skip-initial-checks="true")'
  [ -n "$ENABLE_PAC" ] && ANNOTATE_PAC_PROVISION='| (.metadata.annotations."build.appstudio.openshift.io/request"="configure-pac")'
  yq e "(.metadata.name=\"$NAME\") | (.spec.componentName=\"$NAME\") | (.spec.source.git.url=\"$GIT_URL\") | (.spec.source.git.dockerfileUrl=\"$PATH_TO_DOCKERFILE\") $ANNOTATE_PAC_PROVISION $ANNOTATE_SKIP_INITIAL_CHECKS" $SCRIPTDIR/templates/component.yaml | oc apply -f-
}

create-secret

if [ -z "$COMPONENT" ]; then
  create-component https://github.com/devfile-samples/devfile-sample-java-springboot-basic docker/Dockerfile
  create-component https://github.com/devfile-samples/devfile-sample-code-with-quarkus src/main/docker/Dockerfile.jvm.staged
  create-component https://github.com/devfile-samples/devfile-sample-python-basic docker/Dockerfile
else
  create-component $COMPONENT $PATH_TO_DOCKERFILE
fi
echo "Run this to show running builds: tkn pr list"
