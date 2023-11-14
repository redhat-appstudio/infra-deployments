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

Steps:

1. Run `./hack/bootstrap-cluster.sh [preview]` which will bootstrap Argo CD (using OpenShift GitOps) and setup the Argo CD `Application` Custom Resources (CRs) for each component. This command will output the Argo CD Web UI route when it's finished. `preview` will enable preview mode used for development and testing on non-production clusters, described in section [Preview mode for your clusters](#preview-mode-for-your-clusters).

2. Open the Argo CD Web UI to see the status of your deployments. You can use the route from the previous step and login using your OpenShift credentials (using the 'Login with OpenShift' button), or login to the OpenShift Console and navigate to Argo CD using the OpenShift Gitops menu in the Applications pulldown.
![OpenShift Gitops menu with Cluster Argo CD menu option](./argo-cd-login.png?raw=true "OpenShift Gitops menu")

3. If your deployment was successful, you should see several applications running.

## Preview mode for your clusters

Once you bootstrap a cluster without `preview` argument, the root ArgoCD Application and all of the component applications will each point to the upstream repository. Or you can bootstrap cluster directly in mode which you need.

To enable development for a team or individual to test changes on your own cluster, you need to replace the references to `https://github.com/redhat-appstudio/infra-deployments.git` with references to your own fork.

There are a set of scripts that help with this, and minimize the changes needed in your forks.

