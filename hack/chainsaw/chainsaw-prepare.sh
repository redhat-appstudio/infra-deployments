#!/bin/bash

set -e

## Create the kind cluster
kind create cluster --name infra-deployments-chainsaw

## Install kyverno
kustomize build --enable-helm components/kyverno/chainsaw | \
  kubectl apply -f - --server-side

## wait for kyverno to rollout
kubectl rollout status deployment \
  --namespace konflux-kyverno \
  --selector '!job-name' \
  --timeout=300s
