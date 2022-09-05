# AppStudio Infrastructure Deployments

This repository is an initial set of Argo-CD-based deployments of AppStudio components to a cluster and a kcp workspace, plus a script to bootstrap Argo CD onto that cluster (to drive these Argo-CD-based deployments, via OpenShift GitOps) and configure AppStudio kcp workspaces.

This repository is structured as a GitOps monorepo (e.g. the repository contains the K8s resources for *multiple* applications), using [Kustomize](https://kustomize.io/).

The contents of this repository are not owned by any single individual, and should instead be collectively managed and maintained through PRs by individual teams.

## How to add your own component

You may use the `has` component as an example for how to add your own component. Here `has` refers to the HAS service team's K8s resources.

These are the steps to add your own component:

1) Create a new directory for your team's components, under `components/(team-name)`.
2) Add a `kustomization.yaml` file under that directory, which points to the individual K8s YAML resources you wish to deploy.
    - You may also structure your deployment into directories and files. See the Kustomize documentation for more information, and/or examples below.
    - See `components/has/staging` for an example of this.
3) Create an Argo CD `ApplicationSet` resource in `argo-cd-apps/base/(team-name).yaml`).
    - See `has.yaml` for a template of how this should look.
    - The `.spec.template.spec.source.path` value should point to the directory you created in previous step.
    - The `.spec.template.spec.destination.namespace` should match the target namespace you wish to deploy your resources to.
    - The suffix of the `.spec.template.metadata.name` should correspond to your team name, but keep the `{{kcp-name}}-` prefix for proper templating: `{{kcp-name}}-(team-name)`.
4) Add a reference to your new `(team-name).yaml` file, to `argo-cd-apps/base/kustomization.yaml` (the reference to your YAML file should be in the `resources:` list field).
5) Run `kustomize build (repo root)/argo-cd-apps/overlays/staging` and ensure it passes, and outputs your new Argo CD Application CR.
6) Add an entry in `argo-cd-apps/overlays/development/repo-overlay.yaml` for your new component so you can use the preview mode for testing.
7) Open a PR for all of the above.

More examples of using Kustomize to drive deployments using GitOps can be [found here](https://github.com/redhat-cop/gitops-catalog).

## Maintaining your components

Simply update the files under `components/(team-name)`, and open a PR with the changes.

**TIP**: For development purposes, you can use `kustomize build .` to output the K8s resources that are being generated for your folder.

## Bootstrapping a cluster

### Required prerequisites

The prerequisites are:

- You must have `kubectl`, `oc`, `jq` and [`yq`](https://github.com/mikefarah/yq) installed.
- You must have a `kubeconfig` pointing to an existing OpenShift cluster, that you wish to use. Alternatively, you can configure a local CodeReady Containers VM to use.
- You must have another `kubeconfig` pointing to an existing kcp instance, that you wish to deploy to. You can use either a CPS or a local kcp instance.
- The script `./hack/setup/install-pre-req.sh` will install these prerequisites for you, if they're not already installed.

### Optional: CodeReady Containers Setup

If you don't already have a test OpenShift cluster available, CodeReady Containers is a popular option. It runs a small OpenShift cluster in a single VM on your local workstation.

1) Create or log in using your free Red Hat account, and [install CodeReady Containers (CRC)](https://console.redhat.com/openshift/create/local).
2) Make sure you have the latest version of CRC: `crc version`
3) Run `./hack/setup/prepare-crc.sh` to configure CodeReady Containers with the recommended minimum memory (16 GiB) and CPUs (6) for App Studio. The script has optional parameters for customizing `memory` and `cpu` allowance. It also supports `force delete` of existing cluster. Run `./hack/setup/prepare-crc.sh --help` to see the options. The script will also enable cluster monitoring and log you in as the cluster administrator.

### Optional: Quicklab storage setup for clusters

If you are using Quicklab to provision your development cluster, you will need to setup storage prior to running the bootstrap script.

See [hack/quicklab/README.md](hack/quicklab/README.md)

### Bootstrap App Studio

To boostrap AppStudio run:
```bash
./hack/bootstrap.sh -kk [kubeconfig-pointing-to-kcp] -ck [kubeconfig-pointing-to-openshift] -rw [workspace-to-be-used-as-root] -m [mode|upstream,dev,preview]
```
which will:
* Bootstrap Argo CD (using OpenShift GitOps) - it will output the Argo CD Web UI route when it's finished.
* Create kcp workspaces and SyncTarget pointing to the OpenShift cluster.
* Create an ArgoCD representation of a cluster pointing to `redhat-appstudio` workspace (as a secret in ArgoCD format).
* Setup the Argo CD `Application`/`ApplicationSet` Custom Resources (CRs) for each component.

