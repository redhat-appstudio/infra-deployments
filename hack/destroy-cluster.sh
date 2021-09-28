#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..


kustomize build $ROOT/argo-cd/ | kubectl delete -f -

kubectl delete namespace/cluster-argocd

