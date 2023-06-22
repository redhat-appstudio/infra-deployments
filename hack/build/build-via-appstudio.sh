#!/bin/bash
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

COMPONENT=$1

# Configure namespace
$SCRIPTDIR/setup-namespace.sh

oc delete --ignore-not-found -f $SCRIPTDIR/templates/application.yaml
oc create -f $SCRIPTDIR/templates/application.yaml
if ! oc wait --for=condition=Created application/test-application; then
  echo "Application was not created sucessfully, check:"
  echo "oc get applications test-application -o yaml"
  exit 1
fi

function create-component {
  GIT_URL=$1
  REPO=$(echo $GIT_URL | grep -o '[^/]*$')
  NAME=${REPO%%.git}
  oc delete --ignore-not-found component $NAME
  [ -n "$SKIP_INITIAL_CHECKS" ] && ANNOTATE_SKIP_INITIAL_CHECKS='| (.metadata.annotations.skip-initial-checks="true")'
  [ -n "$ENABLE_PAC" ] && ANNOTATE_PAC_PROVISION='| (.metadata.annotations."appstudio.openshift.io/request"="configure-pac")'
  yq e "(.metadata.name=\"$NAME\") | (.spec.componentName=\"$NAME\") | (.spec.source.git.url=\"$GIT_URL\") | (.spec.containerImage=\"$IMAGE\") $ANNOTATE_PAC_PROVISION $ANNOTATE_SKIP_INITIAL_CHECKS" $SCRIPTDIR/templates/component.yaml | oc apply -f-
}

echo Git Repo created:
oc get application/test-application -o jsonpath='{.status.devfile}' | grep appModelRepository.url | cut -f2- -d':'

if [ -z "$COMPONENT" ]; then
  create-component https://github.com/devfile-samples/devfile-sample-java-springboot-basic
  create-component https://github.com/devfile-samples/devfile-sample-code-with-quarkus
  create-component https://github.com/devfile-samples/devfile-sample-python-basic
else
  create-component $COMPONENT
fi
echo "Run this to show running builds: tkn pr list"
