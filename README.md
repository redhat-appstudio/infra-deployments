# AppStudio Infrastructure Deployments

This repository is an initial set of Argo-CD-based deployments of AppStudio components to a cluster, plus a script to bootstrap Argo CD onto that cluster (to drive these Argo-CD-based deployments, via OpenShift GitOps).

This repository is structured as a GitOps monorepo (e.g. the repository contains the K8s resources for *multiple* applications), using [Kustomize](https://kustomize.io/).

The contents of this repository are not owned by any single individual, and should instead be collectively managed and maintained through PRs by individual teams.

## How to add your own component

You may use the `gitops` component as an example for how to add your own component. Here `gitops` refers to the GitOps service team's K8s resources.

These are the steps to add your own component:

1) Create a new directory for your team's components, under `components/(team-name)`.
2) Add a `kustomization.yaml` file under that directory, which points to the individual K8s YAML resources you wish to deploy.
    - You may also structure your deployment into directories and files. See the Kustomize documentation for more information, and/or examples below.
    - See `components/gitops/staging` for an example of this.
3) Create an Argo CD `Application` resource in `argo-cd-apps/base/(team-name).yaml`).
    - See `gitops.yaml` for a template of how this should look.
    - The `.spec.source.path` value should point to the directory you created in previous step.
    - The `.spec.destination.namespace` should match the target namespace you wish to deploy your resources to.
    - The `.metadata.name` should correspond to your `(team-name)`
4) Add a reference to your new `(team-name).yaml` file, to `argo-cd-apps/base/kustomization.yaml` (the reference to your YAML file should be in the `resources:` list field).
5) Run `kustomize build (repo root)/argo-cd-apps/overlays/staging` and ensure it passes, and outputs your new Argo CD Application CR.
6) Add an entry in `argo-cd-apps/overlays/development/repo-overlay.yaml` for your new component so you can use the preview mode for testing.
7) Open a PR for all of the above.

More examples of using Kustomize to drive deployments using GitOps can be [found here](https://github.com/redhat-cop/gitops-catalog).

## Component testing and building of images

[Pipelines as Code](https://pipelinesascode.com/) is deployed and available for testing and building of images.
To test and run builds for a component, add your github repository to `components/tekton-ci/repository.yaml`.

Target repository has to have installed GitHub app - [AppStudio Staging CI](https://github.com/apps/appstudio-staging-ci) and pipelineRuns created in `.tekton` folder, example [Build Service](https://github.com/redhat-appstudio/build-service/tree/main/.tekton). Target image repository in quay.io must exist and robot account `redhat-appstudio+staginguser` has to have `write` permission on the repository.


## Maintaining your components

Simply update the files under `components/(team-name)`, and open a PR with the changes.

**TIP**: For development purposes, you can use `kustomize build .` to output the K8s resources that are being generated for your folder.

## Bootstrapping a cluster

### Required prerequisites

The prerequisites are:

- You must have `kubectl`, `oc`, `jq` and [`yq`](https://github.com/mikefarah/yq) installed.
- You must have `kubectl` and `oc` pointing to an existing OpenShift cluster, that you wish to deploy to. Alternatively, you can configure a local CodeReady Containers VM to deploy to.
- The script `./hack/setup/install-pre-req.sh` will install these prerequisites for you, if they're not already installed.

**Note - Mac OS**

If you're using Mac OS, make sure you are using GNU version of `sed` (`sed --version` -> **GNU sed 4.8**), openssl `openssl version` >= v3.0.2 and bash (`bash --version` >= **GNU bash, version 5.2**), because some of the configuration scripts in this repository depend on them.

You can install correct versions of these tools with:
```bash
brew install openssl@3 gnu-sed bash
```
Then make sure the $PATH is updated to point to those tools' binaries (by updating your .bashrc/.zshrc file):
```bash
export PATH="/usr/local/opt/openssl@3/bin:$PATH"
export PATH="/usr/local/opt/gnu-sed/libexec/gnubin:$PATH"
export PATH="/usr/local/bin/bash:${PATH}"
```
After opening a new terminal window you should be using correct versions of these tools by default.

### Optional: CodeReady Containers Setup

If you don't already have a test OpenShift cluster available, CodeReady Containers is a popular option. It runs a small OpenShift cluster in a single VM on your local workstation.

1) Create or log in using your free Red Hat account, and [install CodeReady Containers (CRC)](https://console.redhat.com/openshift/create/local).
2) Make sure you have the latest version of CRC: `crc version`
3) Run `./hack/setup/prepare-crc.sh` to configure CodeReady Containers with the recommended minimum memory (16 GiB) and CPUs (6) for App Studio. The script has optional parameters for customizing `memory` and `cpu` allowance. It also supports `force delete` of existing cluster. Run `./hack/setup/prepare-crc.sh --help` to see the options. The script will also enable cluster monitoring and log you in as the cluster administrator.

### Optional: Quicklab storage setup for clusters

If you are using Quicklab to provision your development cluster, you will need to setup storage prior to running the bootstrap script.

See [hack/quicklab/README.md](hack/quicklab/README.md)

### Bootstrap App Studio

Steps:

1) Run `./hack/bootstrap-cluster.sh [preview]` which will bootstrap Argo CD (using OpenShift GitOps) and setup the Argo CD `Application` Custom Resources (CRs) for each component. This command will output the Argo CD Web UI route when it's finished. `preview` will enable preview mode used for development and testing on non-production clusters, described in section [Preview mode for your clusters](#preview-mode-for-your-clusters).
2) Open the Argo CD Web UI to see the status of your deployments. You can use the route from the previous step and login using your OpenShift credentials (using the 'Login with OpenShift' button), or login to the OpenShift Console and navigate to Argo CD using the OpenShift Gitops menu in the Applications pulldown.
![OpenShift Gitops menu with Cluster Argo CD menu option](documentation/images/argo-cd-login.png?raw=true "OpenShift Gitops menu")
3) If your deployment was successful, you should see several applications running, such as "all-components-staging", "gitops", and so on.

