#!/bin/bash -e
set -o pipefail

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..

# =============================================================================
# Constants
# =============================================================================
ARGOCD_NAMESPACE="openshift-gitops"
PIPELINES_NAMESPACE="openshift-pipelines"
SYNC_INTERVAL=10
MAX_TEKTON_CRD_RETRIES=5
MAX_SYNC_TIMEOUT=2700           # 45 minutes max for all apps to sync
MAX_TEKTON_READY_TIMEOUT=900    # 15 minutes max for Tekton to become ready
DETAILED_STATUS_INTERVAL=120    # Show detailed status every 2 minutes

# Track execution timing for summary
SCRIPT_START_TIME=$(date +%s)
SCRIPT_STATUS="in_progress"
TOTAL_APPS_DEPLOYED=0
FAILED_APPS=""

# =============================================================================
# Logging Functions
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

log_debug() {
    echo "[$(timestamp)] [DEBUG] $*"
}

# Print cluster context information for LLM understanding
print_cluster_context() {
    log_step "Cluster Context Information"

    local ocp_version cluster_url node_count api_server cluster_id

    # Get OCP version
    ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}' 2>/dev/null || echo "Unknown")

    # Get cluster API server URL
    api_server=$(oc whoami --show-server 2>/dev/null || echo "Unknown")

    # Get cluster ID
    cluster_id=$(oc get clusterversion version -o jsonpath='{.spec.clusterID}' 2>/dev/null || echo "Unknown")

    # Get node count and info
    node_count=$(oc get nodes --no-headers 2>/dev/null | wc -l | tr -d ' ' || echo "0")

    # Get cluster domain
    cluster_url=$(oc get ingresses.config.openshift.io cluster --template={{.spec.domain}} 2>/dev/null || echo "Unknown")

    log_info "OpenShift Version: $ocp_version"
    log_info "API Server: $api_server"
    log_info "Cluster ID: $cluster_id"
    log_info "Cluster Domain: $cluster_url"
    log_info "Total Nodes: $node_count"

    # Show node details (master/worker breakdown)
    local master_nodes worker_nodes
    master_nodes=$(oc get nodes -l node-role.kubernetes.io/master --no-headers 2>/dev/null | wc -l | tr -d ' ' || echo "0")
    worker_nodes=$(oc get nodes -l node-role.kubernetes.io/worker --no-headers 2>/dev/null | wc -l | tr -d ' ' || echo "0")
    log_info "  - Master nodes: $master_nodes"
    log_info "  - Worker nodes: $worker_nodes"

    # Check cluster health
    local cluster_operators_degraded
    cluster_operators_degraded=$(oc get clusteroperators -o json 2>/dev/null | jq '[.items[] | select(.status.conditions[] | select(.type=="Degraded" and .status=="True"))] | length' 2>/dev/null || echo "Unknown")

    if [ "$cluster_operators_degraded" = "0" ]; then
        log_success "Cluster operators: All healthy"
    else
        log_warn "Cluster operators: $cluster_operators_degraded degraded"
    fi

    # Output JSON context for LLM parsing
    echo ""
    echo "[CLUSTER_CONTEXT_JSON] {\"ocp_version\":\"$ocp_version\",\"api_server\":\"$api_server\",\"cluster_id\":\"$cluster_id\",\"cluster_domain\":\"$cluster_url\",\"total_nodes\":$node_count,\"master_nodes\":$master_nodes,\"worker_nodes\":$worker_nodes,\"degraded_operators\":$cluster_operators_degraded}"
}

# Print execution summary in JSON format for LLM parsing
print_execution_summary() {
    local status=$1
    local failure_reason=${2:-""}

    local end_time total_time_seconds total_time_min total_time_sec
    end_time=$(date +%s)
    total_time_seconds=$((end_time - SCRIPT_START_TIME))
    total_time_min=$((total_time_seconds / 60))
    total_time_sec=$((total_time_seconds % 60))

    log_step "Execution Summary"

    if [ "$status" = "success" ]; then
        log_success "Status: SUCCESS"
    else
        log_error "Status: FAILED"
        [ -n "$failure_reason" ] && log_error "Failure Reason: $failure_reason"
    fi

    log_info "Total Execution Time: ${total_time_min}m ${total_time_sec}s ($total_time_seconds seconds)"
    log_info "Applications Deployed: $TOTAL_APPS_DEPLOYED"

    if [ -n "$FAILED_APPS" ]; then
        log_error "Failed Applications: $FAILED_APPS"
    fi

    # Build JSON summary
    local json_summary
    if [ "$status" = "success" ]; then
        json_summary="{\"status\":\"success\",\"total_time_seconds\":$total_time_seconds,\"apps_deployed\":$TOTAL_APPS_DEPLOYED,\"ocp_version\":\"${OCP_VERSION:-unknown}\",\"preview_branch\":\"${PREVIEW_BRANCH:-unknown}\",\"git_repo\":\"${MY_GIT_REPO_URL:-unknown}\"}"
    else
        local failed_apps_json
        failed_apps_json=$(echo "$FAILED_APPS" | tr ' ' ',' | sed 's/,$//')
        json_summary="{\"status\":\"failed\",\"total_time_seconds\":$total_time_seconds,\"apps_deployed\":$TOTAL_APPS_DEPLOYED,\"failure_reason\":\"$failure_reason\",\"failed_apps\":\"$failed_apps_json\",\"ocp_version\":\"${OCP_VERSION:-unknown}\",\"preview_branch\":\"${PREVIEW_BRANCH:-unknown}\"}"
    fi

    echo ""
    echo "[EXECUTION_SUMMARY_JSON] $json_summary"
}

