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
    log_info "Bootstrapping common cluster components"
    
    load_global_vars
    
    # Setup Docker Hub pull secret if configured
    if [[ -n "$DOCKER_IO_AUTH" ]]; then
        log_substep "Configuring Docker Hub pull secret"
        if "${ROOT}/secret-creator/set-docker-hub-pull-secret.sh" "$DOCKER_IO_AUTH"; then
            log_success "Docker Hub pull secret configured"
        else
            log_warn "Failed to configure Docker Hub pull secret (may cause rate limiting)"
        fi
    else
        log_info "DOCKER_IO_AUTH not set - skipping Docker Hub pull secret setup"
        log_warn "Without Docker Hub credentials, you may experience rate limiting"
    fi
    
    # Setup DORA metrics exporter secrets
    log_substep "Configuring DORA metrics exporter secrets"
    if [[ -n "$GITHUB_TOKEN" ]]; then
        if "${ROOT}/secret-creator/create-dora-metrics-exporter-secrets.sh" "$GITHUB_TOKEN"; then
            log_success "DORA metrics exporter secrets configured"
        else
            log_warn "Failed to configure DORA metrics exporter secrets"
        fi
    else
        log_info "GITHUB_TOKEN not set - DORA metrics exporter may have limited functionality"
        # Still call the script as it may create placeholder secrets
        "${ROOT}/secret-creator/create-dora-metrics-exporter-secrets.sh" "" 2>/dev/null || true
    fi
    
    log_success "Common cluster bootstrap complete"
}

load_global_vars() {
    local vars_file="$ROOT/preview.env"

    if [[ -f "$vars_file" ]]; then
        log_info "Loading environment variables from: $vars_file"
        source "$vars_file"
        log_success "Environment variables loaded"
    else
        log_info "No preview.env file found at $vars_file - using environment variables only"
    fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