#### Post-bootstrap Service Provider Integration(SPI) Configuration

SPI requires Service Provider to have configured OAuth application so it can process the OAuth flow. Follow [Configuring Service Providers](https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/docs/ADMIN.md#configuring-service-providers) in SPI admin documentation.

> Authorization URL of staging server: `https://spi-oauth-spi-system.apps.appstudio-stage.x99m.p1.openshiftapps.com`  
Callback URL of staging server: `https://spi-oauth-spi-system.apps.appstudio-stage.x99m.p1.openshiftapps.com/oauth/callback`

> **NOTE:**  Following process is automated in `preview` mode

SPI components fails to start right after the bootstrap. It requires manual configuration in order to work properly:

1) Edit `./components/spi/base/config.yaml` [see SPI Configuraton Documentation](https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/docs/ADMIN.md#configuration).
2) Create a `shared-configuration-file` Secret (`kubectl create secret generic `shared-configuration-file` --from-file=components/spi/base/config.yaml -n spi-system`)
3) In few moments, SPI pods should start

SPI Vault instance has to be manually initialized. There is a script to help with that:

1) Make sure that your cluster user has at least permissions `./components/authentication/spi-vault-admin.yaml`
2) Clone SPI operator repo `git clone https://github.com/redhat-appstudio/service-provider-integration-operator && cd service-provider-integration-operator`
3) run `vault-init.sh` script from repo root directory `./hack/vault-init.sh`

### Optional: Install Toolchain (Sandbox) Operators

This part is automated if you use `--toolchain` parameter of `hack/bootstrap-cluster.sh`

There are two scripts which you can use:

- `./hack/sandbox-development-mode.sh` for development mode
- `./hack/sandbox-e2e-mode.sh` for E2E mode

Both of the scripts will:

1. Automatically reduce the resources.requests.cpu values in argocd/openshift-gitops resource.
2. Install & configure the Toolchain (Sandbox) operators in the corresponding mode.
3. Print:
    - The landing-page URL that you can use for signing-up for the Sandbox environment that is running in your cluster.
    - Proxy URL.

#### SSO

