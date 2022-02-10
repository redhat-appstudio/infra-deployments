#!/bin/bash
# Executes OAuth flow based on:
# https://github.com/redhat-appstudio/service-provider-integration-operator#go-through-the-oauth-flow-manually

kubectl -n default apply -f https://raw.githubusercontent.com/redhat-appstudio/service-provider-integration-operator/main/hack/give-default-sa-perms-for-accesstokens.yaml
kubectl -n default apply -f https://raw.githubusercontent.com/redhat-appstudio/service-provider-integration-operator/main/samples/spiaccesstokenbinding.yaml --as system:serviceaccount:default:default
SPI_ACCESS_TOKEN=$(kubectl get -n default spiaccesstokenbindings test-access-token-binding -o=jsonpath='{.status.linkedAccessTokenName}')
OAUTH_URL=$(kubectl get -n default spiaccesstoken $SPI_ACCESS_TOKEN -o=jsonpath='{.status.oAuthUrl}')
LOCATION=$(curl -v -k $OAUTH_URL 2>&1 | grep 'location: ' | cut -d' ' -f3)

echo "OAuth service endpoint:"
echo $LOCATION

echo "After proceeding the flow, check the secret - kubectl get -o default secret token-secret -o yaml"
