#!/bin/bash -e

declare -r ROOT="${BASH_SOURCE[0]%/*}"

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

log_substep() {
    echo "[$(timestamp)] [SUBSTEP] $*"
}

# =============================================================================
# Main
# =============================================================================

main() {
    log_info "Bootstrapping member cluster components"
    
    load_global_vars
    
    # Setup Pipeline Service secrets
    log_substep "Configuring Pipeline Service secrets"
    if "${ROOT}/secret-creator/create-plnsvc-secrets.sh"; then
        log_success "Pipeline Service secrets configured"
    else
        log_warn "Failed to configure Pipeline Service secrets"
    fi
    
    # Setup GitHub secrets (if token provided)
    if [[ -n "$MY_GITHUB_TOKEN" ]]; then
        log_substep "Configuring GitHub secrets"
        log_info "GitHub token provided - creating secrets"
        if [[ -n "${GITHUB_TOKENS_LIST:-}" ]]; then
            log_info "Additional tokens list provided for multi-token support"
        fi
        
        if "${ROOT}/secret-creator/create-github-secret.sh" "$MY_GITHUB_TOKEN" "${GITHUB_TOKENS_LIST:-""}"; then
            log_success "GitHub secrets configured"
        else
            log_warn "Failed to configure GitHub secrets"
        fi
    else
        log_info "MY_GITHUB_TOKEN not set - skipping GitHub secret setup"
        log_warn "Some GitHub-related features may be limited without a token"
    fi
    
    # Setup Image Controller secrets
    log_substep "Configuring Image Controller secrets"
    local quay_org="${IMAGE_CONTROLLER_QUAY_ORG:-undefined}"
    local quay_token="${IMAGE_CONTROLLER_QUAY_TOKEN:-undefined}"
    
    if [[ "$quay_org" != "undefined" && "$quay_token" != "undefined" ]]; then
        log_info "Quay organization: $quay_org"
    else
        log_warn "IMAGE_CONTROLLER_QUAY_ORG or IMAGE_CONTROLLER_QUAY_TOKEN not set"
        log_warn "Image Controller will use default/placeholder values"
    fi
    
    if "${ROOT}/secret-creator/create-image-controller-secret.sh" "$quay_org" "$quay_token"; then
        log_success "Image Controller secrets configured"
    else
        log_warn "Failed to configure Image Controller secrets"
    fi
    
    log_success "Member cluster bootstrap complete"
}

load_global_vars() {
    local vars_file="$ROOT/preview.env"

    if [[ -f "$vars_file" ]]; then
        log_info "Loading environment variables from: $vars_file"
        source "$vars_file"
    else
        log_info "No preview.env file found - using environment variables only"
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
