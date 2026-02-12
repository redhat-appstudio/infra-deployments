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
    log_info "Bootstrapping host cluster components"
    
    load_global_vars
    
    # Setup Quality Dashboard secrets
    log_substep "Configuring Quality Dashboard secrets"
    if "${ROOT}/secret-creator/quality-dashboard/create-quality-dashboard-secrets.sh"; then
        log_success "Quality Dashboard secrets configured"
    else
        log_warn "Failed to configure Quality Dashboard secrets (dashboard may have limited functionality)"
    fi
    
    # Setup CI Helper App secret
    log_substep "Configuring CI Helper App secret"
    if "${ROOT}/secret-creator/create-ci-helper-app-secret.sh"; then
        log_success "CI Helper App secret configured"
    else
        log_warn "Failed to configure CI Helper App secret"
    fi
    
    log_success "Host cluster bootstrap complete"
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