The [development configuration](https://github.com/redhat-appstudio/infra-deployments/tree/main/argo-cd-apps/overlays/development) includes a kustomize overlay that can redirect the default components individual repositories to your fork.
The script also supports branches automatically. If you work in a checked out branch, each of the components in the overlays will mapped to that branch by setting `targetRevision:`.

Preview mode works in a feature branch, apply script which creates new preview branch and create additional commit with customization.

### Setting Preview mode

Steps:

1. Copy `hack/preview-template.env` to `hack/preview.env` and update new file based on [instructions](#previewenv-instructions).
   File `hack/preview.env` should never be included in commit.

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

### preview.env instructions

I want a development deployment of StoneSoup where I can:

* Run simple builds
* Onboard a repository to Pipelines as Code

What environment variables do I need to set in `preview.env` before I run the preview script?
How to set up all the prerequisites?

| Variable                      | How to set up                     |
| ----------------------------- | --------------------------------- |
| `MY_GIT_FORK_REMOTE`          | [Fork the repo](#fork-the-repo)   |
| `MY_GITHUB_ORG`               | [GitHub setup](#github-setup)     |
| `MY_GITHUB_TOKEN`             | [GitHub setup](#github-setup)     |
| `IMAGE_CONTROLLER_QUAY_ORG`   | [Quay setup](#quay-setup)         |
| `IMAGE_CONTROLLER_QUAY_TOKEN` | [Quay setup](#quay-setup)         |
| `PAC_GITHUB_APP_PRIVATE_KEY`  | [PaC setup](#pac-setup)           |
| `PAC_GITHUB_APP_ID`           | [PaC setup](#pac-setup)           |

#### Fork the repo

If you haven't done so yet, fork <https://github.com/redhat-appstudio/infra-deployments> and clone
it locally. Copy [`hack/preview-template.env`][preview-template.env] to `hack/preview.env` - that's
where you want to set all the environment variables.

The first one is `MY_GIT_FORK_REMOTE` - the name of the git remote which points to your fork.

```shell
$ git remote -v
origin  git@github.com:<gh_username>/infra-deployments.git (fetch)
origin  git@github.com:<gh_username>/infra-deployments.git (push)
upstream    https://github.com/redhat-appstudio/infra-deployments.git (fetch)
upstream    https://github.com/redhat-appstudio/infra-deployments.git (push)
```

Set the name of the git remote in `preview.env`:

```shell
export MY_GIT_FORK_REMOTE=origin
```

#### GitHub setup

Create a new organization, see the [GitHub docs: Creating a new organization from scratch][github-create-org].
This is the organization where you will later install the Pipelines as Code application. When testing
the PaC integration features of StoneSoup, you will need to use repositories from this organization
(or fork existing repositories into this org).

Copy the organization name to `preview.env`. For example:

```shell
export MY_GITHUB_ORG=<gh_username>-stonesoup
```

Next, you will need an access token. See the [GitHub docs: Creating a personal access token][github-get-access-token].
The token needs all the `repo` permissions and the `delete_repo` permission.

Copy the token value to `preview.env`:

```shell
export MY_GITHUB_TOKEN=ghp_***
```

#### Quay setup

Create a new [Quay.io](https://quay.io) organization. If you need a new email address for the organization,
you can use the `+` trick - e.g. `<gmail_username>+quayorg@gmail.com`.

Copy the organization name to `preview.env`. For example:

```shell
export IMAGE_CONTROLLER_QUAY_ORG=<quay_username>-stonesoup
```

Next, you need to generate an access token for your Quay.io organization. See the instructions for
`IMAGE_CONTROLLER_QUAY_TOKEN` in [preview.env][preview-template.env]. Once you have created the token,
copy its value to `preview.env`:

```shell
export IMAGE_CONTROLLER_QUAY_TOKEN=***
```

#### PaC setup

Follow the [Pipelines as Code: Setup Manually][pac-setup-manual] instructions to create the GitHub application.
You can create the application in your organization settings (<https://github.com/organizations/MY_STONESOUP_ORG/settings/apps>)
or your personal settings (<https://github.com/settings/apps>). If you create the app under your organization,
you can allow it `Only on this account`, otherwise you will have to allow `Any account`.

Compared to the Pipelines as Code documentation, you will want to set these fields differently:

* **GitHub Application Name**: use a unique name, e.g. `<gh_username>-stonesoup PaC`
* **Webhook URL**: use a dummy url, e.g. <https://example.org>. The `preview.sh` script will fix the
  webhook url automatically.
* **Webhook secret**: leave it empty, the `preview.sh` script will set the secret automatically.

Once you have created the GitHub app, copy its App ID to `preview.env`:

```shell
export PAC_GITHUB_APP_ID=<number>
```

Generate a private key for the application (in the GitHub UI; refer to the Pipelines as Code docs linked
above). Encode the private key as base64 and copy it to `preview.env`:

```shell
base64 --wrap=0 <path to downloaded key>
```

```shell
export PAC_GITHUB_APP_PRIVATE_KEY=<output of the command above>
```

Finally, install the application for your organization. See the [GitHub docs: Installing your own
GitHub App][github-install-app]. Select `All repositories` when installing to make sure it will have
access to all the repos you're going to create/fork into your org in the future.

### Verifying your setup

**Simple build:**

* Fork <https://github.com/devfile-samples/devfile-sample-python-basic> into your GitHub organization.
* Run `hack/build/build-via-appstudio.sh https://github.com/MY_STONESOUP_ORG/devfile-sample-python-basic`

The script will create a test application and a component for you:

```shell
$ oc get application
NAME               AGE   STATUS   REASON
test-application   74s   True     OK

$ oc get component
NAME                          AGE   STATUS   REASON   TYPE
devfile-sample-python-basic   86s   True     OK       Created
```

Build-service should start a pipeline for your new component almost immediately:

```shell
$ tkn pipelinerun list
NAME                                STARTED         DURATION   STATUS
devfile-sample-python-basic-jpg29   2 minutes ago   ---        Running
```

You can also see the PipelineRun in the OpenShift console in your cluster.

**Pipelines as Code onboarding:**

```shell
$ oc annotate component devfile-sample-python-basic build.appstudio.openshift.io/request=configure-pac
component.appstudio.redhat.com/devfile-sample-python-basic annotated
```

Build-service should create a new pull request in your forked devfile-sample-python-basic repository.

If your cluster is accessible on the public internet, commenting `/ok-to-test` on the pull request
will trigger the on-pull-request PipelineRun. Merging the pull request will trigger the on-push PipelineRun.
If your cluster is hidden behind a VPN, this won't work.

### Testing code changes

#### Deploying your versions of operators

First, you will need to build the container image (typically with `make docker-build` in the operator
repository) and push the image to a publicly accessible container repository. Then, set the image
reference for the corresponding service in `preview.env`. For example, to override the build-service
image, set:

```shell
export BUILD_SERVICE_IMAGE_REPO=quay.io/<quay_username>/build-service
export BUILD_SERVICE_IMAGE_TAG=my-test-v1
```

Then, run the `hack/preview.sh` script, which will deploy the overriden image.

To update the operator after you've made some more changes, build a new image and push it to the same
repository with a new tag. Set the new tag in `preview.env` and run `hack/preview.sh` again.

#### Deploying your versions of CRDs

Find the reference to the upstream repository where your CRDs are located. For example, for build-service,
it's in [components/build-service/base/kustomization.yaml][build-service-kustomization]:

```yaml
resources:
- allow-argocd-to-manage.yaml
- https://github.com/redhat-appstudio/build-service/config/default?ref=99cebd0a67a6b25b8ccffb76522861f526c762de
```

Replace this reference with a reference to your fork and the commit you would like to test. Create
a new branch, commit the changes and run `hack/preview.sh`.

## Optional: OpenShift Local Post-Bootstrap Configuration

Even with 6 CPU cores, you will need to reduce the CPU resource requests for each StoneSoup application. Either run `./hack/reduce-gitops-cpu-requests.sh` which will set resources.requests.cpu values to 50m or use `kubectl edit argocd/openshift-gitops -n openshift-gitops` to reduce the values to some other value. More details are in the FAQ below.

[preview-template.env]: https://github.com/redhat-appstudio/infra-deployments/blob/main/hack/preview-template.env
[github-create-org]: https://docs.github.com/en/organizations/collaborating-with-groups-in-organizations/creating-a-new-organization-from-scratch
[github-get-access-token]: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-personal-access-token-classic
[github-install-app]: https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app
[pac-setup-manual]: https://pipelinesascode.com/docs/install/github_apps/#setup-manually
[build-service-kustomization]: https://github.com/redhat-appstudio/infra-deployments/blob/main/components/build-service/base/kustomization.yaml
