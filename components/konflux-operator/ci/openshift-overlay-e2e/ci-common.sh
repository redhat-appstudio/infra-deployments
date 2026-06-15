#!/usr/bin/env bash
# Shared OpenShift CI helpers for operator-overlay install and e2e steps.
# Expects GITHUB_TOKEN and INFRA_DEPLOYMENTS_ROOT to be set by the step entrypoint.
# GITHUB_USER is optional (defaults to x-access-token for PAT-based HTTPS git).
# Callers (CI entrypoint or phase scripts) must enable strict mode (set -euo pipefail).

_ci_kubeconfig_original=""
_ci_kubeconfig_copy=""
_ci_git_state_dir=""
_ci_git_config_path=""
_ci_git_creds_path=""
_ci_git_config_global_was_set=false
_ci_git_config_global_original=""
_ci_git_config_nosystem_was_set=false
_ci_git_config_nosystem_original=""

# Remove the temporary kubeconfig copy and restore KUBECONFIG to the path the step had on entry.
_ci_cleanup_kubeconfig_copy() {
  if [[ -n "${_ci_kubeconfig_copy}" ]]; then
    rm -f "${_ci_kubeconfig_copy}"
    _ci_kubeconfig_copy=""
  fi
  if [[ -n "${_ci_kubeconfig_original}" ]]; then
    export KUBECONFIG="${_ci_kubeconfig_original}"
  fi
}

# Tear down isolated git config/credential files and restore GIT_CONFIG_* env from before ci_configure_git_credentials.
_ci_cleanup_git_credentials() {
  if [[ "${_ci_git_config_global_was_set}" == true ]]; then
    export GIT_CONFIG_GLOBAL="${_ci_git_config_global_original}"
  else
    unset GIT_CONFIG_GLOBAL
  fi
  if [[ "${_ci_git_config_nosystem_was_set}" == true ]]; then
    export GIT_CONFIG_NOSYSTEM="${_ci_git_config_nosystem_original}"
  else
    unset GIT_CONFIG_NOSYSTEM
  fi
  _ci_git_config_global_was_set=false
  _ci_git_config_nosystem_was_set=false

  if [[ -n "${_ci_git_state_dir}" && -d "${_ci_git_state_dir}" ]]; then
    rm -rf "${_ci_git_state_dir}"
    _ci_git_state_dir=""
  fi
  _ci_git_config_path=""
  _ci_git_creds_path=""
}

# EXIT trap handler: run all ephemeral credential and kubeconfig cleanup.
_ci_cleanup() {
  _ci_cleanup_git_credentials
  _ci_cleanup_kubeconfig_copy
}

# Log into the claimed cluster without mutating the original KUBECONFIG file.
# Copies KUBECONFIG to a temp file, relaxes TLS verification on the copy (Hive pool parity with other Konflux CI steps),
# reads kubeadmin password from KUBEADMIN_PASSWORD_FILE or SHARED_DIR, and retries oc login for up to 5 minutes.
# Registers _ci_cleanup on EXIT.
ci_prepare_cluster_access() {
  local openshift_api openshift_password cluster_name

  _ci_kubeconfig_original="${KUBECONFIG:-}"
  if [[ -z "${_ci_kubeconfig_original}" ]]; then
    echo "KUBECONFIG is not set... Aborting job" >&2
    exit 1
  fi

  if [[ -n "${_ci_kubeconfig_copy}" ]]; then
    rm -f "${_ci_kubeconfig_copy}"
  fi
  _ci_kubeconfig_copy="$(mktemp "${TMPDIR:-/tmp}/kubeconfig-overlay-e2e.XXXXXX")"
  cp "${_ci_kubeconfig_original}" "${_ci_kubeconfig_copy}"
  export KUBECONFIG="${_ci_kubeconfig_copy}"
  trap _ci_cleanup EXIT

  # Normalize pool kubeconfig TLS on the copy (same outcome as yq in konflux-ci install steps).
  # Uses only oc so install (task-runner) and e2e (go-toolset) behave identically.
  cluster_name="$(oc config view --minify -o jsonpath='{.clusters[0].name}')"
  openshift_api="$(oc config view --minify -o jsonpath='{.clusters[0].cluster.server}')"
  oc config set-cluster "${cluster_name}" --insecure-skip-tls-verify=true
  oc config unset "clusters.${cluster_name}.certificate-authority-data" 2>/dev/null || true

  if [[ -s "${KUBEADMIN_PASSWORD_FILE:-}" ]]; then
    openshift_password="$(cat "${KUBEADMIN_PASSWORD_FILE}")"
  elif [[ -n "${SHARED_DIR:-}" && -s "${SHARED_DIR}/kubeadmin-password" ]]; then
    openshift_password="$(cat "${SHARED_DIR}/kubeadmin-password")"
  else
    echo "Kubeadmin password file is empty... Aborting job"
    exit 1
  fi

  timeout --foreground 5m bash <<-EOF
    while ! oc login "${openshift_api}" -u "kubeadmin" -p "${openshift_password}" --insecure-skip-tls-verify=true; do
      sleep 20
    done
EOF
}