This part is automated if you use `--toolchain --keycloak` parameters of `hack/bootstrap-cluster.sh`. These parameters install toolchain operators (`./hack/sandbox-development-mode.sh`) and configure them to use keycloak, which is automatically deployed as part of `development` overlay.

In development mode, the Toolchain Operators are configured to use Keycloak instance that is internally used by the Sandbox team. If you want to reconfigure it to use your own Keycloak instance, you need to add a few parameters to `ToolchainConfig` resource in `toolchain-host-operator` namespace.
This is an example of the needed parameters and their values:

```yaml
spec:
  host:
    registrationService:
      auth:
        authClientConfigRaw: '{
                  "realm": "sandbox-dev",
                  "auth-server-url": "https://sso.devsandbox.dev/auth",
                  "ssl-required": "none",
                  "resource": "sandbox-public",
                  "clientId": "sandbox-public",
                  "public-client": true
                }'
        authClientLibraryURL: https://sso.devsandbox.dev/auth/js/keycloak.js
        authClientPublicKeysURL: https://sso.devsandbox.dev/auth/realms/sandbox-dev/protocol/openid-connect/certs
      registrationServiceURL: <The landing page URL>
```

### Optional: CodeReady Containers Post-Bootstrap Configuration

Even with 6 CPU cores, you will need to reduce the CPU resource requests for each App Studio application. Either run `./hack/reduce-gitops-cpu-requests.sh` which will set resources.requests.cpu values to 50m or use `kubectl edit argocd/openshift-gitops -n openshift-gitops` to reduce the values to some other value. More details are in the FAQ below.

## Preview mode for your clusters

Once you bootstrap a cluster without `preview` argument, the root ArgoCD Application and all of the component applications will each point to the upstream repository. Or you can bootstrap cluster directly in mode which you need.

To enable development for a team or individual to test changes on your own cluster, you need to replace the references to `https://github.com/redhat-appstudio/infra-deployments.git` with references to your own fork.

There are a set of scripts that help with this, and minimize the changes needed in your forks.

There is a development configuration in `argo-cd-apps/overlays/development` which includes a kustomize overlay that can redirect the default components individual repositories to your fork.
The script also supports branches automatically. If you work in a checked out branch, each of the components in the overlays will mapped to that branch by setting `targetRevision:`.

Preview mode works in a feature branch, apply script which creates new preview branch and create additional commit with customization.

### Setting Preview mode

Steps:

1) Copy `hack/preview-template.env` to `hack/preview.env` and update new file based on instructions. File `hack/preview.env` should never be included in commit.
2) Work on your changes in a feature branch
3) Commit your changes
4) Run `./hack/preview.sh`, which will do:
  a) New branch is created from your current branch, the name of new branch is `preview-<name-of-current-branch>`
  b) Commit with changes related to your environment is added into preview branch
  c) Preview branch is pushed into your fork
  d) ArgoCD is set to point to your fork and the preview branch
  e) User is switched back to feature branch to create additional changes

If you want to reset your environment you can run the `hack/util-update-app-of-apps.sh https://github.com/redhat-appstudio/infra-deployments.git staging main` to reset everything including your cluster to `https://github.com/redhat-appstudio/infra-deployments.git` and match the upstream config.

Note running these scripts in a clone repo will have no effect as the repo will remain `https://github.com/redhat-appstudio/infra-deployments.git`

## End-to-End Tests

