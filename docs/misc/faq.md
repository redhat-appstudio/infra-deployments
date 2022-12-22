---
title: FAQ
---

Other questions? Ask on `#wg-developer-stonesoup`.

### Q: How do I deliver K8s resources in stages? For example, installing a Custom Resource Definition (CRD) first, then installing the Custom Resource (CR) for that CRD.

As long as your resources are declaratively defined, they will eventually be reconciled with the cluster (it just may take Argo CD a few retries). For example, the CRs might fail before the CRDs are applied, but on retry the CRDs will now exist (as they were applied during the previous retry). So now those CRs can progress.

_However_, this is not true if you are installing an Operator (e.g. Tekton) via OLM `Subscription`, and then using an operand of that operator (e.g. `Pipeline` CRs), at the same time. In this case, you will need to add the `argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true` annotation to the operands (or add it to the parent `kustomization.yaml`).

See the FAQ question [the server could not find the requested resource_' below for details](#q-when-argo-cd-attempts-to-synchronize-my-argo-cd-application-i-am-seeing-the-server-could-not-find-the-requested-resource-sync-error-on-my-custom-resources-how-can-i-fix-this)

For finer-grained control of resource apply ordering, use Argo CD [Sync waves](https://argoproj.github.io/argo-cd/user-guide/sync-waves/) (Here is an [example](https://github.com/argoproj/argocd-example-apps/tree/master/sync-waves)).

### Q: What if I want my service's K8s resources in a separate Git repository? (i.e. not this one)

Having a monorepo for storing the ArgoCD application definition is  the easiest way to gate changes that affects
the entire service. This is done by running the E2E tests for verifying PRs.

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