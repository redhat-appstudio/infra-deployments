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

## How to update an existing ClusterPolicy

In this section you find some guidelines on how to update your ClusterPolicy based on the lessons learned using Kyverno.
If you think your case is different from the ones described here, please reach-out to the Konflux Infrastructure Team or to any of the [OWNERS](./OWNERS).

Updating a ClusterPolicy can be tricky and some care needs to be put in this process.
Especially when a change to an _immutable_ field is involved or the ClusterPolicy is generating resources with the `synchronize` flag enabled.

If no _immutable_ field is changed, you can mostly just update the ClusterPolicy in-place.

Otherwise, if an _immutable_ field is changed, a Kyverno webhook will deny the update.
When ArgoCD is involved, as it will continuously retry to apply the change, it can happen that it is eventually accepted.
We saw that happening when the Kyverno Admission Controller got restarted while ArgoCD was retrying.
In this case, it's important to delete and then recreate the ClusterPolicy, giving to Kyverno the time to properly cleanup the ClusterPolicy.

To check if an _immutable_ field is changed, you can setup a Kind cluster with Kyverno on it, then install the `main`'s version of the ClusterPolicy and finally try to update it to the new version.
The [hack/chainsaw/chainsaw-prepare.sh](../../hack/chainsaw/chainsaw-prepare.sh) can help with the initial setup.

### Generate Rules

[Generate rules](https://release-1-15-0.kyverno.io/docs/policy-types/cluster-policy/generate/) have a `synchronize` field.
If it's set to `true`, deleting the ClusterPolicy will cause the [deletion of all the generated resources](https://release-1-15-0.kyverno.io/docs/policy-types/cluster-policy/generate/#data-source).

#### Deletion of generated resources expected

If this is expected, you can:
1. Delete the _old_ version.
    1. Wait for all the generated resources to be gone.
       Generated resources have labels pointing to Kyverno (`app.kubernetes.io/managed-by: kyverno`), the ClusterPolicy (`generate.kyverno.io/policy-name`), and the Rule (`generate.kyverno.io/rule-name`).
1. Create the _new_ version.
    1. Ensure the _new_ version was accepted by checking the ClusterPolicy's status.

#### Deletion of generated resources NOT acceptable

When this is not acceptable, you can set the `synchronize` field to `false` and `orphanDownstreamOnPolicyDelete` to `true` before deleting it.

1. Set `synchronize: false` and `orphanDownstreamOnPolicyDelete: true`
1. Delete the _old_ version.
    1. Ensure the _old_ version was cleaned up correctly.
1. Create the _new_ version.
    1. (Optional) Set `generateExisting: true`
    1. Ensure the _new_ version was accepted by checking the ClusterPolicy's status.
    1. If the Generate rule needs `synchronize: true`, you can set it back.

While ensuring the _old_ version was deleted and before the _new_ version is accepted, we can miss some events and skip the generation of resources.
If this is not acceptable, it's possible to remediate by (temporarily) setting the `generateExisting` flag to `true`.
This will cause the Kyverno background controller to look at all the existing matches in the cluster and generate the resource for it.

### Validate Rules

When updating a Validating ClusterPolicy, it's usually not desired to have intervals in which no Validating logic is enforced.
To avoid this it's preferred -when it's possible- to install the _new_ version of the ClusterPolicy alongside the _old_ one.

In this case we can:
1. Create the _new_ version of a ClusterPolicy with a different name.
    1. Ensure the _new_ version was accepted by checking the ClusterPolicy's status.
1. Delete the _old_ version.
    1. Ensure the _old_ version was removed.

Doing this in separate steps allows you to check the _new_ version was accepted before removing the _old_ validation logic.

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
