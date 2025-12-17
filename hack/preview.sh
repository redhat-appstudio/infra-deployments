#!/bin/bash -e
set -o pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"/..

# =============================================================================
# Configuration
# =============================================================================
ARGOCD_NAMESPACE="openshift-gitops"
MAX_RETRIES=3
RETRY_DELAY=5
SYNC_INTERVAL=10
MAX_UNKNOWN_RETRIES=3

# =============================================================================
# Logging
# =============================================================================
log()   { echo "[$(date '+%H:%M:%S')] [INFO] $*"; }
warn()  { echo "[$(date '+%H:%M:%S')] [WARN] $*"; }
error() { echo "[$(date '+%H:%M:%S')] [ERROR] $*" >&2; }

# =============================================================================
# Core Functions
# =============================================================================

# Retry a command with exponential backoff
retry() {
  local attempt=1 output exit_code
  while [[ $attempt -le $MAX_RETRIES ]]; do
    output=$("$@" 2>&1) && exit_code=0 || exit_code=$?
    [[ $exit_code -eq 0 ]] && { echo "$output"; return 0; }
    [[ $attempt -lt $MAX_RETRIES ]] && warn "Retry $attempt/$MAX_RETRIES failed, waiting ${RETRY_DELAY}s..."
    sleep $RETRY_DELAY
    ((attempt++))
  done
  error "Command failed after $MAX_RETRIES attempts: $*"
  echo "$output"
  return $exit_code
}

# Wait for a Kubernetes resource to exist
wait_for_resource() {
  local resource=$1 namespace=$2
  log "Waiting for $resource to be created..."
  until retry oc get "$resource" -n "$namespace" &>/dev/null; do sleep 2; done
}

# Wait for ArgoCD app to reach a status
wait_for_app_status() {
  local app=$1 target_status=$2
  log "Waiting for $app to sync..."
  while true; do
    local status=$(retry oc get applications.argoproj.io "$app" -n "$ARGOCD_NAMESPACE" \
      -o jsonpath='{.status.health.status} {.status.sync.status}' 2>/dev/null || echo "")
    [[ "$status" == "$target_status" ]] && break
    log "$app status: ${status:-pending}..."
    sleep $SYNC_INTERVAL
  done
  log "$app is synced"
}

# Override image in kustomization file
override_image() {
  local file=$1 image_name=$2 repo_var=$3 tag_var=$4
  local repo="${!repo_var}" tag="${!tag_var}"
  [[ -n "$repo" ]] && yq -i e "(.images.[] | select(.name==\"$image_name\")) |=.newName=\"$repo\"" "$file"
  [[ -n "$tag" ]] && yq -i e "(.images.[] | select(.name==\"$image_name\")) |=.newTag=\"$tag\"" "$file"
}

# Show detailed failure information for an app
show_app_failure() {
  local app=$1 app_json=$2
  local error_msg=$(echo "$app_json" | jq -r '.status.operationState.message // "No message"')
  local target_ns=$(echo "$app_json" | jq -r '.spec.destination.namespace // ""')
  
  error "App '$app' failed"
  echo "  Message: $error_msg"
  
  [[ -z "$target_ns" || "$target_ns" == "null" ]] && return
  
  # Image pull errors
  local image_errors=$(oc get pods -n "$target_ns" 2>/dev/null | grep -E "ImagePull|ErrImage" || true)
  [[ -n "$image_errors" ]] && echo -e "\n  Image pull failures:\n$(echo "$image_errors" | sed 's/^/    /')"
  
  # Warning events
  echo -e "\n  Recent events:"
  oc get events -n "$target_ns" --field-selector=type=Warning --sort-by='.lastTimestamp' 2>/dev/null | \
    tail -5 | sed 's/^/    /'
}

