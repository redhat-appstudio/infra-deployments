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

_ci_cleanup_kubeconfig_copy() {
  if [[ -n "${_ci_kubeconfig_copy}" ]]; then
    rm -f "${_ci_kubeconfig_copy}"
    _ci_kubeconfig_copy=""
  fi
  if [[ -n "${_ci_kubeconfig_original}" ]]; then
    export KUBECONFIG="${_ci_kubeconfig_original}"
  fi
}

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

_ci_cleanup() {
  _ci_cleanup_git_credentials
  _ci_cleanup_kubeconfig_copy
}

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
