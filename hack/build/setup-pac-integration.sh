#!/usr/bin/env bash

PAC_NAMESPACE='openshift-pipelines'
PAC_SECRET_NAME='pipelines-as-code-secret'
INTEGRATION_NAMESPACE='integration-service'
MAX_ROUTE_WAIT_RETRIES=20
ROUTE_WAIT_INTERVAL=5

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

log_wait() {
    echo "[$(timestamp)] [WAITING] $*"
}

log_debug() {
    echo "[$(timestamp)] [DEBUG] $*"
}

# =============================================================================
# Helper Functions
# =============================================================================

setup-pac-app() (
    # Inspired by implementation by Will Haley at:
    #   http://willhaley.com/blog/generate-jwt-with-bash/

    log_substep "Setting up GitHub App for Pipelines as Code"

    # Shared content to use as template
    header_template='{
    "typ": "JWT",
    "kid": "0001",
    "iss": "https://stackoverflow.com/questions/46657001/how-do-you-create-an-rs256-jwt-assertion-with-bash-shell-scripting"
    }'

    now=$(date +%s)
    build_header() {
        jq -c \
            --arg iat_str "$now" \
            --arg alg "${1:-HS256}" \
        '
        ($iat_str | tonumber) as $iat
        | .alg = $alg
        | .iat = $iat
        | .exp = ($iat + 10)
        ' <<<"$header_template" | tr -d '\n'
    }

    b64enc() { openssl enc -base64 -A | tr '+/' '-_' | tr -d '='; }
    json() { jq -c . | LC_CTYPE=C tr -d '\n'; }
    hs_sign() { openssl dgst -binary -sha"${1}" -hmac "$2"; }
    rs_sign() { openssl dgst -binary -sha"${1}" -sign <(printf '%s\n' "$2"); }

    sign() {
        local algo payload header sig secret=$3
        algo=${1:-RS256}; algo=${algo^^}
        header=$(build_header "$algo") || return
        payload=${2:-$test_payload}
        signed_content="$(json <<<"$header" | b64enc).$(json <<<"$payload" | b64enc)"
        case $algo in
            HS*) sig=$(printf %s "$signed_content" | hs_sign "${algo#HS}" "$secret" | b64enc) ;;
            RS*) sig=$(printf %s "$signed_content" | rs_sign "${algo#RS}" "$secret" | b64enc) ;;
            *) log_error "Unknown algorithm: $algo"; return 1 ;;
        esac
        printf '%s.%s\n' "${signed_content}" "${sig}"
    }
    payload="{ \"iss\": $PAC_GITHUB_APP_ID, \"iat\": ${now}, \"exp\": $((now+10)) }"

    webhook_secret=$(openssl rand -hex 20)
    log_debug "Generated webhook secret (length: ${#webhook_secret})"

    token=$(sign rs256 "$payload" "$(echo "$PAC_GITHUB_APP_PRIVATE_KEY" | base64 -d)")
    if [ -z "$token" ]; then
        log_error "Failed to generate JWT token for GitHub App"
        log_error "ACTION REQUIRED: Verify PAC_GITHUB_APP_PRIVATE_KEY is correctly base64 encoded"
        exit 1
    fi
    log_debug "Generated JWT token for GitHub App authentication"

    # Wait for PAC route to be available
    local retry=0
    log_substep "Waiting for Pipelines as Code controller route to be available"
    while ! oc get -n $PAC_NAMESPACE route pipelines-as-code-controller >/dev/null 2>&1 ; do
        if [ "$retry" -eq "$MAX_ROUTE_WAIT_RETRIES" ]; then
            log_error "============================================================================="
            log_error "TIMEOUT: PAC controller route not available after $((MAX_ROUTE_WAIT_RETRIES * ROUTE_WAIT_INTERVAL)) seconds"
            log_error "============================================================================="
            log_error "Namespace: $PAC_NAMESPACE"
            log_error "Expected route: pipelines-as-code-controller"
            log_error ""
            log_error "Current routes in namespace:"
            oc get routes -n $PAC_NAMESPACE 2>/dev/null || log_error "  (failed to list routes)"
            log_error ""
            log_error "PAC controller pod status:"
            oc get pods -n $PAC_NAMESPACE -l app.kubernetes.io/component=controller 2>/dev/null || log_error "  (failed to list pods)"
            log_error ""
            log_error "ACTION REQUIRED: Check if OpenShift Pipelines operator is installed and PAC is deployed"
            exit 1
        fi
        log_wait "PAC route not yet available (attempt $((retry+1))/$MAX_ROUTE_WAIT_RETRIES)"
        sleep $ROUTE_WAIT_INTERVAL
        retry=$((retry+1))
    done
    
    pac_host=$(oc get -n $PAC_NAMESPACE route pipelines-as-code-controller -o go-template="{{ .spec.host }}")
    log_success "PAC route is available: https://$pac_host"

    # Update GitHub App webhook configuration
    log_substep "Updating GitHub App webhook configuration"
    log_debug "  - Webhook URL: https://$pac_host"
    log_debug "  - GitHub App ID: $PAC_GITHUB_APP_ID"
    
    local curl_response curl_status
    curl_response=$(curl -s -w "\n%{http_code}" \
        -X PATCH \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: Bearer $token" \
        https://api.github.com/app/hook/config \
        -d "{\"content_type\":\"json\",\"insecure_ssl\":\"1\",\"secret\":\"$webhook_secret\",\"url\":\"https://$pac_host\"}")
    
    curl_status=$(echo "$curl_response" | tail -n1)
    
    if [ "$curl_status" -ge 200 ] && [ "$curl_status" -lt 300 ]; then
        log_success "GitHub App webhook configuration updated successfully"
    else
        log_warn "GitHub App webhook update returned status $curl_status (may still work if already configured)"
        log_debug "Response: $(echo "$curl_response" | head -n -1)"
    fi

    echo "$webhook_secret"
)