# Show summary of all failing apps
show_failure_summary() {
  local not_done=$1 done_count=$2 total=$3
  echo ""
  error "=============================================="
  error "DEPLOYMENT FAILED"
  error "=============================================="
  echo ""
  echo "Apps NOT ready:"
  echo "$not_done" | awk '{printf "  - %s: %s %s\n", $1, $2, $3}'
  echo ""
  echo "Progress: $done_count/$total apps ready"
}

# =============================================================================
# Validation
# =============================================================================
print_help() {
  cat << EOF
Usage: $0 [--obo] [--eaas] [-h|--help]

Options:
  --obo     Install Observability operator and Prometheus for federation
  --eaas    Install Environment as a Service components
  -h        Show this help message

Required environment variables:
  MY_GIT_FORK_REMOTE  Your git fork remote name
  MY_GITHUB_ORG       Your GitHub organization
EOF
}

OBO=false EAAS=false
while [[ $# -gt 0 ]]; do
  case $1 in
    --obo)  OBO=true; shift ;;
    --eaas) EAAS=true; shift ;;
    -h|--help) print_help; exit 0 ;;
    *) shift ;;
  esac
done

[[ -f "$ROOT/hack/preview.env" ]] && source "$ROOT/hack/preview.env"

[[ -z "$MY_GIT_FORK_REMOTE" ]] && { error "Set MY_GIT_FORK_REMOTE environment variable"; exit 1; }
[[ -z "$MY_GITHUB_ORG" || "$MY_GITHUB_ORG" == "redhat-appstudio-appdata" ]] && { error "Set MY_GITHUB_ORG environment variable"; exit 1; }

MY_GIT_REPO_URL=$(git ls-remote --get-url "$MY_GIT_FORK_REMOTE" | sed 's|^git@github.com:|https://github.com/|')
MY_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "$MY_GIT_REPO_URL" | grep -q "redhat-appstudio/infra-deployments" && { error "Use your fork repository"; exit 1; }
git diff --exit-code --quiet || { error "Uncommitted changes in working tree"; exit 1; }

trap "git checkout $MY_GIT_BRANCH 2>/dev/null" EXIT

# =============================================================================
# Setup Preview Branch
# =============================================================================
log "Setting up preview branch..."
PREVIEW_BRANCH="preview-${MY_GIT_BRANCH}${TEST_BRANCH_ID:+-$TEST_BRANCH_ID}"
git rev-parse --verify "$PREVIEW_BRANCH" &>/dev/null && git branch -D "$PREVIEW_BRANCH"
git checkout -b "$PREVIEW_BRANCH"

# Patch ArgoCD applications to point to fork
for file in application-patch.yaml application-set-patch.yaml application-set-multisrc-src-1-patch.yaml; do
  filepath="${ROOT}/argo-cd-apps/k-components/inject-infra-deployments-repo-details/$file"
  yq -i ".[0].value = \"$MY_GIT_REPO_URL\" | .[1].value = \"$PREVIEW_BRANCH\"" "$filepath"
done

$OBO && {
  log "Adding Observability operator"
  yq -i '.resources += ["monitoringstack/"]' "$ROOT/components/monitoring/prometheus/development/kustomization.yaml"
}

$EAAS && {
  log "Enabling EaaS cluster role"
  yq -i '.components += ["../../../k-components/assign-eaas-role-to-local-cluster"]' \
    "$ROOT/argo-cd-apps/base/local-cluster-secret/all-in-one/kustomization.yaml"
}

# =============================================================================
# Node Labeling
# =============================================================================
log "Labeling cluster nodes..."
nodes=$(retry kubectl get nodes -o name)
[[ -z "$nodes" ]] && { error "No nodes found"; exit 1; }
node_count=$(echo "$nodes" | grep -c . || echo 0)

for node in $nodes; do
  kubectl label "$node" konflux-ci.dev/workload=konflux-tenants --overwrite 2>/dev/null && \
    log "Labeled $node" || warn "Failed to label $node"
done

