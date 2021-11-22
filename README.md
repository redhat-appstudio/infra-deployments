
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
    - See `components/gitops/backend` for an example of this.
3) Create an Argo CD `Application` resource in `argo-cd-apps/base/(team-name).yaml`). 
    - See `gitops.yaml` for a template of how this should look.
    - The `.spec.source.path` value should point to the directory you created in previous step.
    - The `.spec.destination.namespace` should match the target namespace you wish to deploy your resources to.
    - The `.metadata.name` should correspond to your `(team-name)`
4) Add a reference to your new `(team-name).yaml` file, to `argo-cd-apps/base/kustomization.yaml` (the reference to your YAML file should be in the `resources:` list field).
5) Run `kustomize build (repo root)/argo-cd-apps/overlays/staging` and ensure it passes, and outputs your new Argo CD Application CR.
6) Open a PR for all of the above.

More examples of using Kustomize to drive deployments using GitOps can be [found here](https://github.com/redhat-cop/gitops-catalog).

## Maintaining your components

Simply update the files under `components/(team-name)`, and open a PR with the changes. 

**TIP**: For development purposes, you can use `kustomize build .` to output the K8s resources that are being generated for your folder.

## Bootstrapping a cluster

The prerequisites are:
- You must have `kubectl` and `kustomize` installed. 
- You must have `kubectl` pointing to an existing OpenShift cluster, that you wish to deploy to.

Steps:
1) Run `hack/bootstrap-cluster.sh` which will bootstrap Argo CD (using OpenShift GitOps) and setup the Argo CD `Application` CRs for each component.
2) Retrieve the Argo CD Web UI URL using `kubectl get routes`.
3) Log-in to the Web UI using your OpenShift credentials (using 'Login with OpenShift' button).
3) View the Argo CD UI to see the status of deployments.

## Development mode for your own clusters

Once you bootstrap a cluster above, the root ArgoCD Application and all of the component applications will each point to the upstream repository.

To enable development for a team or individual to test changes on your own cluster, you need to replace the references to `https://github.com/redhat-appstudio/infra-deployments.git` with references to your own fork.

There are a set of scripts that help with this, and minimize the changes needed in your forks.

There is a development configuration in `overlays/development` which includes a kustomize overlay that can redirect the default components individual repositorys to your fork. 

Steps:
1) in your forked repository run `hack/development-mode.sh` and this will update the root application on the cluster and all of the git repo references in `argo-cd-apps/overlays/development/repo-overlay.yaml`
2) you will need to push the updated references in `argo-cd-apps/overlays/development/repo-overlay.yaml` to your fork. Argo will now sync all the changes from your fork into the cluster
3) You can now make changes to your forked repository and test them via the gitops

4) To submit changes back to the upstream make sure you do not include the modified file `argo-cd-apps/overlays/development/repo-overlay.yaml`. 

One option to prevent accidentally including this modified file, you can run the script `hack/upstream-mode.sh` to reset everything including your cluster to `https://github.com/redhat-appstudio/infra-deployments.git` and match the upstream config. You can also checkout the current upstream 
` git fetch upstream; git checkout upstream/main -- argo-cd-apps/overlays/development/repo-overlay.yaml` to ensure you have the original file.  

After you commit your changes you can rerun to `hack/development-mode.sh` and reset your repo to point back to the fork. 

Note running these scripts in a clone repo will have no effect as the repo will remain `https://github.com/redhat-appstudio/infra-deployments.git`

 
## FAQ

Other questions? Ask on `#wg-developer-appstudio`.

### Q: How do I deliver K8s resources in stages? For example, installing a CRD first, then installing the CR (for that CRD).

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
