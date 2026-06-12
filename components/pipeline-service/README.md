# Configuring OpenShift Pipelines (the version of Tekton installed by the OpenShift Pipelines Operator) on Konflux

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

## Kustomize verification commands

All commands in this section are relative to the infra-deployments repository root directory

- development overlay:

```shell
# cd to your infra-deployment git clone base directory
$ kustomize build ./components/pipeline-service/development/
```

- stage overlay:

```shell
# cd to your infra-deployment git clone base directory
$ kustomize build ./components/pipeline-service/staging/base
```

- production overlay

```shell
# cd to your infra-deployment git clone base directory
$ kustomize build ./components/pipeline-service/production/base
```

## Current protocol

The basic pattern followed is to update both the development and stage overlays with as new change.
The development overlay the 'appstudio-e2e-tests' CI job that leverage Konflux QE's [End to End tests](https://github.com/konflux-ci/e2e-tests) utilize.
Their slack channel for questions on the tests, known issues, help in debug, is #forum-konflux-qe.

The stage overlay deploys the Konflux clusters that post merge development testing can be done on, where those
clusters resemble as closely as possible the production clusters.

After vetting the change in stage for an amount of time appropriate for the complexity of your change, move onto
updating the production clusters.

## Update procedure

All commands in this section are relative to the infra-deployments repository root directory

- For development overlay: 

Edit the yaml files under the development subdirectory as needed.  Run the 'kustomize build' command for the 
development overlay as a sanity check, and then git commit to your branch to include in your pull request.

- For stage overlay:

Same process with respect to editing files, though under the 'stage' subdirectory this time,  and running 'kustomize build',
but there is a separate step.  Specific 'deploy.yaml' files are created for each of the actual Konflux clusters.

To update them with your yaml changes:
```shell
# cd to your infra-deployment git clone base directory
$ ./hack/generate-deploy-config.sh -c components/pipeline-service/staging
# to see if your updates made it to the deploy.yaml files
$ git status
```

- For production overlay:

By default production changes are done using a Ring deployment, for which details are included below.
If you're making changes which do not require a ring deployment, the change can be made
the same as the 'stage' overlay, though you'll deal with the yaml files under the 'production' subdirectory, and the
use of 'generate-deploy-config.sh' similarily is tweaked to update the 'production' overlay.

```shell
# cd to your infra-deployment git clone base directory
$ ./hack/generate-deploy-config.sh -c components/pipeline-service/production
# to see if your updates made it to the deploy.yaml files
$ git status
```

## Production Ring Deployments

Production clusters are organized into three rings for gradual rollout to minimize blast radius:

- **Ring 1**: Early adopter clusters (smallest blast radius)
- **Ring 2**: Mid-tier clusters
- **Ring 3**: Remaining production clusters

See `./components/pipeline-service/production/ring-mappings.yaml` for current cluster membership.

### Updating procedure for Ring Deployments

1. Apply the change as a patch in ring-1's kustomization file. For example:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources: []
patches:
  - target:
      kind: CatalogSource
      name: custom-operators
    patch: |-
      - op: replace
        path: /spec/image
        value: quay.io/openshift-pipeline/pipelines-index-4.18@sha256:d4d3f6210a384da6bfb11214a860e72e52d0a38b73037b8135c0571a6365d2a1
```

2. Sync ring configuration and regenerate deploy.yaml files
```bash
# Sync ring references to all clusters
./components/pipeline-service/production/sync-rings.sh

# Regenerate all deploy.yaml files
./hack/generate-deploy-config.sh -c components/pipeline-service/production
```

3. Deploy the changes

4. After the ring has been deployed and validated, repeat steps 1 through 3 for the next ring.

5. Once the change is ready to deploy on all clusters, apply the change to `components/pipeline-service/production/base/main-pipeline-service-configuration.yaml` and remove the patch from the rings' `kustomization.yaml` files.

6. Sync the ring configuration and regenerate `deploy.yaml` files. There should be no changes to the `deploy.yaml` files.


### Managing Ring Membership

To add a cluster to a ring or move it between rings, edit `production/ring-mappings.yaml`:

```yaml
ring-1:
  - kflux-fedora-01
  - stone-prod-p01
  - new-cluster-01  # Add new cluster here

ring-2:
  - stone-prod-p02
  - kflux-rhel-p01
```

Then run `./components/pipeline-service/production/sync-rings.sh` to apply the changes. The script will validate that all mapped clusters exist and all cluster directories are mapped to a ring.

### Common Ring Deployment Patterns

**Index update:**
```yaml
patches:
  - target:
      kind: CatalogSource
      name: custom-operators
    patch: |-
      - op: replace
        path: /spec/image
        value: quay.io/openshift-pipeline/pipelines-index-4.18@sha256:d4d3f6210a384da6bfb11214a860e72e52d0a38b73037b8135c0571a6365d2a1
```

**Image override:**
```yaml
patches:
  - target:
      kind: Subscription
      name: openshift-pipelines-operator
    patch: |-
      - op: add
        path: /spec/config/env
        value: {"name": "IMAGE_PAC_PAC_CLI", "value": "quay.io/openshift-pipeline/pipelines-pipelines-as-code-cli-rhel9@sha256:0fc0dd05236f14265e25cc770a7f9f3aeea2e27964a730dac39cf9f61d349bde"}
```

**Resource limit adjustment:**
```yaml
patches:
  - target:
      kind: TektonConfig
      name: config
    patch: |-
      - op: replace
        path: /spec/pipeline/options/statefulSets/tekton-pipelines-remote-resolvers/spec/template/spec/
containers/0/resources/limits/memory
        value: 12G
```
