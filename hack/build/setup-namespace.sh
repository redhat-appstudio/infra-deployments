#!/bin/bash

# Script for setting namespace which is not managed by toolchain host operator

CURRENT_NAMESPACE=$(oc config view --minify -o 'jsonpath={..namespace}')
oc label namespace $CURRENT_NAMESPACE --overwrite argocd.argoproj.io/managed-by=gitops-service-argocd
