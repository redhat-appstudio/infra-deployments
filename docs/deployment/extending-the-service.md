---
title: Extending The Service
---

If you want to add a new component that will be deployed as part of StoneSoup you are
in the right place. Please follow the steps below for adding you new component.

## Adding A Component

You may use the [gitops](../../components/gitops/) component as an example for how to add your own component. Here `gitops` refers to the GitOps service team's K8s resources.

These are the steps to add your own component:

1. Create a new directory for your team's components, under `components/(team-name)`.
    ```
    📂 INFRA-DEPLOYMENTS
        📂 argo-cd-apps
        📂 components
            📂 (team-name)  --> Your team-name
    ```

2. Add a `kustomization.yaml` file under that directory, which points to the individual K8s YAML resources you wish to deploy.
    - Depending on your application, you may also structure your deployment into directories and files. See the Kustomize documentation for more information, and/or examples below.
        - Exmaple: 1 (team-name directory containing application resources in its root e.g. `file-1.yaml` and `file-2.yaml`, they can be deployments, services, configmaps etc.)
            ```
            📂 INFRA-DEPLOYMENTS
                📂 argo-cd-apps
                📂 components
                    📂 (team-name)  --> Your team-name
                        📄 file-1.yaml
                        📄 file-2.yaml
                        📄 kustomization.yaml  --> This file points to file-1.yaml and file-2.yaml
            ```
            More information about kustomize fundamentals can be [found here](https://kubectl.docs.kubernetes.io/guides/introduction/kustomize/)

        - Example: 2 (Application with base and overlays in its root)
            ```
            📂 INFRA-DEPLOYMENTS
                📂 argo-cd-apps
                📂 components
                    📂 (team-name)  --> Your team-name
                        📂 base
                            📄 file-1.yaml
                            📄 file-2.yaml
                            📄 kustomization.yaml
                        📂 overlays
                            📂 development
                                📄 development-patch-1.yaml
                                📄 development-patch-2.yaml
                                📄 kustomization.yaml
                            📂 staging
                                📄 staging-patch-1.yaml
                                📄 staging-patch-2.yaml
                                📄 kustomization.yaml
                            📂 production
                                📄 production-patch-1.yaml
                                📄 production-patch-2.yaml
                                📄 kustomization.yaml
            ```

    - See `components/gitops/staging` for more complex structure, where overlays are further structured for cluster specific configurations.

3. Create an Argo CD `ApplicationSet` resource in `argo-cd-apps/base/directory/team-name/(team-name).yaml` or `argo-cd-apps/base/team-name/(team-name).yaml` depending on your application.
    - There are quite a few directories in `argo-cd-apps/` directory, such as `base/member/` (for member clusters), `base/eaas/` (for EaaS clusters), `base/all-clusters/` (for all the clusters) etc. Therefore, please choose the appropriate directory to create `ApplicationSet` for your application or create a new directory `team-name` if none of the existing directories suits your application.

    - See `argo-cd-apps/base/member/gitops/gitops.yaml` for a template of how `ApplicationSet` should look like.
    - The `.spec.template.spec.source.path` value should point to the directory you created in previous step.
    - The `.spec.template.spec.destination.namespace` should match the target namespace you wish to deploy your resources to.
    - The `.metadata.name` should correspond to your `(team-name)`.
>
4. Add a reference to your new `(team-name).yaml` file, to `argo-cd-apps/base/directory/team-name/kustomization.yaml` or `argo-cd-apps/base/team-name/kustomization.yaml` (the reference to your YAML file should be in the `resources:` list field).

    ```YAML
    apiVersion: kustomize.config.k8s.io/v1beta1
    kind: Kustomization
    resources:
    - team-name.yaml
    ```
5. Kustomize Components a new kind of Kustomization that allows users to define reusable kustomizations. Components can be included from higher-level overlays to create variants of an application, with a subset of its features enabled. We have `argo-cd-apps/k-components` directory in this reposiroty to place `kustomization.yaml` with `kind: Component`. 

    See `argo-cd-apps/k-components` for such example. 
    
    An example for Kustomize Components looks like below:

    ```YAML
    apiVersion: kustomize.config.k8s.io/v1alpha1
    kind: Component
    commonLabels:
      appstudio.redhat.com/host-cluster: "true"
    ```

    and `kustomization.yaml` referencing above looks like below:

    ```YAML
    apiVersion: kustomize.config.k8s.io/v1beta1
    kind: Kustomization
    resources:
      - ../base
    components:
      - ../../../k-components/assign-host-role-to-local-cluster
    ```

    >**Note:** A component cannot be added to the `resources:` list, and a resource/`Kustomization `cannot be added to the `components:` list

    You can find more information about *Kustomize Components* [here](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cli/1802-kustomize-components/README.md)

6. Run `kustomize build (repo root)/argo-cd-apps/overlays/staging` and ensure it passes, and outputs your new Argo CD Application CR.

7. Add an entry in `argo-cd-apps/overlays/development/repo-overlay.yaml` for your new component so you can use the preview mode for testing.

8. Open a PR for all of the above.

More examples of using Kustomize to drive deployments using GitOps can be [found here](https://github.com/redhat-cop/gitops-catalog).

## Component testing and building of images

[Pipelines as Code](https://pipelinesascode.com/) is deployed and available for testing and building of images.
To test and run builds for a component, add your github repository to `components/tekton-ci/repository.yaml` if you want to publish to quay.io/redhat-appstudio or `components/konflux-ci/repository.yaml` if you want to publish to quay.io/konflux-ci.

Target repository has to have installed GitHub app - [Red Hat Trusted App Pipeline](https://github.com/apps/red-hat-trusted-app-pipeline) and pipelineRuns created in `.tekton` folder, example [Build Service](https://github.com/konflux-ci/build-service/tree/main/.tekton). Target image repository in quay.io must exist and robot account `redhat-appstudio+production_tektonci` has to have `write` permission on the repository.


## Maintaining your components

Simply update the files under `components/(team-name)`, and open a PR with the changes.

**TIP**: For development purposes, you can use `kustomize build .` to output the K8s resources that are being generated for your folder.


## Authentication

Authentication is managed by [components/authentication](components/authentication/). Authentication is disabled in preview mode.

Access to namespaces is managed by [components/authentication](components/authentication/) where `User` is github account and `Group` is team of `redhat-appstudio` organization.

## For Members and Maintainers

### How to add yourself as a reviewer/approver
There is an OWNERS file present in each component folder [like this](https://github.com/redhat-appstudio/infra-deployments/blob/main/components/integration/OWNERS), people mentioned in the file have the respective access to approve/review PR's.

To add yourself change the OWNERS file present in your component folder and Raise a pull request, if you want to be a Approver for the entire repo please change the OWNERS file present in the root level of this repository

Difference Between [Reviewers](https://github.com/kubernetes/community/blob/master/community-membership.md#reviewer) and [Approvers](https://github.com/kubernetes/community/blob/master/community-membership.md#approver)

More about code review using [OWNERS](https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md#code-review-using-owners-files)