# Configure ephemeral git identity and HTTPS credentials for github.com using GITHUB_TOKEN.
# Uses a private temp gitconfig and credential store (mode 600) so the runner user's ~/.gitconfig is untouched.
# Registers _ci_cleanup on EXIT.
ci_configure_git_credentials() {
  local github_user="${GITHUB_USER:-x-access-token}"

  if [[ -n "${_ci_git_state_dir}" && -d "${_ci_git_state_dir}" ]]; then
    rm -rf "${_ci_git_state_dir}"
  fi
  _ci_git_state_dir="$(mktemp -d "${TMPDIR:-/tmp}/overlay-e2e-git-state.XXXXXX")"
  _ci_git_config_path="${_ci_git_state_dir}/gitconfig"
  _ci_git_creds_path="${_ci_git_state_dir}/credentials"
  touch "${_ci_git_config_path}"
  chmod 600 "${_ci_git_config_path}"

  if [[ -v GIT_CONFIG_GLOBAL ]]; then
    _ci_git_config_global_was_set=true
    _ci_git_config_global_original="${GIT_CONFIG_GLOBAL}"
  fi
  if [[ -v GIT_CONFIG_NOSYSTEM ]]; then
    _ci_git_config_nosystem_was_set=true
    _ci_git_config_nosystem_original="${GIT_CONFIG_NOSYSTEM}"
  fi
  export GIT_CONFIG_GLOBAL="${_ci_git_config_path}"
  export GIT_CONFIG_NOSYSTEM=1

  umask 077
  install -m 600 /dev/null "${_ci_git_creds_path}"
  printf '%s\n' "https://${github_user}:${GITHUB_TOKEN}@github.com" > "${_ci_git_creds_path}"

  git config --global user.name "redhat-appstudio-qe-bot"
  git config --global user.email redhat-appstudio-qe-bot@redhat.com
  git config --global credential.helper "store --file ${_ci_git_creds_path}"

  trap _ci_cleanup EXIT
}

# Parse the konflux-ci operator install git ref from invariant/kustomization.yaml
# (the ?ref= on the konflux-ci/operator/config/default remote resource).
# Prints the ref to stdout; returns non-zero if the file or ref is missing.
ci_parse_konflux_ci_ref() {
  local kustomization="${1:?kustomization path required}"
  local ref

  if [[ ! -f "${kustomization}" ]]; then
    echo "ERROR: kustomization not found: ${kustomization}" >&2
    return 1
  fi

  ref="$(grep -E 'konflux-ci/konflux-ci/operator/config/default\?ref=' "${kustomization}" \
    | head -1 \
    | sed -n 's/.*ref=\([^[:space:]]*\).*/\1/p')"
  if [[ -z "${ref}" ]]; then
    echo "ERROR: could not parse konflux-ci ref from ${kustomization}" >&2
    return 1
  fi
  printf '%s\n' "${ref}"
}

# Choose a GitHub user:token pair from konflux-ci-secrets github_accounts by highest API rate limit.
# Exports GITHUB_USER and GITHUB_TOKEN for subsequent git and preview bootstrap operations.
ci_select_github_account() {
  local secrets_dir="${1:?secrets dir required}"
  local accounts_file="${secrets_dir}/github_accounts"
  local account github_user github_token rate previous_rate=0

  if [[ ! -s "${accounts_file}" ]]; then
    echo "ERROR: ${accounts_file} is missing or empty" >&2
    return 1
  fi

  IFS=',' read -r -a _ci_github_accounts <<< "$(cat "${accounts_file}")"
  for account in "${_ci_github_accounts[@]}"; do
    IFS=':' read -r github_user github_token <<< "${account}"
    if rate="$(curl --fail --silent --max-time 10 \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer ${github_token}" \
      https://api.github.com/rate_limit | jq -er '.rate.remaining')"; then
      echo "[INFO] GitHub user ${github_user}: ${rate} requests remaining"
      if [[ "${rate}" -ge "${previous_rate}" ]]; then
        GITHUB_USER="${github_user}"
        GITHUB_TOKEN="${github_token}"
        previous_rate="${rate}"
      fi
    else
      echo "[WARN] Failed to get rate limit for user: ${github_user}" >&2
    fi
  done

  if [[ -z "${GITHUB_USER:-}" || -z "${GITHUB_TOKEN:-}" ]]; then
    echo "ERROR: No valid GitHub credentials found in ${accounts_file}" >&2
    return 1
  fi
  export GITHUB_USER GITHUB_TOKEN
  echo "[INFO] Selected GitHub user: ${GITHUB_USER}"
}