# Show detailed status for an ArgoCD application (for debugging failures)
show_app_details() {
    local app_name=$1
    local app_json

    app_json=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app_name" -o json 2>/dev/null || echo '{}')

    local sync_status health_status message resources_summary
    sync_status=$(echo "$app_json" | jq -r '.status.sync.status // "Unknown"')
    health_status=$(echo "$app_json" | jq -r '.status.health.status // "Unknown"')
    message=$(echo "$app_json" | jq -r '.status.conditions[0].message // "No message"' | head -c 200)

    log_info "  ├─ App: $app_name"
    log_info "  │  ├─ Sync Status: $sync_status"
    log_info "  │  ├─ Health Status: $health_status"

    # Show resource sync status summary
    local out_of_sync degraded_resources
    out_of_sync=$(echo "$app_json" | jq -r '[.status.resources[]? | select(.status != "Synced")] | length // 0')
    degraded_resources=$(echo "$app_json" | jq -r '[.status.resources[]? | select(.health.status == "Degraded" or .health.status == "Missing")] | .[0:3] | .[] | "\(.kind)/\(.name): \(.health.status // .status)"' 2>/dev/null || echo "")

    if [ "$out_of_sync" -gt 0 ]; then
        log_info "  │  ├─ Out-of-sync resources: $out_of_sync"
    fi

    if [ -n "$degraded_resources" ]; then
        log_warn "  │  ├─ Degraded/Missing resources:"
        echo "$degraded_resources" | while IFS= read -r line; do
            log_warn "  │  │  └─ $line"
        done
    fi

    # Show conditions/errors
    local conditions
    conditions=$(echo "$app_json" | jq -r '.status.conditions[]? | "[\(.type)] \(.message // "No message")"' 2>/dev/null | head -3)
    if [ -n "$conditions" ]; then
        log_warn "  │  └─ Conditions:"
        echo "$conditions" | while IFS= read -r line; do
            log_warn "  │     └─ $line"
        done
    else
        log_info "  │  └─ Message: ${message:0:150}"
    fi
}

# Dump full details for all pending apps (used on timeout)
dump_pending_apps_details() {
    local pending_apps="$1"

    log_error "============================================================================="
    log_error "DETAILED STATUS OF ALL PENDING APPLICATIONS"
    log_error "============================================================================="

    for app in $pending_apps; do
        log_error ""
        log_error "Application: $app"
        log_error "-----------------------------------------------------------------------------"

        local app_yaml
        app_yaml=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o yaml 2>/dev/null || echo "Failed to get app details")

        # Extract key fields
        local sync_status health_status repo_url target_revision
        sync_status=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o jsonpath='{.status.sync.status}' 2>/dev/null || echo "Unknown")
        health_status=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o jsonpath='{.status.health.status}' 2>/dev/null || echo "Unknown")
        repo_url=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o jsonpath='{.spec.source.repoURL}' 2>/dev/null || echo "Unknown")
        target_revision=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o jsonpath='{.spec.source.targetRevision}' 2>/dev/null || echo "Unknown")

        log_error "  Sync Status: $sync_status"
        log_error "  Health Status: $health_status"
        log_error "  Repository: $repo_url"
        log_error "  Target Revision: $target_revision"

        # Show all conditions
        log_error "  Conditions:"
        oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o json 2>/dev/null | \
            jq -r '.status.conditions[]? | "    - [\(.type)] \(.message)"' 2>/dev/null || log_error "    (none)"

        # Show degraded/failed resources
        log_error "  Unhealthy Resources:"
        oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o json 2>/dev/null | \
            jq -r '.status.resources[]? | select(.health.status != "Healthy" and .health.status != null) | "    - \(.kind)/\(.name): \(.health.status) - \(.health.message // "no message")"' 2>/dev/null | head -10 || log_error "    (none)"

        # Show operation state if any
        local operation_state
        operation_state=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io "$app" -o json 2>/dev/null | jq -r '.status.operationState.message // empty' 2>/dev/null)
        if [ -n "$operation_state" ]; then
            log_error "  Operation Message: ${operation_state:0:300}"
        fi
    done

    log_error "============================================================================="
}

# =============================================================================
# Helper Functions
# =============================================================================

print_help() {
    echo "Usage: $0 MODE [--obo] [--eaas] [-h|--help]"
    echo "  MODE             upstream/preview (default: upstream)"
    echo "  --obo        (only in preview mode) Install Observability operator and Prometheus instance for federation"
    echo "  --eaas       (only in preview mode) Install environment as a service components"
    echo
    echo "Example usage: \`$0 --obo --eaas"
}

# Patch ArgoCD application files to point to fork
update_patch_file() {
    local file=${1:?}
    yq -i ".[0].value = \"$MY_GIT_REPO_URL\"" "$file"
    yq -i ".[1].value = \"$PREVIEW_BRANCH\""  "$file"
}

# Label all cluster nodes for Konflux workloads
label_cluster_nodes() {
    log_step "Labeling cluster nodes for Konflux workloads"
    log_info "Reference: https://github.com/redhat-appstudio/infra-deployments/pull/4415"

    local nodes node_count labeled_count

    nodes=$(kubectl get nodes -o name)
    node_count=$(echo "$nodes" | wc -l | tr -d ' ')

    log_info "Found $node_count nodes to label with 'konflux-ci.dev/workload=konflux-tenants'"

    for node in $nodes; do
        log_substep "Labeling $node"
        if kubectl label $node konflux-ci.dev/workload=konflux-tenants --overwrite; then
            log_success "Successfully labeled $node"
        else
            log_error "Failed to label $node"
        fi
    done

    log_substep "Verifying labels on all nodes..."
    labeled_count=$(kubectl get nodes --show-labels | grep -c "konflux-ci.dev/workload=konflux-tenants" || echo "0")

    if [ "$node_count" -eq "$labeled_count" ]; then
        log_success "All $node_count nodes labeled and verified successfully"
    else
        log_error "Label verification failed: Expected $node_count labeled nodes, found $labeled_count"
        exit 1
    fi
}

# Filter applications based on DEPLOY_ONLY environment variable
configure_deploy_only() {
    [ -z "$DEPLOY_ONLY" ] && return

    log_step "Configuring selective deployment (DEPLOY_ONLY mode)"
    log_info "DEPLOY_ONLY is set, filtering applications to deploy only: $DEPLOY_ONLY"

    local applications deleted app
    local delete_file="$ROOT/argo-cd-apps/overlays/development/delete-applications.yaml"

    applications=$(oc kustomize argo-cd-apps/overlays/development | yq e --no-doc 'select(.kind == "ApplicationSet") | .metadata.name')
    deleted=$(yq e --no-doc .metadata.name "$delete_file")

    for app in $applications; do
        if ! grep -q "\b$app\b" <<< $DEPLOY_ONLY && ! grep -q "\b$app\b" <<< $deleted; then
            log_substep "Disabling ApplicationSet '$app' (not in DEPLOY_ONLY list)"
            echo '---' >> "$delete_file"
            yq e -n ".apiVersion=\"argoproj.io/v1alpha1\"
                     | .kind=\"ApplicationSet\"
                     | .metadata.name = \"$app\"
                     | .\$patch = \"delete\"" >> "$delete_file"
        fi
    done

    log_success "Selective deployment configured"
}

