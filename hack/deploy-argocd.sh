#!/bin/bash -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"/..

main() {
    verify_permissions || exit $?
    create_subscription
    wait_for_route
    switch_route_to_reencrypt
    grant_admin_role_to_all_authenticated_users
    mark_pending_pvc_as_healty
    add_role_binding
    print_url

}

verify_permissions() {
    if [ "$(oc auth can-i '*' '*' --all-namespaces)" != "yes" ]; then
        echo
        echo "[ERROR] User '$(oc whoami)' does not have the required 'cluster-admin' role." 1>&2
        echo "Log into the cluster with a user with the required privileges (e.g. kubeadmin) and retry."
        return 1
    fi
}

create_subscription() {

    echo "Installing the OpenShift GitOps operator subscription:"
    kubectl apply -k "$ROOT/components/gitops/openshift-gitops/overlays/production-and-dev"
    echo -n "Waiting for default project (and namespace) to exist: "
    while ! kubectl get appproject/default -n openshift-gitops &>/dev/null; do
        echo -n .
        sleep 1
    done
    echo "OK"
}

wait_for_route() {
    echo -n "Waiting for OpenShift GitOps Route: "
    while ! kubectl get route/openshift-gitops-server -n openshift-gitops &>/dev/null; do
        echo -n .
        sleep 1
    done
    echo "OK"
}

switch_route_to_reencrypt() {
    echo Switch the Route to use re-encryption
    kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec": {"server": {"route": {"enabled": true, "tls": {"termination": "reencrypt"}}}}}' --type=merge
    # After changing the tls method, a restart is needed other wise we
    # experience timeouts in the UI.
    echo Restarting ArgoCD Server
    oc delete pod -l app.kubernetes.io/name=openshift-gitops-server -n openshift-gitops
}

grant_admin_role_to_all_authenticated_users() {
    echo Allow any authenticated users to be admin on the Argo CD instance
    # - Once we have a proper access policy in place, this should be updated to be consistent with that policy.
    kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec":{"rbac":{"policy":"g, system:authenticated, role:admin"}}}' --type=merge
}

mark_pending_pvc_as_healty() {
    echo Mark Pending PVC as Healthy, workaround for WaitForFirstConsumer StorageClasses.
    # If the attachment will fail then it will be visible on the pod anyway.
    kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  resourceCustomizations: |
    PersistentVolumeClaim:
      health.lua: |
        hs = {}
        if obj.status ~= nil then
          if obj.status.phase ~= nil then
            if obj.status.phase == "Pending" then
              hs.status = "Healthy"
              hs.message = obj.status.phase
              return hs
            end
            if obj.status.phase == "Bound" then
              hs.status = "Healthy"
              hs.message = obj.status.phase
              return hs
            end
          end
        end
        hs.status = "Progressing"
        return hs
' --type=merge
}

add_role_binding() {
    echo "Add Role/RoleBindings for OpenShift GitOps:"
    kubectl apply --kustomize $ROOT/components/gitops/openshift-gitops/base/cluster-rbac
}

print_url() {
    local argo_cd_route argo_cd_url

    argo_cd_route=$(
        kubectl get \
            -n openshift-gitops \
            -o template \
            --template={{.spec.host}} \
            route/openshift-gitops-server
    )
    argo_cd_url="https://$argo_cd_route"

    echo
    echo "========================================================================="
    echo
    echo "Argo CD URL is: $argo_cd_url"
    echo
    echo "(NOTE: It may take a few moments for the route to become available)"
    echo
    echo -n "Waiting for the route: "
    while ! curl --fail --insecure --output /dev/null --silent "$argo_cd_url"; do
        echo -n .
        sleep 3
    done
    echo "OK"
    echo
    echo "Login/password uses your OpenShift credentials ('Login with OpenShift' button)"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
