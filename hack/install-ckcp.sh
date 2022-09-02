#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

oc apply -k $ROOT/components/ckcp

while ! oc rsh -n ckcp deployment/ckcp ls /etc/kcp/config/admin.kubeconfig; do
  sleep 10
done
oc rsh -n ckcp deployment/ckcp cat /etc/kcp/config/admin.kubeconfig > $KCP_KUBECONFIG
