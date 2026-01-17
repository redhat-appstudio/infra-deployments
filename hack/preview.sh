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

    while :; do
        iteration=$((iteration + 1))
        state=$(oc get apps -n $ARGOCD_NAMESPACE --no-headers)
        total_apps=$(echo "$state" | wc -l | tr -d ' ')
        synced_apps=$(echo "$state" | grep -c "Synced[[:blank:]]*Healthy" || echo "0")
        pending_apps=$((total_apps - synced_apps))
        not_done=$(echo "$state" | grep -v "Synced[[:blank:]]*Healthy" || true)

        log_progress "Applications: $synced_apps/$total_apps ready | $pending_apps pending"

        if [ -z "$not_done" ]; then
            log_success "All $total_apps ArgoCD applications are Synced and Healthy"
            break
        fi

        # Show pending application names (compact)
        local pending_names
        pending_names=$(echo "$not_done" | awk '{print $1}' | tr '\n' ', ' | sed 's/,$//')
        log_info "Pending: $pending_names"

        unknown=$(echo "$not_done" | grep Unknown | grep -v Progressing | cut -f1 -d ' ') || :
        if [ -n "$unknown" ]; then
            log_warn "Found applications in Unknown state (not Progressing), investigating..."

            for app in $unknown; do
                error=$(oc get -n $ARGOCD_NAMESPACE applications.argoproj.io $app -o jsonpath='{.status.conditions}')

                if echo "$error" | grep -q 'context deadline exceeded'; then
                    log_warn "Application '$app' hit context deadline, attempting soft refresh"
                    oc patch applications.argoproj.io $app -n $ARGOCD_NAMESPACE --type merge -p='{"metadata": {"annotations":{"argocd.argoproj.io/refresh": "soft"}}}'

                    while [ -n "$(oc get applications.argoproj.io -n $ARGOCD_NAMESPACE $app -o jsonpath='{.metadata.annotations.argocd\.argoproj\.io/refresh}')" ]; do
                        sleep 5
                    done
                    log_success "Soft refresh of '$app' completed, continuing sync check"
                    continue 2
                fi

                log_error "Application '$app' failed with unrecoverable error"
                log_error "============ ERROR DETAILS FOR '$app' ============"
                if [ -n "$error" ]; then
                    log_error "Conditions: $error"
                else
                    log_error "Full application state:"
                    oc get -n $ARGOCD_NAMESPACE applications.argoproj.io $app -o yaml >&2
                fi
                log_error "=================================================="
            done
            log_error "One or more applications failed to sync. See error details above."
            exit 1
        fi

        log_wait "Waiting $SYNC_INTERVAL seconds before next sync check..."
        sleep $SYNC_INTERVAL
    done
}

# Wait for Tekton components to be ready
wait_for_tekton_ready() {
    log_step "Waiting for Tekton components to be ready"
    log_info "Reference: https://tekton.dev/docs/operator/tektonconfig/#tekton-config"

    local state msg iteration=0

    while :; do
        iteration=$((iteration + 1))
        state=$(oc get tektonconfig config -o json | jq -r '.status.conditions[] | select(.type == "Ready")')
        status_value=$(jq -r '.status' <<< "$state")

        log_progress "Tekton readiness check iteration $iteration: status=$status_value"

        if [ "$status_value" == "True" ]; then
            log_success "All Tekton components are installed and ready"
            break
        fi

        msg=$(jq -r '.message' <<< "$state")
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

# =============================================================================
# Git Setup - INLINE (not in function) as per original script
# =============================================================================
if [ -f $ROOT/hack/preview.env ]; then
    source $ROOT/hack/preview.env
fi

if [ -z "$MY_GIT_FORK_REMOTE" ]; then
    echo "Set MY_GIT_FORK_REMOTE environment to name of your fork remote"
    exit 1
fi

MY_GIT_REPO_URL=$(git ls-remote --get-url $MY_GIT_FORK_REMOTE | sed 's|^git@github.com:|https://github.com/|')
MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
trap "git checkout $MY_GIT_BRANCH" EXIT


if echo "$MY_GIT_REPO_URL" | grep -q redhat-appstudio/infra-deployments; then
    echo "Use your fork repository for preview"
    exit 1
fi

# Do not allow to use default github org
if [ -z "$MY_GITHUB_ORG" ] || [ "$MY_GITHUB_ORG" == "redhat-appstudio-appdata" ]; then
    echo "Set MY_GITHUB_ORG environment variable"
    exit 1
fi

if ! git diff --exit-code --quiet; then
    echo "Changes in working Git working tree, commit them or stash them"
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
