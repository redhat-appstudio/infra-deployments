#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
KUSTOMIZATION_FILE="$ROOT/components/quality-dashboard/frontend/kustomization.yaml"
CLUSTER_URL_HOST=$(oc whoami --show-console|sed 's|https://console-openshift-console.apps.||')

yq e -i "(.configMapGenerator[].literals[] | select(. == \"*BACKEND_ROUTE*\")) = \"BACKEND_ROUTE=https://quality-backend-route-quality-dashboard.apps.$CLUSTER_URL_HOST\"" "$KUSTOMIZATION_FILE"