The E2E test suite can be run against a properly bootstrapped cluster. Please refer to [this repo](https://github.com/redhat-appstudio/e2e-tests) for details on how to build and run the tests.

[Tests are executed for each PR created .ci](.ci/Readme.md).

## Authentication

Authentication is managed by [components/authentication](components/authentication/). Authentication is disabled in preview mode.

For access to Stage cluster the github user has to be part of `stage` team in `redhat-appstudio-sre` organization.

Access to namespaces is managed by [components/authentication](components/authentication/) where `User` is github account and `Group` is team of `redhat-appstudio` organization.

Users can be added to organizations by Michal Kovarik <mkovarik@redhat.com> and by Shoubhik Bose <shbose@redhat.com>.

## Monitoring

Described in [components/monitoring](components/monitoring/README.md)

## App Studio/HACBS Build System

Described in [components/build-service](components/build-service/README.md)

## GitOps Service

Described in [components/gitops](components/gitops/README.md)

## Quality Dashboard

Quality dashboard is managed by `components/quality-dashboard`.

By default the frontend is using AppStudio Staging cluster for backend. If you want to use backend on your cluster you need to set secrets for `rds-endpoint`, `POSTGRES_PASSWORD` and `github-token` in `quality-dashboard` namespace and also push `components/quality-dashboard/frontend/kustomization.yaml`(the route to backend is changed by script `hack/util-set-quality-dashboard-backend-route.sh` in development and preview cluster modes).

More information about quality-dashboard you can found in repo: `https://github.com/redhat-appstudio/quality-dashboard`.

## Setting up observability stack

Script to set up observability stack (i.e. prometheus and grafana)
- [hack/setup_observability.sh](hack/setup_observability.sh)
This script requires few things
* [Github oauth](https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps) tokens for authentication of the components
* [Github Cookie secret](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview)
* [oc](https://docs.openshift.com/container-platform/4.11/cli_reference/openshift_cli/getting-started-cli.html) binary installed and configured to have admin access to the cluster

for running the `hack/setup_observability.sh` script
1. Copy `hack/monitoring-template.env` to `hack/monitoring.env`
2. Update the values for the variables in `hack/monitoring.env` from github oauth
2. ```$ ./hack/setup_observability.sh```

## FAQ

Other questions? Ask on `#wg-developer-appstudio`.

### Q: How do I deliver K8s resources in stages? For example, installing a Custom Resource Definition (CRD) first, then installing the Custom Resource (CR) for that CRD.

As long as your resources are declaratively defined, they will eventually be reconciled with the cluster (it just may take Argo CD a few retries). For example, the CRs might fail before the CRDs are applied, but on retry the CRDs will now exist (as they were applied during the previous retry). So now those CRs can progress.

_However_, this is not true if you are installing an Operator (e.g. Tekton) via OLM `Subscription`, and then using an operand of that operator (e.g. `Pipeline` CRs), at the same time. In this case, you will need to add the `argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true` annotation to the operands (or add it to the parent `kustomization.yaml`).

See the FAQ question '_the server could not find the requested resource_' below for details.

For finer-grained control of resource apply ordering, use Argo CD [Sync waves](https://argoproj.github.io/argo-cd/user-guide/sync-waves/) (Here is an [example](https://github.com/argoproj/argocd-example-apps/tree/master/sync-waves)).

### Q: What if I want my service's K8s resources in a separate Git repository? (i.e. not this one)

Ultimately, as a team, we should decide on a resource deployment strategy going forward: whether we want every team's K8s resources to be defined within this repository (as a GitOps monorepo), or within individual team's Git repositories. IMHO it is easiest to coordinate deployments within a single Git repository (such as this one), rather than multiple independent repositories.

However, if one or more services want to split off their K8s resource into independent repositories owned by those teams, they can modify the Argo CD `Application` CR for their service (created in 'How to add your own component' step 3, above) to point to their new repository.

### Q: How can I install an Operator to the cluster using Argo CD?

To install an operator, you only need to include the OLM `Subscription` and `OperatorGroup` CRs for the operator under your deployed resources folder within this repository. If the operator is not available in OperatorHub, then you need to include also the OLM `CatalogSource` CR.

For an example of this, see the [Red Hat CoP GitOps catalog](https://github.com/redhat-cop/gitops-catalog), for example the [Web Terminal operator example](https://github.com/redhat-cop/gitops-catalog/blob/main/web-terminal-operator/base/operator/web-terminal-subscription.yaml).

### Q: When Argo CD attempts to synchronize my Argo CD Application, I am seeing 'the server could not find the requested resource' sync error on my custom resources. How can I fix this?

Before Argo CD attempts a synchronize operation (syncing your Git repository with the K8s cluster), it performs a dry-run to ensure that all the K8s resources in your Git repository are valid. If your repository contains custom resources which are not yet defined (for example, Tekton `Pipeline` CRs), it will refuse to begin the synchronize operation.

This most often occurs when a Git repository contains both the OLM `Subscription` (which will install the desired operator, e.g. Tekton), and also the operands of that operator (the `Pipeline` CRs).

The easiest way to solve this is to add this annotation to your custom resources operands:  `argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true`.

For example, we would add this annotation to _all of our_ Pipeline CRs:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  # Add this annotation to your CRs:
  annotations:
    argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true
```

If you are using Kustomize, you can place the following in your `kustomization.yaml` file to automatically add it to all resources:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- # (your resources)

# Add these lines to your kustomization.yaml:
commonAnnotations:
  argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true
```

See the Argo CD docs [for more on this sync option](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-options/#skip-dry-run-for-new-custom-resources-types). See the [redhat-cop/gitops-catalog](https://github.com/redhat-cop/gitops-catalog) for examples of this option used with Kustomize and OLM-installed operators.

### Q: Logging into ArgoCD stops working and shows message: Failed to query provider.

If you stopped and started your cluster, a timing issue might cause the following message to appear
when you try logging into ArgoCD with `Log In Via OpenShift`:

```
Failed to query provider "https://openshift-gitops-server-openshift-gitops.apps.myserver.mydomain.com/api/dex": Get "http://openshift-gitops-dex-server.openshift-gitops.svc.cluster.local:5556/api/dex/.well-known/openid-configuration": dial tcp 1##.###.###.###:5556: connect: connection refused
```

To correct this problem:

+ Delete the openshift-gitops-dex-server-* pod:
```
oc delete pods -n openshift-gitops -l app.kubernetes.io/name=openshift-gitops-dex-server
```
+ The pod will automatically restart and ArgoCD `Log In Via OpenShift` should be working again.

### Q: What is the recommended memory and CPU allocation for CodeReady Containers for development purposes?

We recommend 7+ cores and 24+ GiB (24576 MiB) of memory.

### Q: When using CodeReady Containers for development purposes, I am getting an error message similar to: `0/1 nodes available: insufficient memory`.

The default worker node memory allocation of 8192 MiB insufficient to run App Studio. Increase the memory to 16 MiB using `crc config set memory 16384` and then create a new CRC VM to apply your changes, using `crc delete` and `crc start`. Finally, repeat the cluster bootstrapping process.

See the CodeReady Containers docs [for more on this configuration option](https://access.redhat.com/documentation/en-us/red_hat_codeready_containers/1.7/html/getting_started_guide/configuring-codeready-containers_gsg).

### Q: When using CodeReady Containers for development purposes, I am getting an error message similar to: `0/1 nodes available: insufficient cpu`.

The default 4-CPU allocation will not be sufficient for the CPU resource requests in this repo. Increase number of cores, for example, `crc config set cpus 6` if your hardware supports it, and then create a new CRC VM to apply your changes, using `crc delete` and `crc start`. Finally, repeat the cluster bootstrapping process.

See the CodeReady Containers docs [for more on this configuration option](https://access.redhat.com/documentation/en-us/red_hat_codeready_containers/1.7/html/getting_started_guide/configuring-codeready-containers_gsg).

Even with 6 CPU cores, you will need to reduce the CPU resource requests for each App Studio application. Using `kubectl edit argocd/openshift-gitops -n openshift-gitops`, reduce the resources.requests.cpu values from 250m to 100m or less. For example, change each line with

```
requests:
    cpu: 250m
```

to

```
requests:
    cpu: 100m
```

Then [save and exit the editor](https://vim.rtorr.com/). The updates will be applied to the cluster immediately, and the App Studio deployment should complete within a few minutes.

## For Members and Maintainers

### How to add yourself as a reviewer/approver
There is an OWNERS file present in each component folder [like this](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/spi/OWNERS), people mentioned in the file have the respective access to approve/review PR's.

To add yourself change the OWNERS file present in your component folder and Raise a pull request, if you want to be a Approver for the entire repo please change the OWNERS file present in the root level of this repository

Difference Between [Reviewers](https://github.com/kubernetes/community/blob/master/community-membership.md#reviewer) and [Approvers](https://github.com/kubernetes/community/blob/master/community-membership.md#approver)

More about code review using [OWNERS](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review-using-owners-files)
