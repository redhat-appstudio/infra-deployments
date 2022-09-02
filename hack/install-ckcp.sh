#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

oc apply -f $ROOT/components/ckcp/cert-manager.yaml
oc apply -f $ROOT/components/ckcp/namespace.yaml
oc apply -f $ROOT/components/ckcp/route.yaml

URL=$(oc get route -n ckcp ckcp -o jsonpath={.spec.host})
TMP_FILE=$(mktemp)
oc kustomize $ROOT/components/ckcp | sed "s/\$HOSTNAME/$URL/" > $TMP_FILE
while ! oc apply -f $TMP_FILE; do
  sleep 10
done
rm $TMP_FILE

while ! oc rsh -n ckcp deployment/ckcp ls /etc/kcp/config/admin.kubeconfig; do
  sleep 10
done
oc rsh -n ckcp deployment/ckcp sed 's/certificate-authority-data: .*/insecure-skip-tls-verify: true/' /etc/kcp/config/admin.kubeconfig > $KCP_KUBECONFIG
