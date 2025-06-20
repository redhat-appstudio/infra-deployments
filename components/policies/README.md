# Policies component

This component manages all the Kyverno Policies required from our Konflux instances.

## Structure

For each of the following environment there is a dedicated a folder.
* development
* staging
* production

### Development

Due to the simple nature of this environment, the folder is organized as a flat folder.
Policies are grouped by components that require and manage them.

### Staging and Production

The `staging` and `production` folders contain a folder for each cluster in the environment, a folder named `base`, and a folder named `policies`.

The `base` folder contains Policies required by every cluster, grouped by component.
The `policies` folder contains Policies required only by some clusters, grouped by component.

## How to add a new ClusterPolicy

1. If the Policy is required by all clusters, put it the appropriate component in the `base` folder.
Otherwise, use the `policies` folder.
1. If the component folder doesn't exist, initialize it.
    1. Add a OWNERS file with the maintainers of the new Policy in the `reviewers`.
    1. Don't add any approver in the `approvers` section.
1. Update the appropriate `kustomization.yaml` files accordingly.

## Best practices for writing Policies

In this section you find some directions, they are not complete and will be updated in the future.

### Avoid broad matches

Matching too many resources can generate an high load on the cluster.
Scope your ClusterPolicy to only match the strictly required resources.

### Try not to use `generateExisting`, `synchronize`, and `updateExistingOnPolicyUpdate`

The `generateExisting`, `synchronize`, and `updateExistingOnPolicyUpdate` can have a high impact on the cluster.
Use them carefully and only when required. 

If you need them and you need to match a Resource which is 

### Leverage caches

Wherever usefull, use the [GlobalContext](https://release-1-13-0.kyverno.io/docs/writing-policies/external-data-sources/#global-context).
