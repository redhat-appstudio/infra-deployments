---
title: Bootstrap StoneSoup
---

The following steps will help you creating an environment for developing StoneSoup.

## Getting Started

Steps:

1. Run `./hack/bootstrap-cluster.sh [preview]` which will bootstrap Argo CD (using OpenShift GitOps) and setup the Argo CD `Application` Custom Resources (CRs) for each component. This command will output the Argo CD Web UI route when it's finished. `preview` will enable preview mode used for development and testing on non-production clusters, described in section [Preview mode for your clusters](#preview-mode-for-your-clusters).

2. Open the Argo CD Web UI to see the status of your deployments. You can use the route from the previous step and login using your OpenShift credentials (using the 'Login with OpenShift' button), or login to the OpenShift Console and navigate to Argo CD using the OpenShift Gitops menu in the Applications pulldown.
![OpenShift Gitops menu with Cluster Argo CD menu option](./argo-cd-login.png?raw=true "OpenShift Gitops menu")

3. If your deployment was successful, you should see several applications running, such as "all-components-staging", "gitops", and so on.

## Preview mode for your clusters

Once you bootstrap a cluster without `preview` argument, the root ArgoCD Application and all of the component applications will each point to the upstream repository. Or you can bootstrap cluster directly in mode which you need.

To enable development for a team or individual to test changes on your own cluster, you need to replace the references to `https://github.com/redhat-appstudio/infra-deployments.git` with references to your own fork.

There are a set of scripts that help with this, and minimize the changes needed in your forks.

The [development configuration](../../argo-cd-apps/overlays/development) includes a kustomize overlay that can redirect the default components individual repositories to your fork.
The script also supports branches automatically. If you work in a checked out branch, each of the components in the overlays will mapped to that branch by setting `targetRevision:`.

Preview mode works in a feature branch, apply script which creates new preview branch and create additional commit with customization.

### Setting Preview mode

Steps:

1. Copy `hack/preview-template.env` to `hack/preview.env` and update new file based on instructions. File `hack/preview.env` should never be included in commit.

2. Work on your changes in a feature branch

3. Commit your changes

4. Run `./hack/preview.sh`, which will do:
  1. New branch is created from your current branch, the name of new branch is `preview-<name-of-current-branch>`

  2. Commit with changes related to your environment is added into preview branch

  3. Preview branch is pushed into your fork

  4. ArgoCD is set to point to your fork and the preview branch

  4. User is switched back to feature branch to create additional changes

If you want to reset your environment you can run the `hack/util-update-app-of-apps.sh https://github.com/redhat-appstudio/infra-deployments.git staging main` to reset everything including your cluster to `https://github.com/redhat-appstudio/infra-deployments.git` and match the upstream config.

Note running these scripts in a clone repo will have no effect as the repo will remain `https://github.com/redhat-appstudio/infra-deployments.git`

## Optional: OpenShift Local Post-Bootstrap Configuration

Even with 6 CPU cores, you will need to reduce the CPU resource requests for each StoneSoup application. Either run `./hack/reduce-gitops-cpu-requests.sh` which will set resources.requests.cpu values to 50m or use `kubectl edit argocd/openshift-gitops -n openshift-gitops` to reduce the values to some other value. More details are in the FAQ below.