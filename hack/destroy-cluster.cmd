@echo off 
set ROOT=%~dp0\..

rem Remove resources in the reverse order from bootstrapping

echo.
echo "Remove Argo CD Applications:"
kubectl delete -f %ROOT%/argo-cd-apps/app-of-apps/all-applications-staging.yaml
rem kustomize build  %ROOT%/argo-cd-apps/overlays/staging | kubectl delete -f -

echo.
echo "Remove RBAC for OpenShift GitOps:"
kustomize build %ROOT%/openshift-gitops/cluster-rbac | kubectl delete -f -

echo.
echo "Remove the OpenShift GitOps operator subscription:"
kubectl delete -f %ROOT%/openshift-gitops/subscription-openshift-gitops.yaml

echo. 
echo "Removing GitOps operator and operands:"

:L1 
  kubectl patch argocd/openshift-gitops -n openshift-gitops -p "{\"metadata\":{\"finalizers\":null}}" --type=merge
  kubectl delete csv/openshift-gitops-operator.v1.3.0 -n openshift-operators
  kubectl delete csv/openshift-gitops-operator.v1.3.0 -n openshift-gitops
  kubectl delete csv/openshift-gitops-operator.v1.3.1 -n openshift-operators
  kubectl delete csv/openshift-gitops-operator.v1.3.1 -n openshift-gitops
  kubectl delete namespace --timeout=30s openshift-gitops

  kubectl get namespace/openshift-gitops
  set RC=%ERRORLEVEL%
  if %RC% == 1 goto :end  
  echo %RC%
goto :L1

:end

echo.
echo "Complete."