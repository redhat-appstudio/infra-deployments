# Configuring OpenShift Pipelines (the version of Tekton installed by the OpenShift Pipelines Operator) on Konflux

**Migration Status**: In-Progress

As of August, 2024, Konflux no longer uses the [deprecated 'Pipeline Service' repository](https://github.com/openshift-pipelines/pipeline-service/)
as a base of the Tekton related configuration deployed on the various Konflux clusters.

All the different Kubernetes base object and CRD instances reside in the various subdirectories here for each of
the GitOPS overlays employed for Konflux in this repository.

While the OpenShift Pipelines team will continue to provide assistance to questions from the Konflux community, 
via slack messages to #forum-ocp-pipelines (public facing channel) or #team-ocp-pipeline or mentions in pull
requests (@openshift-pipelines/pipelines or individual developer github handles), the expectation is that if 
a Konflux development team needs an update to any Tekton related configuration, they will initiate the change
and review with the OpenShift Pipelines team as needed.

Configuration documentation:
[Upstream configuration reference](https://tekton.dev/docs/pipelines/additional-configs/)

[Downstream documentation for OpenShift Pipelines](https://docs.openshift.com/pipelines/1.15/about/understanding-openshift-pipelines.html)

## What gets deployed

The `CatalogSource` and `Subscription` resources install the OpenShift Pipelines Operator from a pinned index image rather than the default Red Hat catalog, so operator upgrades happen on a schedule controlled by this repo instead of automatically.

### OpenShift Pipelines

The resources here, which run in the `openshift-pipelines` namespace, help manage RBAC permissions and monitor  the Tekton Chains controller. The namespace is created in this component and the `konflux-pipeline-service`/`konflux-sre` groups are granted `edit` access.

Tekton Chains' signing key and public key are synced in from Vault (`appsre-stonesoup-vault` ClusterSecretStore); the public key is readable by all authenticated users. A Tekton Chains `Service` and `ServiceMonitor` are deployed to monitor and expose the Tekton Chains controller. The ArgoCD application controller is granted permission to manage the `ServiceMonitor`, `TektonConfig`, `SecurityContextConstraints`, and `batch/jobs` resources, since the operator doesn't own those directly.

### Tekton Logging

The resources here, which mainly exist in the `tekton-logging` namespace, sync the credentials needed to authenticate to the AWS Tekton Results S3 bucket, expose metrics regarding `PipelineRun`s, and provide the `konflux-pipeline-service` / `konflux-sre` groups cluster-scoped access to debug resources and issues related to  cluster upgrades and cluster health. 

The other resources for exporting pipeline metrics live in the `openshift-pipelines` namespace. A `Deployment`, `Service`, and `ServiceMonitor` regarding exporting pipeline metrics are deployed and a `ServiceAccount` specifically for exporting these metrics is granted viewing permissions for many resources in addition with the ability to patch `Pipeline`/ `TaskRun`s. 

### Tekton Results

The resources here, which run in the `tekton-results` namespace, deploy the main `Deployments` and `Services` related to Tekton Results in addition to other resources required to connect Tekton Results to external sources.  

The API, API for Watcher, and Watcher `Deployments`/`Services` are ran here in addition to a scaled-to-zero retention-policy agent that prunes results after certain configured period of time. An external `Route` is configured for the API, and `ServiceMonitor`s exist for both the API and Watcher services. `ExternalSecret`s pull DB and S3 credentials from Vault and aggregated `tekton-results-admin`/`-readonly`/`-readwrite` ClusterRoles control who can read or manage results.

### Testing Resources

Deploys a dedicated namespace and RBAC used by Tekton Results end-to-end/smoke tests.

| Manifest | Kind | Purpose |
| --- | --- | --- |
| `testing-ns.yaml` | `Namespace` | Creates `plnsvc-tests` |
| `testing-rbac.yaml` | `ServiceAccount` / `ClusterRoleBinding` / `RoleBinding` | Grants the 'tekton-results-tests' SA permissions in the 'tekton-results-readonly' `ClusterRole` and the 'konflux-pipeline-service' group permissions in the 'tekton-results-admin' `ClusterRole` |

## Promotion Tactics

This component is a special case in terms of using the automatic ring deployments workflow. Unlike most other components, there are no images listed in the ring or cluster Kustomize files. Image-based changes can still be automatically detected and promoted when a new image is available from the associated Quay repository, but this configuration allows image-based changes to be paired with manifest-based changes in the same [Freight](https://docs.kargo.io/user-guide/core-concepts#freight) by making changes to each resource's image reference in the `components/pipeline-service-rd/base/` directory.

Both image-based and manifest-based changes must follow the [Manifest-based Promotion Workflows](https://gitlab.cee.redhat.com/konflux/docs/sop/-/blob/main/infra/ring-deployments/component-promotions.md?ref_type=heads#manifest-based-promotion), with the exception that Kargo can automatically create the first PR in the `components/pipeline-service-rd/base/` directory if it detects a new image.

If an image-based change also requires the rollout of a new ring/cluster-level patch, a component owner must checkout the branch that Kargo creates and push the patch for that specific ring/cluster(s) before the auto-generated PR is approved and merged.

The following components reference images:

- 'custom-operators' `CatalogSource`
- 'pipeline-metrics-exporter' `Deployment` (`tekton-logging/`)
- 'tekton-results-retention-policy-agent' `Deployment` (`tekton-results/deployments/`)
- 'tekton-results-api' `Deployment` (`tekton-results/deployments/`)
- 'tekton-results-api-for-watcher' `Deployment` (`tekton-results/deployments/`)
- 'tekton-results-watcher' `Deployment` (`tekton-results/deployments/`)