create_namespace_if_needed() {
    local namespace=$1
    log_substep "Ensuring namespace '$namespace' exists"
    if oc get namespace "$namespace" &>/dev/null; then
        log_debug "Namespace '$namespace' already exists"
    else
        if oc create namespace -o yaml --dry-run=client "$namespace" | oc apply -f-; then
            log_success "Created namespace '$namespace'"
        else
            log_error "Failed to create namespace '$namespace'"
            exit 1
        fi
    fi
}

create_pac_secret() {
    local namespace=$1
    local secret_data=$2
    
    log_substep "Creating PAC secret in namespace '$namespace'"
    if eval "oc -n '$namespace' create secret generic '$PAC_SECRET_NAME' $secret_data -o yaml --dry-run=client" | oc apply -f-; then
        log_success "PAC secret configured in '$namespace'"
    else
        log_error "Failed to create PAC secret in namespace '$namespace'"
        log_error "ACTION REQUIRED: Check if you have permissions to create secrets in this namespace"
        exit 1
    fi
}

# =============================================================================
# Main Execution
# =============================================================================

log_info "============================================================================="
log_info "Pipelines as Code (PAC) Integration Setup"
log_info "============================================================================="
log_info "PAC Namespace: $PAC_NAMESPACE"
log_info "PAC Secret Name: $PAC_SECRET_NAME"
log_info "Integration Namespace: $INTEGRATION_NAMESPACE"

# Determine authentication method and setup
GITHUB_APP_DATA=""
GITHUB_WEBHOOK_DATA=""
GITLAB_WEBHOOK_DATA=""

