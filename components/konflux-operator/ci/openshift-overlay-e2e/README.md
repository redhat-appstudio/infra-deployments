# OpenShift CI: `development-operator` overlay E2E

Scripts for the optional, on-demand Prow job
`appstudio-operator-overlay-e2e-tests` (openshift/release).

Legacy `appstudio-e2e-tests` (`development` overlay) is unchanged and uses separate
step refs (`konflux-ci-install-konflux`, `redhat-appstudio-conformance-tests`).

## Layout

| File | Role |
|------|------|
| `Dockerfile` | Unified CI image (`konflux-overlay-install`): task-runner + Go 1.26 from ubi10/go-toolset |
| `ci-common.sh` | Shared cluster login (temp kubeconfig copy) and ephemeral git credentials |
| `install.sh` | `hack/bootstrap-cluster.sh preview --operator-overlay`, QE secrets, SprayProxy, `e2e-secrets` quay pull secret |
| `run-e2e.sh` | Clone konflux-ci @ ref from `invariant/kustomization.yaml`; `prepare-conformance-env` + `run-conformance-tests.sh` in `default-tenant` (no `deploy-test-resources.sh`) |

## CI flow (both steps use the same pattern)

1. ci-operator builds `konflux-overlay-install` from `Dockerfile` (per job, not promoted to `ci/`).
2. Install and e2e steps both use `from: konflux-overlay-install` in openshift/release.
3. Shared entrypoint `redhat-appstudio-operator-overlay-commands.sh` clones `infra-deployments`, merges the PR when applicable, and calls `ci-common.sh`.
4. The e2e step sets `OVERLAY_E2E_SCRIPT_NAME=run-e2e.sh` and sources that same entrypoint.
5. `install.sh` or `run-e2e.sh` runs the phase-specific logic.

Both refs set `cli: latest` so `oc` is available in the pod.

## Local build of the CI image

```bash
cd infra-deployments
podman build -f components/konflux-operator/ci/openshift-overlay-e2e/Dockerfile \
  -t konflux-overlay-install:local .
```

## Local scripts

Run from an `infra-deployments` checkout (after setting `KUBECONFIG`, `GITHUB_TOKEN`, etc.).
`GITHUB_USER` is optional; when unset, git HTTPS auth uses the `x-access-token` placeholder
(GitHub accepts any username with a PAT). CI sets `GITHUB_USER=github-token` in the step entrypoint.

`ci_prepare_cluster_access` copies `KUBECONFIG` to a temp file before TLS/login changes so your
original kubeconfig file is not modified. While the step runs, `KUBECONFIG` points at the copy; an
`EXIT` trap removes the copy and restores `KUBECONFIG` to the original path when the shell exits.

`ci_configure_git_credentials` uses a temp `GIT_CONFIG_GLOBAL` (with `GIT_CONFIG_NOSYSTEM=1`) and
a temp credential store file so your `~/.gitconfig` is not modified and the PAT is removed on
`EXIT`. For interactive testing, run via a wrapper script or subshell so env vars from the step do
not linger mid-session.

```bash
./components/konflux-operator/ci/openshift-overlay-e2e/install.sh
./components/konflux-operator/ci/openshift-overlay-e2e/run-e2e.sh
```
