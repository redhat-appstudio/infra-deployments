#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

if [ "$1" == "--destroy" ]; then
  oc -n openshift-gitops delete application pipeline-service
  oc -n openshift-gitops delete application all-components --wait=false
  oc -n openshift-gitops delete applicationset --all
  sleep 5
  for APP in `oc get -n openshift-gitops applications.argoproj.io -o name`; do 
    oc -n openshift-gitops patch  $APP --type json --patch='[ { "op": "remove", "path": "/metadata/finalizers" } ]'
  done
  KCP_PROJECTS=$(oc get projects -o name | grep kcp)
  while [ -n "$KCP_PROJECTS" ]; do 
    oc delete $KCP_PROJECTS
    KCP_PROJECTS=$(oc get projects -o name | grep kcp)
  done
  oc delete project spi-vault
fi

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

CKCP_KUBECONFIG=${CKCP_KUBECONFIG:-/tmp/ckcp-admin.kubeconfig}
oc rsh -n ckcp deployment/ckcp sed 's/certificate-authority-data: .*/insecure-skip-tls-verify: true/' /etc/kcp/config/admin.kubeconfig > ${CKCP_KUBECONFIG}

echo
echo "=========================================================================================="
echo "ckcp admin kubeconfig was stored in ${CKCP_KUBECONFIG}. To use the kubeconfig run:"
echo
echo "export KUBECONFIG=${CKCP_KUBECONFIG}"
echo
echo
echo "To use the kubeconfig for bootstrap.sh copy it to file pointed by KCP_KUBECONFIG variable in hack/preview.env"
echo
echo "=========================================================================================="
echo
