---
title: Extending The Service
---

If you want to add a new component that will be deployed as part of StoneSoup you are
in the right place. Please follow the steps below for adding you new component.

## Adding A Component

You may use the [gitops](../../components/gitops/) component as an example for how to add your own component. Here `gitops` refers to the GitOps service team's K8s resources.

These are the steps to add your own component:

1. Create a new directory for your team's components, under `components/(team-name)`.

2. Add a `kustomization.yaml` file under that directory, which points to the individual K8s YAML resources you wish to deploy.
    - You may also structure your deployment into directories and files. See the Kustomize documentation for more information, and/or examples below.
    - See `components/gitops/staging` for an example of this.

3. Create an Argo CD `Application` resource in `argo-cd-apps/base/(team-name).yaml`).
    - See `gitops.yaml` for a template of how this should look.
    - The `.spec.source.path` value should point to the directory you created in previous step.
    - The `.spec.destination.namespace` should match the target namespace you wish to deploy your resources to.
    - The `.metadata.name` should correspond to your `(team-name)`

4. Add a reference to your new `(team-name).yaml` file, to `argo-cd-apps/base/kustomization.yaml` (the reference to your YAML file should be in the `resources:` list field).

5. Run `kustomize build (repo root)/argo-cd-apps/overlays/staging` and ensure it passes, and outputs your new Argo CD Application CR.

6. Add an entry in `argo-cd-apps/overlays/development/repo-overlay.yaml` for your new component so you can use the preview mode for testing.

7. Open a PR for all of the above.

More examples of using Kustomize to drive deployments using GitOps can be [found here](https://github.com/redhat-cop/gitops-catalog).

## Component testing and building of images

[Pipelines as Code](https://pipelinesascode.com/) is deployed and available for testing and building of images.
To test and run builds for a component, add your github repository to `components/tekton-ci/repository.yaml`.

Target repository has to have installed GitHub app - [Red Hat Trusted App Pipeline](https://github.com/apps/red-hat-trusted-app-pipeline) and pipelineRuns created in `.tekton` folder, example [Build Service](https://github.com/redhat-appstudio/build-service/tree/main/.tekton). Target image repository in quay.io must exist and robot account `redhat-appstudio+production_tektonci` has to have `write` permission on the repository.


## Maintaining your components

Simply update the files under `components/(team-name)`, and open a PR with the changes.

**TIP**: For development purposes, you can use `kustomize build .` to output the K8s resources that are being generated for your folder.


## Authentication

Authentication is managed by [components/authentication](components/authentication/). Authentication is disabled in preview mode.

Access to namespaces is managed by [components/authentication](components/authentication/) where `User` is github account and `Group` is team of `redhat-appstudio` organization.

## For Members and Maintainers

### How to add yourself as a reviewer/approver
There is an OWNERS file present in each component folder [like this](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/spi/OWNERS), people mentioned in the file have the respective access to approve/review PR's.

To add yourself change the OWNERS file present in your component folder and Raise a pull request, if you want to be a Approver for the entire repo please change the OWNERS file present in the root level of this repository

Difference Between [Reviewers](https://github.com/kubernetes/community/blob/master/community-membership.md#reviewer) and [Approvers](https://github.com/kubernetes/community/blob/master/community-membership.md#approver)

More about code review using [OWNERS](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review-using-owners-files)