# Disable Kueue for OCP versions < 4.16
configure_kueue_for_ocp_version() {
    log_step "Checking OCP version for Kueue compatibility"

    local ocp_version ocp_minor delete_file

    ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
    ocp_minor=$(echo "$ocp_version" | cut -d. -f2)

    log_info "Detected OpenShift Container Platform version: $ocp_version (minor: $ocp_minor)"

    if [[ "$ocp_minor" -ge 16 ]]; then
        log_success "OCP version $ocp_version meets Kueue requirements - Kueue will be deployed"
        return
    fi

    log_warn "OCP version $ocp_version is below 4.16 - Kueue will be disabled"

    delete_file="$ROOT/argo-cd-apps/overlays/development/delete-applications.yaml"

    if ! grep -q "name: kueue" "$delete_file"; then
        log_substep "Adding Kueue to delete-applications.yaml"
        echo '---' >> "$delete_file"
        yq e -n ".apiVersion=\"argoproj.io/v1alpha1\"
                  | .kind=\"ApplicationSet\"
                  | .metadata.name = \"kueue\"
                  | .\$patch = \"delete\"" >> "$delete_file"
        log_success "Kueue ApplicationSet marked for deletion"
    else
        log_info "Kueue already exists in delete-applications.yaml, skipping"
    fi

    yq -i 'del(.resources[] | select(test("^kueue/?$")))' "$ROOT/components/policies/development/kustomization.yaml"
    log_success "Kueue disabled for OCP version $ocp_version"
}

