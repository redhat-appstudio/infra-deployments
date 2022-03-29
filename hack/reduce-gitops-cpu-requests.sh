#!/bin/bash

OPENSHIFT_GITOPS=$(kubectl get argocd/openshift-gitops -n openshift-gitops -o json)

echo
echo "Reducing CPU resource requests in argocd/openshift-gitops:"

PATCH="["
for KEY in $(echo ${OPENSHIFT_GITOPS} | jq -r '.spec | keys[]');
do
	if [[ -n "$(echo ${OPENSHIFT_GITOPS} | jq -r ".spec.${KEY} | select(.resources.requests.cpu != null)" 2>/dev/null)" ]];
	then
	    PATCH+='{"op": "replace", "path": "/spec/'${KEY}'/resources/requests/cpu", "value": "50m"},'
	fi
done
kubectl patch argocd/openshift-gitops -n openshift-gitops --type='json' -p "${PATCH}]"
