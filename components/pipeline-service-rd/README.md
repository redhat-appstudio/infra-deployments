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

- Ring 0 (development) overlay:

```shell
# cd to your infra-deployment git clone base directory
$ kustomize build ./components/pipeline-service/rings/ring-0/base/
```

- Ring 1 (staging) overlay:

```shell
# cd to your infra-deployment git clone base directory
$ kustomize build ./components/pipeline-service/rings/ring-1/base
```

- Production overlays are present in rings 2 - 4
