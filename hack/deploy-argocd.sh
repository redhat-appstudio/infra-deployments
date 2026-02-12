#!/bin/bash -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"/..

# =============================================================================
# Logging Functions (consistent with preview.sh)
# =============================================================================
timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

log_info() {
    echo "[$(timestamp)] [INFO] $*"
}

log_success() {
    echo "[$(timestamp)] [SUCCESS] $*"
}

log_warn() {
    echo "[$(timestamp)] [WARN] $*"
}

log_error() {
    echo "[$(timestamp)] [ERROR] $*" >&2
}

log_step() {
    echo ""
    echo "============================================================================="
    echo "[$(timestamp)] [STEP] $*"
    echo "============================================================================="
}

log_substep() {
    echo "[$(timestamp)] [SUBSTEP] $*"
}

log_wait() {
    echo "[$(timestamp)] [WAITING] $*"
}

log_progress() {
    echo "[$(timestamp)] [PROGRESS] $*"
}

# =============================================================================
# Main Functions
# =============================================================================

main() {
    log_step "Deploying OpenShift GitOps (ArgoCD)"
    
    verify_permissions || exit $?
    create_subscription
    wait_for_route
    update_repo_server_requests_and_timeout
    update_application_controller_resources
    switch_route_to_reencrypt
    grant_admin_role_to_all_authenticated_users
    mark_pending_pvc_as_healty
    set_kustomize_build_options
    set_ignoreaggregatedroles
    set_trackingmethod_annotation
    restart_gitops_server
    print_url
    
    log_step "OpenShift GitOps Deployment Complete"
}

verify_permissions() {
    log_substep "Verifying cluster-admin permissions"
    
    local current_user
    current_user=$(oc whoami 2>/dev/null || echo "unknown")
    
    if [ "$(oc auth can-i '*' '*' --all-namespaces)" != "yes" ]; then
        log_error "============================================================================="
        log_error "PERMISSION DENIED: Cluster-admin role required"
        log_error "============================================================================="
        log_error "Current user: $current_user"
        log_error "Required permission: cluster-admin"
        log_error ""
        log_error "ACTION REQUIRED: Log into the cluster with a user that has cluster-admin role"
        log_error "  Example: oc login -u kubeadmin"
        log_error "  Or grant cluster-admin: oc adm policy add-cluster-role-to-user cluster-admin $current_user"
        return 1
    fi
    
    log_success "User '$current_user' has cluster-admin permissions"
}

create_subscription() {
    log_substep "Installing OpenShift GitOps operator subscription"
    
    if kubectl apply -k "$ROOT/components/openshift-gitops"; then
        log_success "GitOps operator subscription applied"
    else
        log_error "Failed to apply GitOps operator subscription"
        log_error "ACTION REQUIRED: Check if $ROOT/components/openshift-gitops exists and is valid"
        exit 1
    fi
    
    log_substep "Waiting for default ArgoCD project to be created"
    local wait_time=0
    local max_wait=300  # 5 minutes
    
    while ! kubectl get appproject/default -n openshift-gitops &>/dev/null; do
        wait_time=$((wait_time + 5))
        if [ "$wait_time" -ge "$max_wait" ]; then
            log_error "TIMEOUT: Default ArgoCD project not created after ${max_wait}s"
            log_error "ACTION REQUIRED: Check OpenShift GitOps operator status"
            log_error "  Run: oc get csv -n openshift-operators | grep gitops"
            exit 1
        fi
        log_wait "Default project not yet available (${wait_time}s/${max_wait}s)"
        sleep 5
    done
    
    log_success "Default ArgoCD project is available"
}

wait_for_route() {
    log_substep "Waiting for OpenShift GitOps route to be created"
    local wait_time=0
    local max_wait=180  # 3 minutes
    
    while ! kubectl get route/openshift-gitops-server -n openshift-gitops &>/dev/null; do
        wait_time=$((wait_time + 5))
        if [ "$wait_time" -ge "$max_wait" ]; then
            log_error "TIMEOUT: GitOps server route not created after ${max_wait}s"
            log_error "ACTION REQUIRED: Check ArgoCD deployment status"
            log_error "  Run: oc get pods -n openshift-gitops"
            exit 1
        fi
        log_wait "GitOps route not yet available (${wait_time}s/${max_wait}s)"
        sleep 5
    done
    
    log_success "OpenShift GitOps route is available"
}

update_repo_server_requests_and_timeout() {
    log_substep "Configuring ArgoCD repo server resources and timeout"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  repo:
    env:
      - name: ARGOCD_EXEC_TIMEOUT
        value: 5m
    resources:
      requests:
        cpu: 100m
        memory: 100Mi
' --type=merge; then
        log_success "Repo server configured: timeout=5m, cpu=100m, memory=100Mi"
    else
        log_warn "Failed to patch repo server configuration (may already be set)"
    fi
}

