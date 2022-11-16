#!/bin/bash
#
# Make your laptop trust the local cluster's SSL cert.
#
# Probably only needed for a CRC cluster running locally.
#
# Actually cosign does have an --allow-insecure-registry option but
# trusting this cert means we don't have to use it, and I think it makes
# things easier generally for registry auth.
#
kubectl get secret router-ca -n openshift-ingress-operator -o jsonpath='{.data.tls\.crt}' | base64 -d > /tmp/cluster-ingress.pem
sudo cp /tmp/cluster-ingress.pem /etc/pki/ca-trust/source/anchors
sudo update-ca-trust