labeled_count=$(retry kubectl get nodes --show-labels | grep -c "konflux-ci.dev/workload=konflux-tenants" || echo 0)
[[ "$node_count" -eq "$labeled_count" ]] || { error "Label verification failed: $labeled_count/$node_count"; exit 1; }
log "All nodes labeled ($labeled_count/$node_count)"

# =============================================================================
# Conditional Application Deployment
# =============================================================================
if [[ -n "$DEPLOY_ONLY" ]]; then
  log "Filtering applications based on DEPLOY_ONLY..."
  APPLICATIONS=$(oc kustomize argo-cd-apps/overlays/development | yq e --no-doc 'select(.kind == "ApplicationSet") | .metadata.name')
  DELETED=$(yq e --no-doc .metadata.name "$ROOT/argo-cd-apps/overlays/development/delete-applications.yaml")
  DELETE_FILE="$ROOT/argo-cd-apps/overlays/development/delete-applications.yaml"
  
  for APP in $APPLICATIONS; do
    if ! grep -q "\b$APP\b" <<< "$DEPLOY_ONLY" && ! grep -q "\b$APP\b" <<< "$DELETED"; then
      log "Disabling $APP"
      echo '---' >> "$DELETE_FILE"
      yq e -n ".apiVersion=\"argoproj.io/v1alpha1\" | .kind=\"ApplicationSet\" | .metadata.name=\"$APP\" | .\$patch=\"delete\"" >> "$DELETE_FILE"
    fi
  done
fi

# Handle Kueue based on OCP version
OCP_MINOR=$(retry oc get clusterversion version -o jsonpath='{.status.desired.version}' | cut -d. -f2)
log "OCP version: 4.$OCP_MINOR"
if [[ "$OCP_MINOR" -lt 16 ]]; then
  DELETE_FILE="$ROOT/argo-cd-apps/overlays/development/delete-applications.yaml"
  if ! grep -q "name: kueue" "$DELETE_FILE"; then
    log "Disabling kueue (OCP < 4.16)"
    echo '---' >> "$DELETE_FILE"
    yq e -n ".apiVersion=\"argoproj.io/v1alpha1\" | .kind=\"ApplicationSet\" | .metadata.name=\"kueue\" | .\$patch=\"delete\"" >> "$DELETE_FILE"
  fi
  yq -i 'del(.resources[] | select(test("^kueue/?$")))' "$ROOT/components/policies/development/kustomization.yaml"
fi

# =============================================================================
# Image Overrides
# =============================================================================
"$ROOT/hack/util-set-github-org" "$MY_GITHUB_ORG"

domain=$(retry oc get ingresses.config.openshift.io cluster --template={{.spec.domain}})
sed -i.bak "s/rekor-server.enterprise-contract-service.svc/rekor.$domain/" \
  "$ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml" && \
  rm "$ROOT/argo-cd-apps/base/member/optional/helm/rekor/rekor.yaml.bak"

# Service image overrides
override_image "$ROOT/components/build-service/development/kustomization.yaml" \
  "quay.io/konflux-ci/build-service" BUILD_SERVICE_IMAGE_REPO BUILD_SERVICE_IMAGE_TAG
override_image "$ROOT/components/has/base/kustomization.yaml" \
  "quay.io/redhat-appstudio/application-service" HAS_IMAGE_REPO HAS_IMAGE_TAG
override_image "$ROOT/components/integration/development/kustomization.yaml" \
  "quay.io/konflux-ci/integration-service" INTEGRATION_SERVICE_IMAGE_REPO INTEGRATION_SERVICE_IMAGE_TAG
override_image "$ROOT/components/release/development/kustomization.yaml" \
  "quay.io/konflux-ci/release-service" RELEASE_SERVICE_IMAGE_REPO RELEASE_SERVICE_IMAGE_TAG
