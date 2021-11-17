@echo off 

set ROOT=%~dp0\..

echo "Installing the OpenShift GitOps operator subscription:"
kubectl apply -f %ROOT%/openshift-gitops/subscription-openshift-gitops.yaml

echo.
echo "Waiting for default project (and namespace) to exist:" 
:L1
choice /D N /T 1  > nul
kubectl get appproject/default -n openshift-gitops 2>nul 
if %ERRORLEVEL% == 1 goto :L1 

echo.
echo "Waiting for OpenShift GitOps Route:"
:L2
choice /D N /T 1  > nul
kubectl get route/openshift-gitops-server -n openshift-gitops 2>nul 
if %ERRORLEVEL% == 1 goto :L2
 
echo.
echo "Add Role/RoleBindings for OpenShift GitOps:"
kustomize build %ROOT%/openshift-gitops/cluster-rbac | kubectl apply -f -
 
echo.
echo "Add parent Argo CD Application:"
kubectl apply -f %ROOT%/argo-cd-apps/app-of-apps/all-applications-staging.yaml
 
echo.
echo "========================================================================="
echo "Argo CD Route is:"
kubectl get route/openshift-gitops-server -n openshift-gitops
echo.
echo "(NOTE: It may take a few moments for the route to become available)"
echo.
echo "Login/password uses your OpenShift credentials ('Login with OpenShift' button)"
echo.