# Apply service image overrides from environment variables
apply_service_image_overrides() {
    log_step "Applying service image overrides from environment variables"

    local has_overrides=false

    # Build Service
    if [ -n "${BUILD_SERVICE_IMAGE_REPO}" ] || [ -n "${BUILD_SERVICE_IMAGE_TAG}" ] || [ -n "${BUILD_SERVICE_PR_OWNER}" ]; then
        log_substep "Configuring Build Service overrides"
        [ -n "${BUILD_SERVICE_IMAGE_REPO}" ] && log_info "  - Image repo: ${BUILD_SERVICE_IMAGE_REPO}" && has_overrides=true
        [ -n "${BUILD_SERVICE_IMAGE_TAG}" ] && log_info "  - Image tag: ${BUILD_SERVICE_IMAGE_TAG}" && has_overrides=true
        [ -n "${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}" ] && log_info "  - Tag expiration: ${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}" && has_overrides=true
        [[ -n "${BUILD_SERVICE_PR_OWNER}" && -n "${BUILD_SERVICE_PR_SHA}" ]] && log_info "  - PR source: ${BUILD_SERVICE_PR_OWNER}@${BUILD_SERVICE_PR_SHA}" && has_overrides=true
    fi
    [ -n "${BUILD_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/build-service\")) |=.newName=\"${BUILD_SERVICE_IMAGE_REPO}\"" $ROOT/components/build-service/development/kustomization.yaml
    [ -n "${BUILD_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/build-service\")) |=.newTag=\"${BUILD_SERVICE_IMAGE_TAG}\"" $ROOT/components/build-service/development/kustomization.yaml
    [ -n "${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}" ] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"IMAGE_TAG_ON_PR_EXPIRATION\") | .value) |= \"${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}\"" $ROOT/components/build-service/development/image-expiration-patch.yaml
    [[ -n "${BUILD_SERVICE_PR_OWNER}" && "${BUILD_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/build-service*\")) |= \"https://github.com/${BUILD_SERVICE_PR_OWNER}/build-service/config/default?ref=${BUILD_SERVICE_PR_SHA}\"" $ROOT/components/build-service/development/kustomization.yaml

    # Application Service (HAS)
    if [ -n "${HAS_IMAGE_REPO}" ] || [ -n "${HAS_IMAGE_TAG}" ] || [ -n "${HAS_PR_OWNER}" ]; then
        log_substep "Configuring Application Service (HAS) overrides"
        [ -n "${HAS_IMAGE_REPO}" ] && log_info "  - Image repo: ${HAS_IMAGE_REPO}" && has_overrides=true
        [ -n "${HAS_IMAGE_TAG}" ] && log_info "  - Image tag: ${HAS_IMAGE_TAG}" && has_overrides=true
        [[ -n "${HAS_PR_OWNER}" && -n "${HAS_PR_SHA}" ]] && log_info "  - PR source: ${HAS_PR_OWNER}@${HAS_PR_SHA}" && has_overrides=true
    fi
    [ -n "${HAS_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newName=\"${HAS_IMAGE_REPO}\"" $ROOT/components/has/base/kustomization.yaml
    [ -n "${HAS_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/redhat-appstudio/application-service\")) |=.newTag=\"${HAS_IMAGE_TAG}\"" $ROOT/components/has/base/kustomization.yaml
    [[ -n "${HAS_PR_OWNER}" && "${HAS_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/application-service*\")) |= \"https://github.com/${HAS_PR_OWNER}/application-service/config/default?ref=${HAS_PR_SHA}\"" $ROOT/components/has/base/kustomization.yaml

    # Integration Service
    if [ -n "${INTEGRATION_SERVICE_IMAGE_REPO}" ] || [ -n "${INTEGRATION_SERVICE_IMAGE_TAG}" ] || [ -n "${INTEGRATION_SERVICE_PR_OWNER}" ]; then
        log_substep "Configuring Integration Service overrides"
        [ -n "${INTEGRATION_SERVICE_IMAGE_REPO}" ] && log_info "  - Image repo: ${INTEGRATION_SERVICE_IMAGE_REPO}" && has_overrides=true
        [ -n "${INTEGRATION_SERVICE_IMAGE_TAG}" ] && log_info "  - Image tag: ${INTEGRATION_SERVICE_IMAGE_TAG}" && has_overrides=true
        [[ -n "${INTEGRATION_SERVICE_PR_OWNER}" && -n "${INTEGRATION_SERVICE_PR_SHA}" ]] && log_info "  - PR source: ${INTEGRATION_SERVICE_PR_OWNER}@${INTEGRATION_SERVICE_PR_SHA}" && has_overrides=true
    fi
    [ -n "${INTEGRATION_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/integration-service\")) |=.newName=\"${INTEGRATION_SERVICE_IMAGE_REPO}\"" $ROOT/components/integration/development/kustomization.yaml
    [ -n "${INTEGRATION_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/integration-service\")) |=.newTag=\"${INTEGRATION_SERVICE_IMAGE_TAG}\"" $ROOT/components/integration/development/kustomization.yaml
    [[ -n "${INTEGRATION_SERVICE_PR_OWNER}" && "${INTEGRATION_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/integration-service*\")) |= (sub(\"\?ref=.+\", \"?ref=${INTEGRATION_SERVICE_PR_SHA}\" ) | sub(\"github.com/redhat-appstudio\", \"github.com/${INTEGRATION_SERVICE_PR_OWNER}\"))" $ROOT/components/integration/development/kustomization.yaml

    # Release Service
    if [ -n "${RELEASE_SERVICE_IMAGE_REPO}" ] || [ -n "${RELEASE_SERVICE_IMAGE_TAG}" ] || [ -n "${RELEASE_SERVICE_PR_OWNER}" ]; then
        log_substep "Configuring Release Service overrides"
        [ -n "${RELEASE_SERVICE_IMAGE_REPO}" ] && log_info "  - Image repo: ${RELEASE_SERVICE_IMAGE_REPO}" && has_overrides=true
        [ -n "${RELEASE_SERVICE_IMAGE_TAG}" ] && log_info "  - Image tag: ${RELEASE_SERVICE_IMAGE_TAG}" && has_overrides=true
        [[ -n "${RELEASE_SERVICE_PR_OWNER}" && -n "${RELEASE_SERVICE_PR_SHA}" ]] && log_info "  - PR source: ${RELEASE_SERVICE_PR_OWNER}@${RELEASE_SERVICE_PR_SHA}" && has_overrides=true
    fi
    [ -n "${RELEASE_SERVICE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/release-service\")) |=.newName=\"${RELEASE_SERVICE_IMAGE_REPO}\"" $ROOT/components/release/development/kustomization.yaml
    [ -n "${RELEASE_SERVICE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/release-service\")) |=.newTag=\"${RELEASE_SERVICE_IMAGE_TAG}\"" $ROOT/components/release/development/kustomization.yaml
    [[ -n "${RELEASE_SERVICE_PR_OWNER}" && "${RELEASE_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/release-service*\")) |= \"https://github.com/${RELEASE_SERVICE_PR_OWNER}/release-service/config/default?ref=${RELEASE_SERVICE_PR_SHA}\"" $ROOT/components/release/development/kustomization.yaml

    # Mintmaker
    if [ -n "${MINTMAKER_IMAGE_REPO}" ] || [ -n "${MINTMAKER_IMAGE_TAG}" ] || [ -n "${MINTMAKER_PR_OWNER}" ]; then
        log_substep "Configuring Mintmaker overrides"
        [ -n "${MINTMAKER_IMAGE_REPO}" ] && log_info "  - Image repo: ${MINTMAKER_IMAGE_REPO}" && has_overrides=true
        [ -n "${MINTMAKER_IMAGE_TAG}" ] && log_info "  - Image tag: ${MINTMAKER_IMAGE_TAG}" && has_overrides=true
        [[ -n "${MINTMAKER_PR_OWNER}" && -n "${MINTMAKER_PR_SHA}" ]] && log_info "  - PR source: ${MINTMAKER_PR_OWNER}@${MINTMAKER_PR_SHA}" && has_overrides=true
    fi
    [ -n "${MINTMAKER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker\")) |=.newName=\"${MINTMAKER_IMAGE_REPO}\"" $ROOT/components/mintmaker/development/kustomization.yaml
    [ -n "${MINTMAKER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker\")) |=.newTag=\"${MINTMAKER_IMAGE_TAG}\"" $ROOT/components/mintmaker/development/kustomization.yaml
    [[ -n "${MINTMAKER_PR_OWNER}" && "${MINTMAKER_PR_SHA}" ]] && yq -i "(.resources[] | select(contains(\"konflux-ci/mintmaker\"))) |= (sub(\"konflux-ci/mintmaker\", \"${MINTMAKER_PR_OWNER}/mintmaker\") | sub(\"ref=.*\", \"ref=${MINTMAKER_PR_SHA}\"))" $ROOT/components/mintmaker/development/kustomization.yaml

    # Mintmaker Renovate
    if [ -n "${MINTMAKER_RENOVATE_IMAGE_REPO}" ] || [ -n "${MINTMAKER_RENOVATE_IMAGE_TAG}" ]; then
        log_substep "Configuring Mintmaker Renovate overrides"
        [ -n "${MINTMAKER_RENOVATE_IMAGE_REPO}" ] && log_info "  - Image repo: ${MINTMAKER_RENOVATE_IMAGE_REPO}" && has_overrides=true
        [ -n "${MINTMAKER_RENOVATE_IMAGE_TAG}" ] && log_info "  - Image tag: ${MINTMAKER_RENOVATE_IMAGE_TAG}" && has_overrides=true
    fi
    [ -n "${MINTMAKER_RENOVATE_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker-renovate-image\")) |=.newName=\"${MINTMAKER_RENOVATE_IMAGE_REPO}\"" $ROOT/components/mintmaker/development/kustomization.yaml
    [ -n "${MINTMAKER_RENOVATE_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/mintmaker-renovate-image\")) |=.newTag=\"${MINTMAKER_RENOVATE_IMAGE_TAG}\"" $ROOT/components/mintmaker/development/kustomization.yaml

    # Image Controller
    if [ -n "${IMAGE_CONTROLLER_IMAGE_REPO}" ] || [ -n "${IMAGE_CONTROLLER_IMAGE_TAG}" ] || [ -n "${IMAGE_CONTROLLER_PR_OWNER}" ]; then
        log_substep "Configuring Image Controller overrides"
        [ -n "${IMAGE_CONTROLLER_IMAGE_REPO}" ] && log_info "  - Image repo: ${IMAGE_CONTROLLER_IMAGE_REPO}" && has_overrides=true
        [ -n "${IMAGE_CONTROLLER_IMAGE_TAG}" ] && log_info "  - Image tag: ${IMAGE_CONTROLLER_IMAGE_TAG}" && has_overrides=true
        [[ -n "${IMAGE_CONTROLLER_PR_OWNER}" && -n "${IMAGE_CONTROLLER_PR_SHA}" ]] && log_info "  - PR source: ${IMAGE_CONTROLLER_PR_OWNER}@${IMAGE_CONTROLLER_PR_SHA}" && has_overrides=true
    fi
    [ -n "${IMAGE_CONTROLLER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/image-controller\")) |=.newName=\"${IMAGE_CONTROLLER_IMAGE_REPO}\"" $ROOT/components/image-controller/development/kustomization.yaml
    [ -n "${IMAGE_CONTROLLER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"quay.io/konflux-ci/image-controller\")) |=.newTag=\"${IMAGE_CONTROLLER_IMAGE_TAG}\"" $ROOT/components/image-controller/development/kustomization.yaml
    [[ -n "${IMAGE_CONTROLLER_PR_OWNER}" && "${IMAGE_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/image-controller*\")) |= \"https://github.com/${IMAGE_CONTROLLER_PR_OWNER}/image-controller/config/default?ref=${IMAGE_CONTROLLER_PR_SHA}\"" $ROOT/components/image-controller/development/kustomization.yaml

    # Multi-Arch Controller
    if [ -n "${MULTI_ARCH_CONTROLLER_IMAGE_REPO}" ] || [ -n "${MULTI_ARCH_CONTROLLER_IMAGE_TAG}" ] || [ -n "${MULTI_ARCH_CONTROLLER_PR_OWNER}" ]; then
        log_substep "Configuring Multi-Arch Controller overrides"
        [ -n "${MULTI_ARCH_CONTROLLER_IMAGE_REPO}" ] && log_info "  - Image repo: ${MULTI_ARCH_CONTROLLER_IMAGE_REPO}" && has_overrides=true
        [ -n "${MULTI_ARCH_CONTROLLER_IMAGE_TAG}" ] && log_info "  - Image tag: ${MULTI_ARCH_CONTROLLER_IMAGE_TAG}" && has_overrides=true
        [[ -n "${MULTI_ARCH_CONTROLLER_PR_OWNER}" && -n "${MULTI_ARCH_CONTROLLER_PR_SHA}" ]] && log_info "  - PR source: ${MULTI_ARCH_CONTROLLER_PR_OWNER}@${MULTI_ARCH_CONTROLLER_PR_SHA}" && has_overrides=true
    fi
    [ -n "${MULTI_ARCH_CONTROLLER_IMAGE_REPO}" ] && yq -i e "(.images.[] | select(.name==\"multi-platform-controller\")) |=.newName=\"${MULTI_ARCH_CONTROLLER_IMAGE_REPO}\"" $ROOT/components/multi-platform-controller/base/kustomization.yaml
    [ -n "${MULTI_ARCH_CONTROLLER_IMAGE_TAG}" ] && yq -i e "(.images.[] | select(.name==\"multi-platform-controller\")) |=.newTag=\"${MULTI_ARCH_CONTROLLER_IMAGE_TAG}\"" $ROOT/components/multi-platform-controller/base/kustomization.yaml
    [[ -n "${MULTI_ARCH_CONTROLLER_PR_OWNER}" && "${MULTI_ARCH_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/multi-platform-controller*\")) |= (sub(\"\?ref=.+\", \"?ref=${MULTI_ARCH_CONTROLLER_PR_SHA}\" ) | sub(\"github.com/konflux-ci\", \"github.com/${MULTI_ARCH_CONTROLLER_PR_OWNER}\"))" $ROOT/components/multi-platform-controller/base/kustomization.yaml

    # EaaS Hypershift
    if [ -n "${EAAS_HYPERSHIFT_BASE_DOMAIN}" ] || [ -n "${EAAS_HYPERSHIFT_CLI_ROLE_ARN}" ]; then
        log_substep "Configuring EaaS Hypershift overrides"
        [ -n "${EAAS_HYPERSHIFT_BASE_DOMAIN}" ] && log_info "  - Base domain: ${EAAS_HYPERSHIFT_BASE_DOMAIN}" && has_overrides=true
        [ -n "${EAAS_HYPERSHIFT_CLI_ROLE_ARN}" ] && log_info "  - CLI Role ARN: ${EAAS_HYPERSHIFT_CLI_ROLE_ARN}" && has_overrides=true
    fi
    [ -n "${EAAS_HYPERSHIFT_BASE_DOMAIN}" ] && yq -i e "(.[] | select(.value.name==\"baseDomain\")).value.value |= \"${EAAS_HYPERSHIFT_BASE_DOMAIN}\"" $ROOT/components/cluster-as-a-service/development/add-hypershift-params.yaml
    [ -n "${EAAS_HYPERSHIFT_CLI_ROLE_ARN}" ] && yq -i e "(.[] | select(.value.name==\"hypershiftRoleArn\")).value.value |= \"${EAAS_HYPERSHIFT_CLI_ROLE_ARN}\"" $ROOT/components/cluster-as-a-service/development/add-hypershift-params.yaml

    # Pipeline Service
    if [[ -n "${PIPELINE_PR_OWNER}" && -n "${PIPELINE_PR_SHA}" ]]; then
        log_substep "Configuring Pipeline Service overrides"
        log_info "  - PR source: ${PIPELINE_PR_OWNER}@${PIPELINE_PR_SHA}"
        has_overrides=true
        yq -i e ".resources[] |= sub(\"ref=[^ ]*\"; \"ref=${PIPELINE_PR_SHA}\") | .resources[] |= sub(\"openshift-pipelines\"; \"${PIPELINE_PR_OWNER}\")" $ROOT/components/pipeline-service/development/kustomization.yaml
    fi

    if [ "$has_overrides" = false ]; then
        log_info "No service image overrides configured - using default images"
    else
        log_success "Service image overrides applied"
    fi
}

# Deploy ArgoCD applications and wait for sync
deploy_and_wait_for_argocd() {
    log_step "Deploying ArgoCD applications"

    local apps app state not_done unknown error
    local total_apps synced_apps pending_apps iteration=0

    # Create the root Application
    log_substep "Applying root Application from: $ROOT/argo-cd-apps/app-of-app-sets/development"
    oc apply -k $ROOT/argo-cd-apps/app-of-app-sets/development
    log_success "Root Application 'all-application-sets' created"

    # Wait for root application to sync
    log_substep "Waiting for 'all-application-sets' to become Healthy and Synced"
    local root_wait=0
    while true; do
        local root_status
        root_status=$(oc get applications.argoproj.io all-application-sets -n $ARGOCD_NAMESPACE -o jsonpath='{.status.health.status} {.status.sync.status}')

        if [ "$root_status" == "Healthy Synced" ]; then
            break
        fi

        root_wait=$((root_wait + 5))
        log_wait "Root application status: '$root_status' (target: 'Healthy Synced') - ${root_wait}s elapsed"
        sleep 5
    done
    log_success "Root application 'all-application-sets' is Healthy and Synced"

    # Trigger hard refresh of all apps
    log_substep "Triggering hard refresh on all ArgoCD applications"
    apps=$(oc get apps -n $ARGOCD_NAMESPACE -o name)
    total_apps=$(echo "$apps" | wc -l | tr -d ' ')
    log_info "Found $total_apps applications to refresh"

    for app in $apps; do
        oc patch $app -n $ARGOCD_NAMESPACE --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "hard"}}}' &
    done
    wait
    log_success "Hard refresh triggered on all $total_apps applications"

    # Wait for refresh to complete
    log_substep "Waiting for refresh operations to complete"
    local refresh_wait=0
    while true; do
        local refresh_pending
        refresh_pending=$(oc get applications.argoproj.io -n $ARGOCD_NAMESPACE -o json 2>/dev/null | jq '[.items[] | select(.metadata.annotations["argocd.argoproj.io/refresh"] != null)] | length' 2>/dev/null || echo "0")

        if [ "$refresh_pending" -eq 0 ] || [ -z "$refresh_pending" ]; then
            break
        fi

        refresh_wait=$((refresh_wait + 5))
        local refresh_done=$((total_apps - refresh_pending))
        log_progress "Refresh: $refresh_done/$total_apps complete | $refresh_pending still refreshing (${refresh_wait}s elapsed)"
        sleep 5
    done
    log_success "All $total_apps applications refreshed"

    # Wait for all applications to sync and become healthy
    log_step "Waiting for all ArgoCD applications to sync and become healthy"
    log_info "Timeout: $MAX_SYNC_TIMEOUT seconds ($((MAX_SYNC_TIMEOUT / 60)) minutes)"

    local sync_start_time last_detailed_status_time elapsed_time
    sync_start_time=$(date +%s)
    last_detailed_status_time=$sync_start_time

    while :; do
        iteration=$((iteration + 1))
        local current_time=$(date +%s)
        elapsed_time=$((current_time - sync_start_time))
        local time_since_detailed=$((current_time - last_detailed_status_time))

        # Check for timeout
        if [ "$elapsed_time" -ge "$MAX_SYNC_TIMEOUT" ]; then
            log_error "============================================================================="
            log_error "TIMEOUT: Applications failed to sync within $((MAX_SYNC_TIMEOUT / 60)) minutes"
            log_error "============================================================================="

            local pending_app_names
            pending_app_names=$(echo "$not_done" | awk '{print $1}')
            dump_pending_apps_details "$pending_app_names"

            TOTAL_APPS_DEPLOYED=$synced_apps
            FAILED_APPS=$pending_app_names
            print_execution_summary "failed" "ARGOCD_SYNC_TIMEOUT: $pending_apps apps failed to sync"
            exit 1
        fi

        state=$(oc get apps -n $ARGOCD_NAMESPACE --no-headers 2>/dev/null || echo "")
        total_apps=$(echo "$state" | grep -c "." || echo "0")
        synced_apps=$(echo "$state" | grep -c "Synced[[:blank:]]*Healthy" || echo "0")
        pending_apps=$((total_apps - synced_apps))
        not_done=$(echo "$state" | grep -v "Synced[[:blank:]]*Healthy" || true)

        local elapsed_min=$((elapsed_time / 60))
        local elapsed_sec=$((elapsed_time % 60))
        log_progress "Applications: $synced_apps/$total_apps ready | $pending_apps pending (${elapsed_min}m ${elapsed_sec}s elapsed)"

        if [ -z "$not_done" ]; then
            log_success "All $total_apps ArgoCD applications are Synced and Healthy in ${elapsed_min}m ${elapsed_sec}s"
            TOTAL_APPS_DEPLOYED=$total_apps
            break
        fi

        # Show pending application names (compact)
        local pending_names
        pending_names=$(echo "$not_done" | awk '{print $1}' | tr '\n' ', ' | sed 's/,$//')
        log_info "Pending: $pending_names"

        # Show detailed status every DETAILED_STATUS_INTERVAL seconds
        if [ "$time_since_detailed" -ge "$DETAILED_STATUS_INTERVAL" ]; then
            log_substep "Detailed status of pending applications:"
            for app in $(echo "$not_done" | awk '{print $1}'); do
                show_app_details "$app"
            done
            last_detailed_status_time=$current_time
        fi

        unknown=$(echo "$not_done" | grep Unknown | grep -v Progressing | cut -f1 -d ' ') || :
        if [ -n "$unknown" ]; then
            log_warn "Found applications in Unknown state (not Progressing), investigating..."

            for app in $unknown; do
                error=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io $app -o jsonpath='{.status.conditions}' 2>/dev/null || echo "")

                if echo "$error" | grep -q 'context deadline exceeded'; then
                    log_warn "Application '$app' hit context deadline, attempting soft refresh"
                    oc patch applications.argoproj.io $app -n $ARGOCD_NAMESPACE --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "soft"}}}' 2>/dev/null || true

                    local refresh_wait=0
                    while [ -n "$(oc get applications.argoproj.io -n $ARGOCD_NAMESPACE $app -o jsonpath='{.metadata.annotations.argocd\.argoproj\.io/refresh}' 2>/dev/null)" ]; do
                        refresh_wait=$((refresh_wait + 5))
                        if [ "$refresh_wait" -gt 60 ]; then
                            log_warn "Soft refresh of '$app' timed out after 60s, continuing anyway"
                            break
                        fi
                        sleep 5
                    done
                    log_success "Soft refresh of '$app' completed, continuing sync check"
                    continue 2
                fi

                # Show detailed error for this app
                log_error "Application '$app' is in Unknown state without 'context deadline exceeded'"
                show_app_details "$app"
            done
        fi

        log_wait "Waiting $SYNC_INTERVAL seconds before next sync check..."
        sleep $SYNC_INTERVAL
    done
}

# Wait for Tekton components to be ready
wait_for_tekton_ready() {
    log_step "Waiting for Tekton components to be ready"
    log_info "Reference: https://tekton.dev/docs/operator/tektonconfig/#tekton-config"
    log_info "Timeout: $MAX_TEKTON_READY_TIMEOUT seconds ($((MAX_TEKTON_READY_TIMEOUT / 60)) minutes)"

    local state msg iteration=0
    local tekton_start_time elapsed_time
    tekton_start_time=$(date +%s)

    while :; do
        iteration=$((iteration + 1))
        local current_time=$(date +%s)
        elapsed_time=$((current_time - tekton_start_time))
        local elapsed_min=$((elapsed_time / 60))
        local elapsed_sec=$((elapsed_time % 60))

        # Check for timeout
        if [ "$elapsed_time" -ge "$MAX_TEKTON_READY_TIMEOUT" ]; then
            log_error "============================================================================="
            log_error "TIMEOUT: Tekton components failed to become ready within $((MAX_TEKTON_READY_TIMEOUT / 60)) minutes"
            log_error "============================================================================="
            
            # Dump Tekton status for debugging
            log_error "TektonConfig status:"
            oc get tektonconfig config -o yaml 2>/dev/null | head -100 || log_error "  (failed to get tektonconfig)"
            
            log_error ""
            log_error "Tekton operator pods status:"
            oc get pods -n openshift-operators -l app=openshift-pipelines-operator 2>/dev/null || log_error "  (failed to get operator pods)"
            
            log_error ""
            log_error "Tekton namespace pods status:"
            oc get pods -n $PIPELINES_NAMESPACE 2>/dev/null || log_error "  (failed to get pipelines namespace pods)"
            
            FAILED_APPS="tekton-operator"
            print_execution_summary "failed" "TEKTON_READY_TIMEOUT"
            exit 1
        fi

        state=$(oc get tektonconfig config -o json 2>/dev/null | jq -r '.status.conditions[] | select(.type == "Ready")' 2>/dev/null || echo "{}")
        status_value=$(jq -r '.status // "Unknown"' <<< "$state")

        log_progress "Tekton readiness check iteration $iteration: status=$status_value (${elapsed_min}m ${elapsed_sec}s elapsed)"

        if [ "$status_value" == "True" ]; then
            log_success "All Tekton components are installed and ready in ${elapsed_min}m ${elapsed_sec}s"
            break
        fi

        msg=$(jq -r '.message // "No message available"' <<< "$state")
        log_warn "Tekton not ready: $msg"

        # Workaround for https://issues.redhat.com/browse/SRVKP-3245
        if echo "$msg" | grep -q 'Components not in ready state: OpenShiftPipelinesAsCode: reconcile again and proceed'; then
            log_info "Detected SRVKP-3245 condition, checking Pipelines-as-Code deployments directly"

            local pac_controller pac_watcher pac_webhook
            pac_controller=$(oc get deployment/pipelines-as-code-controller -n $PIPELINES_NAMESPACE -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "Unknown")
            pac_watcher=$(oc get deployment/pipelines-as-code-watcher -n $PIPELINES_NAMESPACE -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "Unknown")
            pac_webhook=$(oc get deployment/pipelines-as-code-webhook -n $PIPELINES_NAMESPACE -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "Unknown")

            log_debug "  - pipelines-as-code-controller: $pac_controller"
            log_debug "  - pipelines-as-code-watcher: $pac_watcher"
            log_debug "  - pipelines-as-code-webhook: $pac_webhook"

            if [[ "$pac_controller" != "True" ]]; then
                log_wait "pipelines-as-code-controller not yet available"
                sleep $SYNC_INTERVAL
                continue
            fi
            if [[ "$pac_watcher" != "True" ]]; then
                log_wait "pipelines-as-code-watcher not yet available"
                sleep $SYNC_INTERVAL
                continue
            fi
            if [[ "$pac_webhook" != "True" ]]; then
                log_wait "pipelines-as-code-webhook not yet available"
                sleep $SYNC_INTERVAL
                continue
            fi

            log_warn "BYPASSING tektonconfig ready check due to SRVKP-3245"
            log_info "All Pipelines-as-Code components are available, proceeding despite tektonconfig status"
            log_info "Reference: https://issues.redhat.com/browse/SRVKP-3245"
            break
        fi

        sleep $SYNC_INTERVAL
    done
}

# Wait for Tekton CRDs to be available
wait_for_tekton_crds() {
    log_step "Waiting for Tekton CRDs to be available"

    local retry=0 tekton_crds
    local required_crds=("pipelines" "tasks" "pipelineruns" "taskruns")

    while true; do
        retry=$((retry + 1))
        log_progress "Tekton CRD check attempt $retry/$MAX_TEKTON_CRD_RETRIES"

        if [ "$retry" -gt "$MAX_TEKTON_CRD_RETRIES" ]; then
            log_error "Tekton CRDs are not available after $MAX_TEKTON_CRD_RETRIES attempts"
            log_error "Required CRDs: ${required_crds[*]}"
            log_error "This may indicate that the Tekton operator failed to install properly"
            FAILED_APPS="tekton-crds"
            print_execution_summary "failed" "TEKTON_CRD_UNAVAILABLE: Required CRDs not found after $MAX_TEKTON_CRD_RETRIES attempts"
            exit 1
        fi

        tekton_crds=$(oc api-resources --api-group="tekton.dev" --no-headers 2>/dev/null || echo "")

        if [[ $tekton_crds =~ pipelines && $tekton_crds =~ tasks && $tekton_crds =~ pipelineruns && $tekton_crds =~ taskruns ]]; then
            log_success "All required Tekton CRDs are available: ${required_crds[*]}"
            break
        fi

        log_wait "Some Tekton CRDs not yet available, waiting $SYNC_INTERVAL seconds..."
        sleep $SYNC_INTERVAL
    done
}

# =============================================================================
# Argument Parsing
# =============================================================================
OBO=false
EAAS=false

while [[ $# -gt 0 ]]; do
    key=$1
    case $key in
        --obo)
            OBO=true
            shift
            ;;
        --eaas)
            EAAS=true
            shift
            ;;
        -h|--help)
            print_help
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

# =============================================================================
# Main Execution
# =============================================================================

log_step "Starting Konflux Preview Environment Setup"
log_info "Script: $0"
log_info "Options: OBO=$OBO, EAAS=$EAAS"
log_info "Start time: $(date '+%Y-%m-%d %H:%M:%S %Z')"

# =============================================================================
# Cluster Context (for LLM understanding)
# =============================================================================
print_cluster_context

# Store OCP version globally for summary
OCP_VERSION=$(oc get clusterversion version -o jsonpath='{.status.desired.version}' 2>/dev/null || echo "Unknown")

# =============================================================================
# Git Setup - INLINE (not in function) as per original script
# =============================================================================
if [ -f $ROOT/hack/preview.env ]; then
    source $ROOT/hack/preview.env
fi

if [ -z "$MY_GIT_FORK_REMOTE" ]; then
    log_error "MY_GIT_FORK_REMOTE environment variable is not set"
    log_error "ACTION REQUIRED: Set MY_GIT_FORK_REMOTE to the name of your fork remote (e.g., 'origin' or 'fork')"
    exit 1
fi

MY_GIT_REPO_URL=$(git ls-remote --get-url $MY_GIT_FORK_REMOTE | sed 's|^git@github.com:|https://github.com/|')
MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
trap "git checkout $MY_GIT_BRANCH" EXIT


if echo "$MY_GIT_REPO_URL" | grep -q redhat-appstudio/infra-deployments; then
    log_error "Cannot use the upstream repository (redhat-appstudio/infra-deployments) for preview"
    log_error "ACTION REQUIRED: Fork the repository and use your fork's remote"
    log_error "Current MY_GIT_REPO_URL: $MY_GIT_REPO_URL"
    exit 1
fi

# Do not allow to use default github org
if [ -z "$MY_GITHUB_ORG" ] || [ "$MY_GITHUB_ORG" == "redhat-appstudio-appdata" ]; then
    log_error "MY_GITHUB_ORG environment variable is not set or is using the default value"
    log_error "ACTION REQUIRED: Set MY_GITHUB_ORG to your GitHub organization name"
    log_error "Current value: '${MY_GITHUB_ORG:-<not set>}'"
    log_error "Cannot use 'redhat-appstudio-appdata' (reserved for production)"
    exit 1
fi

if ! git diff --exit-code --quiet; then
    log_error "Uncommitted changes detected in Git working tree"
    log_error "ACTION REQUIRED: Commit or stash your changes before running preview"
    log_error "Run 'git status' to see pending changes"
    exit 1
fi

# Create preview branch for preview configuration
PREVIEW_BRANCH=preview-${MY_GIT_BRANCH}${TEST_BRANCH_ID+-$TEST_BRANCH_ID}
if git rev-parse --verify $PREVIEW_BRANCH &> /dev/null; then
    git branch -D $PREVIEW_BRANCH
fi
git checkout -b $PREVIEW_BRANCH

log_success "Git environment initialized"
log_info "  - Repository URL: $MY_GIT_REPO_URL"
log_info "  - Source branch: $MY_GIT_BRANCH"
log_info "  - Preview branch: $PREVIEW_BRANCH"
log_info "  - GitHub org: $MY_GITHUB_ORG"

# =============================================================================
# Patch ArgoCD Applications
# =============================================================================
log_step "Patching ArgoCD application manifests to use fork repository"
log_info "Setting repoURL to: $MY_GIT_REPO_URL"
log_info "Setting targetRevision to: $PREVIEW_BRANCH"

update_patch_file "${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/application-patch.yaml"
log_substep "Patched: application-patch.yaml"

update_patch_file "${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/application-set-patch.yaml"
log_substep "Patched: application-set-patch.yaml"

update_patch_file "${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/application-set-multisrc-src-1-patch.yaml"
log_substep "Patched: application-set-multisrc-src-1-patch.yaml"

log_success "All ArgoCD patch files updated"

# =============================================================================
# Optional Components
# =============================================================================
if $OBO; then
    log_step "Enabling Observability (OBO) components"
    log_info "Adding Observability operator and Prometheus for federation"
    yq -i '.resources += ["monitoringstack/"]' $ROOT/components/monitoring/prometheus/development/kustomization.yaml
    log_success "Observability components enabled"
fi

if $EAAS; then
    log_step "Enabling Environment-as-a-Service (EaaS) components"
    log_info "Enabling EaaS cluster role assignment"
    yq -i '.components += ["../../../k-components/assign-eaas-role-to-local-cluster"]' \
        $ROOT/argo-cd-apps/base/local-cluster-secret/all-in-one/kustomization.yaml
    log_success "EaaS components enabled"
fi

# =============================================================================
# Cluster Configuration
# =============================================================================
label_cluster_nodes

configure_deploy_only
configure_kueue_for_ocp_version

# Configure GitHub org
log_step "Configuring GitHub organization"
log_info "Setting GitHub org to: $MY_GITHUB_ORG"
$ROOT/hack/util-set-github-org $MY_GITHUB_ORG
log_success "GitHub organization configured"

# Configure Rekor server hostname
log_step "Configuring Rekor server hostname"
domain=$(oc get ingresses.config.openshift.io cluster --template={{.spec.domain}})
rekor_server="rekor.$domain"
log_info "Cluster domain: $domain"
log_info "Rekor server hostname: $rekor_server"
sed -i.bak "s/rekor-server.enterprise-contract-service.svc/$rekor_server/" $ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml && rm $ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml.bak
log_success "Rekor server hostname configured"

# =============================================================================
# Service Image Overrides
# =============================================================================
apply_service_image_overrides

# =============================================================================
# Commit and Push - INLINE (not in function) as per original script
# =============================================================================
log_step "Committing and pushing preview changes"
if ! git diff --exit-code --quiet; then
    git commit -a -m "Preview mode, do not merge into main"
    git push -f --set-upstream $MY_GIT_FORK_REMOTE $PREVIEW_BRANCH
    log_success "Preview changes committed and pushed to $MY_GIT_FORK_REMOTE/$PREVIEW_BRANCH"
else
    log_info "No changes to commit"
fi

# =============================================================================
# Deploy Applications
# =============================================================================
deploy_and_wait_for_argocd

# =============================================================================
# Wait for Tekton
# =============================================================================
wait_for_tekton_ready
wait_for_tekton_crds

# =============================================================================
# Final Configuration
# =============================================================================
log_step "Configuring Pipelines as Code integration"
$ROOT/hack/build/setup-pac-integration.sh
log_success "Pipelines as Code configured"

# =============================================================================
# Complete
# =============================================================================
log_step "Preview Environment Setup Complete"
log_success "Konflux preview environment is ready!"
log_info "  - Fork: $MY_GIT_REPO_URL"
log_info "  - Branch: $PREVIEW_BRANCH"
log_info "  - GitHub Org: $MY_GITHUB_ORG"
log_info "  - OpenShift Version: $OCP_VERSION"
log_info "  - End time: $(date '+%Y-%m-%d %H:%M:%S %Z')"

# Print execution summary for LLM parsing
print_execution_summary "success"
