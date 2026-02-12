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

# =============================================================================
# Main
# =============================================================================

main() {
    local mode obo eaas
    local start_time end_time total_time
    start_time=$(date +%s)
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        key=$1
        case $key in
        --obo | -o)
            obo="--obo"
            shift
            ;;
        --eaas | -e)
            eaas="--eaas"
            shift
            ;;
        preview | upstream)
            mode=$1
            shift
            ;;
        -h | --help)
            print_help
            exit 0
            ;;
        *)
            shift
            ;;
        esac
    done

    log_step "Starting Konflux Cluster Bootstrap"
    log_info "Mode: ${mode:-upstream}"
    log_info "Options: OBO=${obo:-disabled}, EAAS=${eaas:-disabled}"
    log_info "Start time: $(date '+%Y-%m-%d %H:%M:%S %Z')"

    # Deploy ArgoCD
    log_step "Phase 1: Deploying ArgoCD"
    "${ROOT}/hack/deploy-argocd.sh"

    # Bootstrap host cluster
    log_step "Phase 2: Bootstrapping Host Cluster"
    "${ROOT}/hack/bootstrap-host-cluster.sh"

    # Bootstrap member cluster
    log_step "Phase 3: Bootstrapping Member Cluster"
    "${ROOT}/hack/bootstrap-member-cluster.sh"

    # Bootstrap common components
    log_step "Phase 4: Bootstrapping Common Components"
    "${ROOT}/hack/bootstrap-cluster-common.sh"

    # Mode-specific deployment
    log_step "Phase 5: Mode-Specific Deployment (${mode:-upstream})"
    case $mode in
    "" | "upstream")
        log_info "Deploying upstream/staging configuration"
        kubectl create -k $ROOT/argo-cd-apps/app-of-app-sets/staging
        
        # Remove any explicit transparency.url setting from chains-config
        log_info "Cleaning up chains-config for upstream mode"
        if kubectl get namespace openshift-pipelines &>/dev/null; then
            if kubectl get configmap chains-config -n openshift-pipelines &>/dev/null; then
                log_info "Removing transparency.url from chains-config (not needed for upstream)"
                if kubectl patch configmap/chains-config -n openshift-pipelines --type=json --patch '[{"op":"remove","path":"/data/transparency.url"}]' 2>/dev/null; then
                    log_success "Removed transparency.url from chains-config"
                    kubectl delete pod -n openshift-pipelines -l app=tekton-chains-controller
                    log_success "Restarted tekton-chains-controller"
                else
                    log_info "transparency.url not present in chains-config (already clean)"
                fi
            else
                log_info "chains-config ConfigMap not found (will be created by ArgoCD)"
            fi
        else
            log_info "openshift-pipelines namespace not yet created"
        fi
        log_success "Upstream configuration deployed"
        ;;
    "preview")
        log_info "Deploying preview configuration"
        $ROOT/hack/preview.sh $obo $eaas
        ;;
    esac

    # EaaS-specific setup (if enabled)
    if [ -n "$eaas" ]; then
        log_step "Phase 6: EaaS Cluster Bootstrap"
        log_info "OIDC secrets deployment (requires MCE operator to create local-cluster namespace)"
        "${ROOT}/hack/bootstrap-eaas-cluster.sh"
        log_success "EaaS cluster bootstrap complete"
    fi

    # Calculate total time
    end_time=$(date +%s)
    total_time=$((end_time - start_time))
    local total_min=$((total_time / 60))
    local total_sec=$((total_time % 60))

    log_step "Cluster Bootstrap Complete"
    log_success "Konflux cluster bootstrap finished successfully"
    log_info "  - Mode: ${mode:-upstream}"
    log_info "  - Total time: ${total_min}m ${total_sec}s"
    log_info "  - End time: $(date '+%Y-%m-%d %H:%M:%S %Z')"
    
    # Output JSON for LLM parsing
    echo ""
    echo "[BOOTSTRAP_SUMMARY_JSON] {\"status\":\"success\",\"mode\":\"${mode:-upstream}\",\"obo_enabled\":$([ -n "$obo" ] && echo "true" || echo "false"),\"eaas_enabled\":$([ -n "$eaas" ] && echo "true" || echo "false"),\"total_time_seconds\":$total_time}"
}

print_help() {
    echo "Usage: $0 MODE [-o|--obo] [-e|--eaas] [-h|--help]"
    echo ""
    echo "Bootstrap a Konflux cluster with ArgoCD and required components."
    echo ""
    echo "Arguments:"
    echo "  MODE             Deployment mode: 'upstream' or 'preview' (default: upstream)"
    echo ""
    echo "Options:"
    echo "  -o, --obo        Install Observability operator and Prometheus for federation"
    echo "                   (only applicable in preview mode)"
    echo "  -e, --eaas       Install Environment-as-a-Service components"
    echo "                   (only applicable in preview mode)"
    echo "  -h, --help       Show this help message and exit"
    echo ""
    echo "Examples:"
    echo "  $0                      # Bootstrap in upstream mode"
    echo "  $0 preview              # Bootstrap in preview mode"
    echo "  $0 preview --obo --eaas # Preview mode with OBO and EaaS"
    echo ""
    echo "Environment variables:"
    echo "  MY_GIT_FORK_REMOTE      Git remote name for your fork (required for preview)"
    echo "  MY_GITHUB_ORG           GitHub organization for components (required for preview)"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
