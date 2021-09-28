
# AppStudio Infrastructure Deployments

This repository is an initial bootstrap of an Argo-CD-based deployment of AppStudio components to a cluster, plus a script to bootstrap Argo CD onto that cluster (to drive these Argo-CD-based deployments).

This repository is structured as a GitOps monorepo (eg the repository contains the K8s resources for *multiple* applications), using [Kustomize](https://kustomize.io/).

The contents of this repository are not owned by any single individual, and should instead be collectively managed and maintained through PRs by individual teams.

## How to add your own component

You may use the `gitops` component as an example for how to add your own component. Here `gitops` refers to the GitOps service team's K8s resources.

These are the steps to add your own component:
1) Create a new directory for your team's components, under `components/(team-name)`.
2) Add a `kustomization.yaml` file under that directory, which points to the individual K8s YAML resources you wish to deploy.
    - You may also structure your deployment into directories and files. See the Kustomize documentation for more information, and/or examples below.
    - See `components/gitops/backend` for an example of this.
3) Create an Argo CD `Application` resource in `argo-cd-apps/base/(team-name).yaml`). 
    - See `gitops.yaml` for a template of how this should look.
    - The `.spec.source.path` value should point to the directory you created in previous step.
    - The `.spec.destination.namespace` should match the target namespace you wish to deploy your resources to.
    - The `metadata.namespace` must be `cluster-argocd` (do not change it).
    - The `metadata.name` should correspond to your `(team-name)`
4) Add a reference to your new `(team-name).yaml` file, to `argo-cd-apps/base/kustomization.yaml` (the reference to your YAML file should be in the `resources:` list field).
5) Run `kustomize build (repo root)/argo-cd-apps/overlays/staging` and ensure it passes, and displays your new Argo CD Application CR.
6) Open a PR for all of the above.

More examples of using Kustomize to drive deployments with GitOps can be [found here](https://github.com/redhat-cop/gitops-catalog).

## Maintaining your components

Simply update the files under `components/(team-name)`, and open a PR with the changes. 

**TIP**: For development purposes, you can use `kustomize build .` to output the K8s resources that are being generated for your folder.

## Bootstrapping a cluster

The pre-requisities are:
- You must have `kubectl` and `kustomize` installed. 
- You must have `kubectl` pointing to an existing K8s cluster, that you wish to deploy to.

Steps:
1) Run `hack/bootstrap-cluster.sh` which will bootstrap Argo CD and setup the Argo CD `Application` CRs for each component.
2) Retrieve the Argo CD Web UI URL using `kubectl get routes`.
2) Retrieve the `admin` password using `oc get secret argocd-initial-admin-secret -n cluster-argocd -o jsonpath='{.data.password}' | base64 -d`. Username is `admin`
3) View the Argo CD UI to see the status of deployments.

## FAQ

Other questions? Ask on `#wg-developer-appstudio`.

### Q: How do I deliver K8s resources in stages? For example, installing a CRD first, then installing the CR (for that CRD).

As long as your resources are declaratively defined, they will eventually be reconciled with the cluster (it just may take Argo CD a few retries). For example, the CRs might fail before the CRDs are applied, but on retry the CRDs will now exist (as they were applied during the previous retry). So now those CRs can progress.

However, for finer-grained control use Argo CD [Sync waves])(https://argoproj.github.io/argo-cd/user-guide/sync-waves/). ([Example](https://github.com/argoproj/argocd-example-apps/tree/master/sync-waves)).


### Q: What if I want my service's K8s resources in a separate Git repository? (ie not this one)

Ultimately, as a team, we should decide on a resource deployment strategy going forward, however, it is easiest to coordinate deployments across a single Git repository (such as this one), rather than multiple independent repositories.

If one or more services want to split off their K8s resource into independent repositories, they would modify the `Application` for their service (created in step 3, above) to point to their new repository.


