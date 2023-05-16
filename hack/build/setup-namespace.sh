#!/bin/bash

# Script for setting namespace which is not managed by toolchain host operator

CURRENT_NAMESPACE=$(oc config view --minify -o 'jsonpath={..namespace}')
oc label namespace $CURRENT_NAMESPACE --overwrite argocd.argoproj.io/managed-by=gitops-service-argocd
oc apply -f https://raw.githubusercontent.com/codeready-toolchain/member-operator/master/config/appstudio-pipelines-runner/base/appstudio_pipelines_runner_role.yaml
oc create --dry-run=client -o yaml serviceaccount appstudio-pipeline | oc apply -f-
oc create --dry-run=client -o yaml rolebinding appstudio-pipelines-runner-rolebinding --clusterrole=appstudio-pipelines-runner --serviceaccount=$CURRENT_NAMESPACE:appstudio-pipeline | oc apply -f-