# Load QE secret files and export environment variables expected by hack/preview.sh and hack/bootstrap-cluster.sh.
# Selects the best GitHub account, wires the redhat-appstudio-qe fork remote, and sets Quay, PAC, Smee, and
# PREVIEW_WAIT_KONFLUX_CR_READY. Requires INFRA_DEPLOYMENTS_ROOT (the cloned infra-deployments tree).
ci_export_preview_install_env() {
  local secrets_dir="${1:?secrets dir required}"

  : "${INFRA_DEPLOYMENTS_ROOT:?INFRA_DEPLOYMENTS_ROOT must be set}"
  cd "${INFRA_DEPLOYMENTS_ROOT}" || return 1

  ci_select_github_account "${secrets_dir}"

  git remote add origin "https://github.com/redhat-appstudio-qe/infra-deployments.git" 2>/dev/null || true
  git fetch origin main 2>/dev/null || true

  export MY_GITHUB_ORG="${MY_GITHUB_ORG:-redhat-appstudio-qe}"
  export MY_GITHUB_TOKEN="${GITHUB_TOKEN}"
  export MY_GIT_FORK_REMOTE="${MY_GIT_FORK_REMOTE:-origin}"
  export TEST_BRANCH_ID="${TEST_BRANCH_ID:-$(head /dev/urandom | tr -dc a-z0-9 | head -c 4)}"
  export QUAY_TOKEN
  QUAY_TOKEN="$(cat "${secrets_dir}/quay-token")"
  export IMAGE_CONTROLLER_QUAY_ORG="${IMAGE_CONTROLLER_QUAY_ORG:-redhat-appstudio-qe}"
  export IMAGE_CONTROLLER_QUAY_TOKEN
  IMAGE_CONTROLLER_QUAY_TOKEN="$(cat "${secrets_dir}/default-quay-org-token")"
  export BUILD_SERVICE_IMAGE_TAG_EXPIRATION="${BUILD_SERVICE_IMAGE_TAG_EXPIRATION:-5d}"
  export PAC_GITHUB_APP_ID
  PAC_GITHUB_APP_ID="$(cat "${secrets_dir}/pac-github-app-id")"
  export PAC_GITHUB_APP_PRIVATE_KEY
  PAC_GITHUB_APP_PRIVATE_KEY="$(cat "${secrets_dir}/pac-github-app-private-key")"
  export PAC_GITHUB_APP_WEBHOOK_SECRET
  PAC_GITHUB_APP_WEBHOOK_SECRET="$(cat "${secrets_dir}/pac-github-app-webhook-secret")"
  export SMEE_CHANNEL
  SMEE_CHANNEL="$(cat "${secrets_dir}/smee-channel")"
  export PREVIEW_WAIT_KONFLUX_CR_READY="${PREVIEW_WAIT_KONFLUX_CR_READY:-true}"
}

# Register the cluster's Pipelines-as-Code controller route with the shared QE SprayProxy service
# so GitHub webhooks reach PAC during install and e2e. Reads qe-sprayproxy-* from secrets_dir.
ci_register_sprayproxy_pac_route() {
  local secrets_dir="${1:?secrets dir required}"
  local sprayproxy_host sprayproxy_token pac_route http_code

  sprayproxy_host="$(cat "${secrets_dir}/qe-sprayproxy-host")"
  sprayproxy_token="$(cat "${secrets_dir}/qe-sprayproxy-token")"

  pac_route="$(oc get route pipelines-as-code-controller -n openshift-pipelines -o jsonpath='{.spec.host}')"
  if [[ -z "${pac_route}" ]]; then
    echo "ERROR: PAC route pipelines-as-code-controller not found in openshift-pipelines" >&2
    return 1
  fi

  echo "[INFO] Registering PAC route https://${pac_route} with SprayProxy..."
  http_code="$(curl -s -o /dev/null -w "%{http_code}" \
    --connect-timeout 10 \
    --max-time 30 \
    --retry 3 \
    -X POST \
    -H "Authorization: Bearer ${sprayproxy_token}" \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"https://${pac_route}\"}" \
    "${sprayproxy_host}/backends")"

  if [[ "${http_code}" =~ ^(200|201|302)$ ]]; then
    echo "[INFO] SprayProxy registration succeeded (HTTP ${http_code})"
    return 0
  fi
  echo "ERROR: SprayProxy registration failed (HTTP ${http_code})" >&2
  return 1
}

# Create or update the e2e-secrets/quay-repository docker-registry secret from a base64-encoded QUAY_TOKEN
# (same pattern as legacy appstudio e2e install). Used by tests that pull from the QE Quay org.
ci_create_e2e_quay_secret() {
  local quay_token="${1:?quay token required}"
  local temp_dockerconfig

  temp_dockerconfig="$(mktemp)"
  echo "${quay_token}" | base64 -d > "${temp_dockerconfig}"
  oc create namespace e2e-secrets --dry-run=client -o yaml | oc apply -f -
  oc create secret docker-registry quay-repository \
    -n e2e-secrets \
    --from-file=.dockerconfigjson="${temp_dockerconfig}" \
    --dry-run=client -o yaml | oc apply -f -
  rm -f "${temp_dockerconfig}"
}
