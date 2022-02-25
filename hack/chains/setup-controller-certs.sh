#!/bin/bash
#
# Based on Christophe's procedure to get working SSL in a
# local CRC cluster so the chains signed provenance demo works.
# See https://docs.engineering.redhat.com/x/zFp-Dw .
#
# This is a temporary workaround. There should be a better way to
# ensure the chains controller has access a set of CA certs -
# maybe with https://cert-manager.io/ , or maybe the certs are
# available already somewhere else in the cluster and just need
# to be exposed to the chains controller.
#

source $(dirname $0)/_helpers.sh
set -u

TMP_DIR=$( mktemp -d )

#
# First prepare some CA certs and create a secret to hold them
#

# A cert from your local cluster to get local SSL working
kubectl get secret router-ca -n openshift-ingress-operator -o jsonpath="{.data.tls\.crt}" | base64 -d > $TMP_DIR/cluster-ingress.pem

# Certs from your laptop to get SSL working on the internet
cp /etc/ssl/cert.pem $TMP_DIR/cert.pem

CREATE_OR_REPLACE=create
kubectl get secret chains-ca-cert -n tekton-chains >/dev/null 2>&1 && CREATE_OR_REPLACE=replace

# Put them all together in a secret
# (Use sed for yaml indenting)
# (FYI we can't use apply here because the data is too big for the annotation field)
cat << EOF | kubectl $CREATE_OR_REPLACE -n tekton-chains -f -
apiVersion: v1
kind: Secret
metadata:
  name: chains-ca-cert
data:
  ca-certificates.crt: |
$( cat $TMP_DIR/*.pem | base64 | sed 's/^/    /' )
EOF

# Uncomment if you want to show all the certs:
#kubectl get secret chains-ca-cert -n tekton-chains -o jsonpath='{.data.ca-certificates\.crt}' | base64 -d

#
# Now make the certs accessible where they're needed
#

PATCH_YAML="
spec:
  template:
    spec:
      containers:
      - name: tekton-chains-controller
        volumeMounts:
        - mountPath: /etc/ssl/certs
          name: chains-ca-cert
      volumes:
      - name: chains-ca-cert
        secret:
          secretName: chains-ca-cert
"
kubectl patch deployment tekton-chains-controller -n tekton-chains --patch "$PATCH_YAML"

# Cleanup
rm -rf $TMP_DIR