override_image "$ROOT/components/mintmaker/development/kustomization.yaml" \
  "quay.io/konflux-ci/mintmaker" MINTMAKER_IMAGE_REPO MINTMAKER_IMAGE_TAG
override_image "$ROOT/components/mintmaker/development/kustomization.yaml" \
  "quay.io/konflux-ci/mintmaker-renovate-image" MINTMAKER_RENOVATE_IMAGE_REPO MINTMAKER_RENOVATE_IMAGE_TAG
override_image "$ROOT/components/image-controller/development/kustomization.yaml" \
  "quay.io/konflux-ci/image-controller" IMAGE_CONTROLLER_IMAGE_REPO IMAGE_CONTROLLER_IMAGE_TAG
override_image "$ROOT/components/multi-platform-controller/base/kustomization.yaml" \
  "multi-platform-controller" MULTI_ARCH_CONTROLLER_IMAGE_REPO MULTI_ARCH_CONTROLLER_IMAGE_TAG

# PR-based overrides (kept as-is due to complex yq expressions)
[[ -n "${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}" ]] && yq -i e "(.spec.template.spec.containers[].env[] | select(.name==\"IMAGE_TAG_ON_PR_EXPIRATION\") | .value) |= \"${BUILD_SERVICE_IMAGE_TAG_EXPIRATION}\"" "$ROOT/components/build-service/development/image-expiration-patch.yaml"
[[ -n "${BUILD_SERVICE_PR_OWNER}" && -n "${BUILD_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/build-service*\")) |= \"https://github.com/${BUILD_SERVICE_PR_OWNER}/build-service/config/default?ref=${BUILD_SERVICE_PR_SHA}\"" "$ROOT/components/build-service/development/kustomization.yaml"
[[ -n "${HAS_PR_OWNER}" && -n "${HAS_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/redhat-appstudio/application-service*\")) |= \"https://github.com/${HAS_PR_OWNER}/application-service/config/default?ref=${HAS_PR_SHA}\"" "$ROOT/components/has/base/kustomization.yaml"
[[ -n "${INTEGRATION_SERVICE_PR_OWNER}" && -n "${INTEGRATION_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/integration-service*\")) |= (sub(\"\?ref=.+\", \"?ref=${INTEGRATION_SERVICE_PR_SHA}\" ) | sub(\"github.com/redhat-appstudio\", \"github.com/${INTEGRATION_SERVICE_PR_OWNER}\"))" "$ROOT/components/integration/development/kustomization.yaml"
[[ -n "${RELEASE_SERVICE_PR_OWNER}" && -n "${RELEASE_SERVICE_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/release-service*\")) |= \"https://github.com/${RELEASE_SERVICE_PR_OWNER}/release-service/config/default?ref=${RELEASE_SERVICE_PR_SHA}\"" "$ROOT/components/release/development/kustomization.yaml"
[[ -n "${MINTMAKER_PR_OWNER}" && -n "${MINTMAKER_PR_SHA}" ]] && yq -i "(.resources[] | select(contains(\"konflux-ci/mintmaker\"))) |= (sub(\"konflux-ci/mintmaker\", \"${MINTMAKER_PR_OWNER}/mintmaker\") | sub(\"ref=.*\", \"ref=${MINTMAKER_PR_SHA}\"))" "$ROOT/components/mintmaker/development/kustomization.yaml"
[[ -n "${IMAGE_CONTROLLER_PR_OWNER}" && -n "${IMAGE_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/image-controller*\")) |= \"https://github.com/${IMAGE_CONTROLLER_PR_OWNER}/image-controller/config/default?ref=${IMAGE_CONTROLLER_PR_SHA}\"" "$ROOT/components/image-controller/development/kustomization.yaml"
[[ -n "${MULTI_ARCH_CONTROLLER_PR_OWNER}" && -n "${MULTI_ARCH_CONTROLLER_PR_SHA}" ]] && yq -i e "(.resources[] | select(. ==\"*github.com/konflux-ci/multi-platform-controller*\")) |= (sub(\"\?ref=.+\", \"?ref=${MULTI_ARCH_CONTROLLER_PR_SHA}\" ) | sub(\"github.com/konflux-ci\", \"github.com/${MULTI_ARCH_CONTROLLER_PR_OWNER}\"))" "$ROOT/components/multi-platform-controller/base/kustomization.yaml"
[[ -n "${EAAS_HYPERSHIFT_BASE_DOMAIN}" ]] && yq -i e "(.[] | select(.value.name==\"baseDomain\")).value.value |= \"${EAAS_HYPERSHIFT_BASE_DOMAIN}\"" "$ROOT/components/cluster-as-a-service/development/add-hypershift-params.yaml"
[[ -n "${EAAS_HYPERSHIFT_CLI_ROLE_ARN}" ]] && yq -i e "(.[] | select(.value.name==\"hypershiftRoleArn\")).value.value |= \"${EAAS_HYPERSHIFT_CLI_ROLE_ARN}\"" "$ROOT/components/cluster-as-a-service/development/add-hypershift-params.yaml"
[[ -n "${PIPELINE_PR_OWNER}" && -n "${PIPELINE_PR_SHA}" ]] && yq -i e ".resources[] |= sub(\"ref=[^ ]*\"; \"ref=${PIPELINE_PR_SHA}\") | .resources[] |= sub(\"openshift-pipelines\"; \"${PIPELINE_PR_OWNER}\")" "$ROOT/components/pipeline-service/development/kustomization.yaml"

# =============================================================================
# Push Changes and Deploy
# =============================================================================
if ! git diff --exit-code --quiet; then
  git commit -a -m "Preview mode, do not merge into main"
  retry git push -f --set-upstream "$MY_GIT_FORK_REMOTE" "$PREVIEW_BRANCH"
fi

log "Deploying ArgoCD applications..."
retry oc apply -k "$ROOT/argo-cd-apps/app-of-app-sets/development"

# =============================================================================
# Wait for ArgoCD Sync
# =============================================================================
wait_for_resource "applications.argoproj.io/all-application-sets" "$ARGOCD_NAMESPACE"
wait_for_app_status "all-application-sets" "Healthy Synced"

# Trigger refresh
log "Triggering refresh of all apps..."
APPS=$(retry oc get apps -n "$ARGOCD_NAMESPACE" -o name)
for APP in $APPS; do
  oc patch "$APP" -n "$ARGOCD_NAMESPACE" --type merge \
    -p='{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}' 2>/dev/null &
done
wait

# Wait for refresh to complete
log "Waiting for refresh to complete..."
while true; do
  pending=$(retry oc get applications.argoproj.io -n "$ARGOCD_NAMESPACE" \
    -o jsonpath='{range .items[?(@.metadata.annotations.argocd\.argoproj\.io/refresh)]}{.metadata.name}{" "}{end}' 2>/dev/null || echo "")
  [[ -z "$pending" ]] && break
  pending_count=$(echo "$pending" | wc -w)
  pending_list=$(echo "$pending" | tr ' ' '\n' | head -5 | tr '\n' ',' | sed 's/,$//')
  log "Refresh pending: $pending_count apps ($pending_list$([ "$pending_count" -gt 5 ] && echo ", ..."))"
  sleep $SYNC_INTERVAL
done
log "Refresh complete"

# Wait for all apps to sync
log "Waiting for all applications to sync..."
unknown_retries=0

while true; do
  STATE=$(retry oc get apps -n "$ARGOCD_NAMESPACE" --no-headers) || { sleep $SYNC_INTERVAL; continue; }
  NOT_DONE=$(echo "$STATE" | grep -v "Synced[[:blank:]]*Healthy" || true)
  
  [[ -z "$NOT_DONE" ]] && { log "All applications synced and healthy"; break; }
  
  total=$(echo "$STATE" | grep -c . || echo 0)
  done_count=$(echo "$STATE" | grep -c "Synced[[:blank:]]*Healthy" || echo 0)
  log "Progress: $done_count/$total apps ready"
  
  # Handle Unknown apps
  UNKNOWN=$(echo "$NOT_DONE" | awk '/Unknown/ && !/Progressing/ {print $1}') || :
  if [[ -n "$UNKNOWN" ]]; then
    for app in $UNKNOWN; do
      APP_JSON=$(oc get applications.argoproj.io "$app" -n "$ARGOCD_NAMESPACE" -o json 2>/dev/null || echo '{}')
      
      # Retry on timeout
      if echo "$APP_JSON" | jq -e '.status.conditions[]? | select(.message | contains("context deadline"))' &>/dev/null; then
        warn "App '$app' timeout, refreshing..."
        oc patch applications.argoproj.io "$app" -n "$ARGOCD_NAMESPACE" \
          --type merge -p='{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"soft"}}}' 2>/dev/null || true
        sleep 10
        unknown_retries=0
        continue 2
      fi
      
      # Give Unknown state time to recover
      ((unknown_retries++))
      if [[ $unknown_retries -lt $MAX_UNKNOWN_RETRIES ]]; then
        warn "App '$app' Unknown (attempt $unknown_retries/$MAX_UNKNOWN_RETRIES)..."
        sleep $SYNC_INTERVAL
        continue 2
      fi
      
      # Failed - show details
      show_app_failure "$app" "$APP_JSON"
    done
    show_failure_summary "$NOT_DONE" "$done_count" "$total"
    exit 1
  fi
  
  unknown_retries=0
  PENDING=$(echo "$NOT_DONE" | awk '{print $1}' | head -5 | tr '\n' ',' | sed 's/,$//')
  log "Waiting for: $PENDING..."
  sleep $SYNC_INTERVAL
done

# =============================================================================
# Wait for Tekton
# =============================================================================
log "Waiting for Tekton..."
wait_for_resource "tektonconfig/config" ""

while true; do
  TEKTON_JSON=$(retry oc get tektonconfig config -o json 2>/dev/null || echo '{}')
  ready=$(echo "$TEKTON_JSON" | jq -r '.status.conditions[] | select(.type=="Ready") | .status' 2>/dev/null)
  [[ "$ready" == "True" ]] && { log "Tekton is ready"; break; }
  
  msg=$(echo "$TEKTON_JSON" | jq -r '.status.conditions[] | select(.type=="Ready") | .message' 2>/dev/null || echo "waiting")
  warn "Tekton not ready: $msg"
  
  # Workaround for SRVKP-3245
  if echo "$msg" | grep -q 'OpenShiftPipelinesAsCode: reconcile again'; then
    all_available=true
    for deploy in pipelines-as-code-controller pipelines-as-code-watcher pipelines-as-code-webhook; do
      [[ "$(retry oc get deployment/$deploy -n openshift-pipelines -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')" != "True" ]] && all_available=false
    done
    $all_available && { warn "Bypassing tektonconfig check (SRVKP-3245)"; break; }
  fi
  sleep $SYNC_INTERVAL
done

# Wait for CRDs
log "Waiting for Tekton CRDs..."
for i in {1..10}; do
  crds=$(retry oc api-resources --api-group="tekton.dev" --no-headers 2>/dev/null || echo "")
  [[ $crds =~ pipelines && $crds =~ tasks && $crds =~ pipelineruns && $crds =~ taskruns ]] && { log "Tekton CRDs ready"; break; }
  [[ $i -eq 10 ]] && { error "Tekton CRDs not available"; exit 1; }
  log "Waiting for CRDs (attempt $i/10)..."
  sleep $SYNC_INTERVAL
done

# =============================================================================
# Final Setup
# =============================================================================
"$ROOT/hack/build/setup-pac-integration.sh"
log "Preview deployment complete"
