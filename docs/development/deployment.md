---
title: Bootstrap StoneSoup
---

The following steps will help you creating an environment for developing StoneSoup.
For development purposes, all the components will be deployed into a single cluster,
as opposed to a [production deployment](../deployment/multi-cluster.md) where multiple clusters are needed.

The deployment of StoneSoup's components is done via ArgoCD. The script mentioned below, will deploy an
ArgoCD instance to the [development cluster](./pre.md#bootstrapping-a-cluster) and configure it to
deploy StoneSoup.


## Getting Started

***Note:** Running the bootstrap script in preview mode with `./hack/bootstrap-cluster.sh preview` will deploy the environment with your local feature branch **committed** changes. To run in preview mode, the required fields in `./hack/preview.env` must be completed. This includes API tokens for both GitHub and Quay. See [Preview mode for your clusters](#preview-mode-for-your-clusters) for details before starting these steps.*

### Steps:

1. Run `./hack/bootstrap-cluster.sh [preview]`, which will bootstrap Argo CD (using OpenShift GitOps) and setup the Argo CD `Application` Custom Resources (CRs) for each component. This command will output the Argo CD Web UI route when it's finished. `preview` will enable preview mode used for development and testing on non-production clusters, described in section [Preview mode for your clusters](#preview-mode-for-your-clusters).

2. Open the Argo CD Web UI to see the status of your deployments. You can use the route from the previous step and login using your OpenShift credentials (using the 'Login with OpenShift' button), or login to the OpenShift Console and navigate to Argo CD using the OpenShift Gitops menu in the Applications pulldown.
![OpenShift Gitops menu with Cluster Argo CD menu option](./argo-cd-login.png?raw=true "OpenShift Gitops menu")

3. If your deployment was successful, you should see several applications running.

## Upstream mode

When you bootstrap a cluster without `preview` argument, the root ArgoCD Application and all of the component applications will each point to the upstream repository. Or you can bootstrap cluster directly in mode which you need.

To enable development for a team or individual to test changes on your own cluster, you need to replace the references to `https://github.com/redhat-appstudio/infra-deployments.git` with references to your own fork.

There are a set of scripts that help with this, and minimize the changes needed in your forks.

The [development configuration](https://github.com/redhat-appstudio/infra-deployments/tree/main/argo-cd-apps/overlays/development) includes a kustomize overlay that can redirect the default components individual repositories to your fork.
The script also supports branches automatically. If you work in a checked out branch, each of the components in the overlays will mapped to that branch by setting `targetRevision:`.

## Preview mode for your clusters

Preview mode works from a local feature branch for testing. The `./hack/preview.sh` script creates a new `preview-<name-of-current-branch>` branch and pushes it to your GitHub repo. This includes an additional commit with customizations, including for your ArgoCD environment to load applications from your GitHub repository.

### Setting Preview mode

1. Copy `./hack/preview-template.env` to `./hack/preview.env` and update new file based on the instructions within. **The file `./hack/preview.env` should never be included in a commit.**

2. Work on your changes in a feature branch.

3. Commit your changes.

4. Run `./hack/preview.sh`, which will do the following:

    1. A new branch is created from your current branch, the name of new branch is `preview-<name-of-current-branch>`.

    2. A commit with changes related to your environment is added into the preview branch.

    3. The preview branch is pushed into your GitHub fork.

    4. The ArgoCD configuration is set to point to your fork and the preview branch

  4. User is switched back to feature branch to create additional changes

***Note:** The `./hack/preview.sh` script is run automatically at the end of the `./hack/bootstrap-cluster.sh preview` script. Once the cluster is bootstrapped, you just need to run `./hack/preview.sh` again to update your environment with new committed changes from your feature branch.*

~~If you want to reset your environment you can run the `hack/util-update-app-of-apps.sh https://github.com/redhat-appstudio/infra-deployments.git staging main` to reset everything including your cluster to `https://github.com/redhat-appstudio/infra-deployments.git` and match the upstream config.~~

Note running these scripts in a clone repo will have no effect as the repo will remain `https://github.com/redhat-appstudio/infra-deployments.git`

### Storage for Persistent Volume Claims

The PVCs for the deployment need a default StorageClass and available PVs or automation to create those PVs. You can use [Configuring NFS storage provisioner on QuickCluster clusters](../../hack/quickcluster/README.html) to connect to an existing NFS provider.

Another option for a stand-alone test environment is to use the **Local Storage Operator** to create a **LocalVolumeSet** from available disks on your worker nodes. Use the name `managed-nfs-storage` for the LocalVolumeSet and the StorageClass to match the deployment expectations, and annotate the StorageClass with `storageclass.kubernetes.io/is-default-class=true` to set it as the default. Having around 6 or more available PVs of 8GB or greater is a good simple starting point, but YMMV.

## Optional: OpenShift Local Post-Bootstrap Configuration

Even with 6 CPU cores, you will need to reduce the CPU resource requests for each StoneSoup application. Either run `./hack/reduce-gitops-cpu-requests.sh` which will set resources.requests.cpu values to 50m or use `kubectl edit argocd/openshift-gitops -n openshift-gitops` to reduce the values to some other value. More details are in the FAQ below.