#### Modes:
* `upstream` (default) mode will expect access to both CPS instances `kcp-stable` and `kcp-unstable` - each of them should be represented by a kubeconfig context having the same name as the CPS instance.  
* `dev` mode will use one kcp instance as the deployment target - it can be any instance (either CPS or local kcp). The current kubeconfig context should point to it.  
* `preview` mode will enable preview mode used for development and testing on non-production clusters using the same deployment target as `dev` mode. See [Preview mode for your clusters](#preview-mode-for-your-clusters).

#### Workspaces:
If `-rw | --root-workspace` parameter is not specified, then by default, all workspaces are automatically created under the `root` workspace.  
There are two workspaces created per kcp instance:
* `redhat-appstudio-internal-compute` - This is the workspace where the SyncTarget for the OpenShift workload cluster is configured. If the root workspace is different from `root`, then the name of the workspace is set to `compute` to work around [this issue](https://github.com/kcp-dev/kcp/issues/1843). (The name of the workspace can be overridden by setting the `COMPUTE_WORKSPACE` variable) 
* `redhat-appstudio` - In this workspace ArgoCD deploys all kcp-related manifests from the infra-deployments repository. It's the place where all AppStudio components run. (The name of the workspace can be overridden by setting the `APPSTUDIO_WORSKPACE` variable)  

#### Configure kcp for upstream mode:
If you decide to run the upstream mode, then the `bootstrap.sh` script tries to configure two instances of kcp: `kcp-stable` and `kcp-unstable`. However, `kcp-stable` instance may require different version of kubectl kcp plugin than the `kcp-unstable` one. This makes running the bootstrap script impossible for the upstream mode, because you cannot use two versions of the plugin at the same time.

To work around the issue, you can skip the configuration of the kcp part by using `-sk | --skip-kcp parameter <true/false>`:  
```bash
./hack/bootstrap.sh -sk true ...
```
The bootstrap.sh script then configures only ArgoCD (including the Application/ApplicationsSets) in the workload cluster and doesn't do anything for kcp.

To configure the kcp instances separately use the `configure-kcp.sh` script:
```bash
./hack/configure-kcp.sh -kk [kubeconfig-pointing-to-kcp] -ck [kubeconfig-pointing-to-openshift] -rw [workspace-to-be-used-as-root] -kn [kcp-name|kcp-stable,kcp-unstable,dev]
```
which takes care of creation of the workspaces, SyncTarget, and the representation of the cluster in ArgoCD for the given kcp instance. So to fully finish the upstream configuration run the script for both kcp instances using the parameter `-kn kcp-stable` and `-kn kcp-unstable`.

#### Access Argo CD Web UI
Open the Argo CD Web UI to see the status of your deployments. You can use the route from the previous step and login using your OpenShift credentials (using the 'Login with OpenShift' button), or login to the OpenShift Console and navigate to Argo CD using the OpenShift Gitops menu in the Applications pulldown.
   ![OpenShift Gitops menu with Cluster Argo CD menu option](documentation/images/argo-cd-login.png?raw=true "OpenShift Gitops menu")

If your deployment was successful, you should see several applications running, such as "all-components", "has", and so on.

### Optional: CodeReady Containers Post-Bootstrap Configuration

Even with 6 CPU cores, you will need to reduce the CPU resource requests for each App Studio application. Either run `./hack/reduce-gitops-cpu-requests.sh` which will set resources.requests.cpu values to 50m or use `kubectl edit argocd/openshift-gitops -n openshift-gitops` to reduce the values to some other value. More details are in the FAQ below.

## Preview mode for your clusters

Once you bootstrap your environment without `preview` argument, the root ArgoCD Application and all of the component applications will each point to the upstream repository. Or you can bootstrap cluster directly in mode which you need.

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

## Authorization

Authorization is managed by [components/authorization](components/authorization/). Authorization is disabled in dev and preview mode.

For access to OpenShift Stage cluster the github user has to be part of `stage` team in `redhat-appstudio-sre` organization.

Authorization in `root:redhat-appstudio` workspace in CPS is managed by [components/authorization/kcp/members-view.yaml](components/authorization/kcp/members-view.yaml).

## FAQ

Other questions? Ask on `#wg-developer-appstudio`.

### Q: What if I want my service's K8s resources in a separate Git repository? (i.e. not this one)

Ultimately, as a team, we should decide on a resource deployment strategy going forward: whether we want every team's K8s resources to be defined within this repository (as a GitOps monorepo), or within individual team's Git repositories. IMHO it is easiest to coordinate deployments within a single Git repository (such as this one), rather than multiple independent repositories.

However, if one or more services want to split off their K8s resource into independent repositories owned by those teams, they can modify the Argo CD `ApplicationSet` CR for their service (created in 'How to add your own component' step 3, above) to point to their new repository.

### Q: When Argo CD attempts to synchronize my Argo CD Application, I am seeing 'the server could not find the requested resource' sync error on my custom resources. How can I fix this?

Before Argo CD attempts a synchronize operation (syncing your Git repository with the K8s cluster), it performs a dry-run to ensure that all the K8s resources in your Git repository are valid. If your repository contains custom resources which are not yet defined, it will refuse to begin the synchronize operation.

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