if [ -n "${PAC_GITHUB_APP_ID}" ] && [ -n "${PAC_GITHUB_APP_PRIVATE_KEY}" ]; then
    log_info "Authentication method: GitHub App"
    log_info "  - GitHub App ID: $PAC_GITHUB_APP_ID"
    
    # if using the existing QE sprayproxy, we suppose the setup between sprayproxy and github App is already done
    if [ -n "${PAC_GITHUB_APP_WEBHOOK_SECRET}" ]; then
        log_info "Using existing QE sprayproxy configuration (webhook secret provided)"
        WEBHOOK_SECRET="$PAC_GITHUB_APP_WEBHOOK_SECRET"
    else
        # if not, we setup pac with github App directly, this step will update the webhook secret and webhook url in the github App
        log_info "Configuring GitHub App directly (no sprayproxy)"
        WEBHOOK_SECRET=$(setup-pac-app)
        if [ -z "$WEBHOOK_SECRET" ]; then
            log_error "Failed to setup GitHub App - no webhook secret returned"
            exit 1
        fi
    fi
    
    GITHUB_APP_PRIVATE_KEY=$(echo "$PAC_GITHUB_APP_PRIVATE_KEY" | base64 -d)
    if [ -z "$GITHUB_APP_PRIVATE_KEY" ]; then
        log_error "Failed to decode GitHub App private key"
        log_error "ACTION REQUIRED: Verify PAC_GITHUB_APP_PRIVATE_KEY is valid base64-encoded PEM key"
        exit 1
    fi
    
    GITHUB_APP_DATA="--from-literal github-private-key='$GITHUB_APP_PRIVATE_KEY' --from-literal github-application-id='${PAC_GITHUB_APP_ID}' --from-literal webhook.secret='$WEBHOOK_SECRET'"
    log_success "GitHub App credentials configured"
else
    log_warn "GitHub App credentials not provided (PAC_GITHUB_APP_ID or PAC_GITHUB_APP_PRIVATE_KEY missing)"
    log_info "PAC will operate in limited mode without GitHub App integration"
fi

# Configure GitHub token (for webhook events)
if [ -n "${PAC_GITHUB_TOKEN}" ]; then
    log_info "GitHub token provided via PAC_GITHUB_TOKEN"
    GITHUB_WEBHOOK_DATA="--from-literal github.token='${PAC_GITHUB_TOKEN}'"
elif [ -n "${MY_GITHUB_TOKEN}" ]; then
    log_info "GitHub token provided via MY_GITHUB_TOKEN (fallback)"
    GITHUB_WEBHOOK_DATA="--from-literal github.token='${MY_GITHUB_TOKEN}'"
else
    log_warn "No GitHub token provided - some PAC features may be limited"
fi

# Configure GitLab token
if [ -n "${PAC_GITLAB_TOKEN}" ]; then
    log_info "GitLab token provided"
    GITLAB_WEBHOOK_DATA="--from-literal gitlab.token='${PAC_GITLAB_TOKEN}'"
else
    log_debug "No GitLab token provided (PAC_GITLAB_TOKEN not set)"
fi

# Create required namespaces
log_info "Creating required namespaces"
create_namespace_if_needed "$PAC_NAMESPACE"
create_namespace_if_needed "build-service"
create_namespace_if_needed "$INTEGRATION_NAMESPACE"

# Create PAC secrets in all required namespaces
log_info "Configuring PAC secrets across namespaces"

# Full credentials for PAC namespace, build-service, and integration-service
FULL_SECRET_DATA="$GITHUB_APP_DATA $GITHUB_WEBHOOK_DATA $GITLAB_WEBHOOK_DATA"
create_pac_secret "$PAC_NAMESPACE" "$FULL_SECRET_DATA"
create_pac_secret "build-service" "$FULL_SECRET_DATA"
create_pac_secret "$INTEGRATION_NAMESPACE" "$FULL_SECRET_DATA"

# Mintmaker only needs GitHub App data (no webhook tokens)
create_pac_secret "mintmaker" "$GITHUB_APP_DATA"

log_info "============================================================================="
log_success "PAC Integration Setup Complete"
log_info "============================================================================="
log_info "Configured namespaces:"
log_info "  - $PAC_NAMESPACE (PAC controller)"
log_info "  - build-service (Build Service)"
log_info "  - $INTEGRATION_NAMESPACE (Integration Service)"
log_info "  - mintmaker (Mintmaker)"

# Output summary for LLM parsing
echo ""
echo "[PAC_SETUP_JSON] {\"status\":\"success\",\"namespaces\":[\"$PAC_NAMESPACE\",\"build-service\",\"$INTEGRATION_NAMESPACE\",\"mintmaker\"],\"github_app_configured\":$([ -n "$GITHUB_APP_DATA" ] && echo "true" || echo "false"),\"github_token_configured\":$([ -n "$GITHUB_WEBHOOK_DATA" ] && echo "true" || echo "false"),\"gitlab_token_configured\":$([ -n "$GITLAB_WEBHOOK_DATA" ] && echo "true" || echo "false")}"