update_application_controller_resources() {
    log_substep "Configuring ArgoCD application controller resources"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  controller:
    resources:
      limits:
        cpu: 4
        memory: 4Gi
' --type=merge; then
        log_success "Application controller configured: cpu=4, memory=4Gi"
    else
        log_warn "Failed to patch controller resources (may already be set)"
    fi
}

switch_route_to_reencrypt() {
    log_substep "Switching ArgoCD route to re-encryption TLS"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec": {"server": {"route": {"enabled": true, "tls": {"termination": "reencrypt"}}}}}' --type=merge; then
        log_success "Route TLS termination set to 'reencrypt'"
    else
        log_warn "Failed to set route TLS (may already be configured)"
    fi
    
    # Restart required after TLS change to avoid UI timeouts
    log_substep "Restarting ArgoCD server after TLS configuration change"
    if oc delete pod -l app.kubernetes.io/name=openshift-gitops-server -n openshift-gitops 2>/dev/null; then
        log_success "ArgoCD server pods deleted for restart"
    else
        log_warn "No ArgoCD server pods found to restart"
    fi
}

grant_admin_role_to_all_authenticated_users() {
    log_substep "Granting admin role to authenticated users"
    log_info "Note: This should be updated once proper access policy is in place"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec":{"rbac":{"policy":"g, system:authenticated, role:admin"}}}' --type=merge; then
        log_success "RBAC policy set: system:authenticated -> role:admin"
    else
        log_warn "Failed to set RBAC policy (may already be configured)"
    fi
}

mark_pending_pvc_as_healty() {
    log_substep "Configuring PVC health check (WaitForFirstConsumer workaround)"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
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
' --type=merge; then
        log_success "PVC health customization applied (Pending/Bound = Healthy)"
    else
        log_warn "Failed to apply PVC health customization"
    fi
}

set_kustomize_build_options() {
    log_substep "Enabling Helm support in Kustomize builds"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '{"spec":{"kustomizeBuildOptions":"--enable-helm"}}' --type=merge; then
        log_success "Kustomize build options set: --enable-helm"
    else
        log_warn "Failed to set kustomize build options"
    fi
}

set_ignoreaggregatedroles() {
    log_substep "Configuring ArgoCD to ignore aggregated roles in diff"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  extraConfig:
    resource.compareoptions: |
      # disables status field diffing in specified resource types
      ignoreAggregatedRoles: true
' --type=merge; then
        log_success "ignoreAggregatedRoles set to true"
    else
        log_warn "Failed to set ignoreAggregatedRoles"
    fi
}

set_trackingmethod_annotation() {
    log_substep "Setting ArgoCD tracking method to annotation"
    
    if kubectl patch argocd/openshift-gitops -n openshift-gitops -p '
spec:
  resourceTrackingMethod: annotation
' --type=merge; then
        log_success "Resource tracking method set to 'annotation'"
    else
        log_warn "Failed to set tracking method"
    fi
}

restart_gitops_server() {
    log_substep "Restarting GitOps server deployment"
    
    if kubectl rollout restart -n openshift-gitops deployments openshift-gitops-server; then
        log_success "GitOps server restart initiated"
        
        # Wait for rollout to complete
        log_info "Waiting for rollout to complete..."
        if kubectl rollout status -n openshift-gitops deployment/openshift-gitops-server --timeout=120s 2>/dev/null; then
            log_success "GitOps server rollout complete"
        else
            log_warn "Rollout status check timed out (server may still be starting)"
        fi
    else
        log_warn "Failed to restart GitOps server"
    fi
}

print_url() {
    log_step "ArgoCD Access Information"
    
    local argo_cd_route argo_cd_url
    
    argo_cd_route=$(
        kubectl get \
            -n openshift-gitops \
            -o template \
            --template={{.spec.host}} \
            route/openshift-gitops-server 2>/dev/null || echo ""
    )
    
    if [ -z "$argo_cd_route" ]; then
        log_error "Failed to get ArgoCD route"
        return 1
    fi
    
    argo_cd_url="https://$argo_cd_route"
    
    log_info "ArgoCD URL: $argo_cd_url"
    log_info "Authentication: Use 'Login with OpenShift' button (OpenShift credentials)"
    
    log_substep "Verifying ArgoCD route is accessible"
    local wait_time=0
    local max_wait=120
    
    while ! curl --fail --insecure --output /dev/null --silent "$argo_cd_url"; do
        wait_time=$((wait_time + 5))
        if [ "$wait_time" -ge "$max_wait" ]; then
            log_warn "Route not responding after ${max_wait}s (may still be starting)"
            break
        fi
        log_wait "Waiting for route to respond (${wait_time}s/${max_wait}s)"
        sleep 5
    done
    
    if [ "$wait_time" -lt "$max_wait" ]; then
        log_success "ArgoCD is accessible at $argo_cd_url"
    fi
    
    # Output JSON for LLM parsing
    echo ""
    echo "[ARGOCD_DEPLOY_JSON] {\"status\":\"success\",\"url\":\"$argo_cd_url\",\"namespace\":\"openshift-gitops\",\"auth_method\":\"openshift\"}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
