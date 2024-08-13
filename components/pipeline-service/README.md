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

- For development overlay: 

Edit the yaml files under the development subdirectory as needed.  Run the 'kustomize build' command for the 
development overlay as a sanity check, and then git commit to your branch to include in your pull request.

- For stage overlay:

Same process with respect to editing files, though under the 'stage' subdirectory this time,  and running 'kustomize build',
but there is a separate step.  Specific 'deploy.yaml' files are created for each of the actual Konflux clusters.

To update them with your yaml changes:
```shell
# cd to your infra-deployment git clone base directory
$ cd components/pipeline-service/staging
$ ../../../hack/generate-deploy-config.sh 
$ cd -
# to see if your updates made it to the deploy.yaml files
$ git status
```

- For production overlay:

Same as 'stage' overlay, though you'll deal with the yaml files under the 'production' subdirectory, and the 
use of 'generate-deploy-config.sh' similarily is tweaked to update the 'production' overlay.

```shell
# cd to your infra-deployment git clone base directory
$ cd components/pipeline-service/production
$ ../../../hack/generate-deploy-config.sh 
$ cd -
# to see if your updates made it to the deploy.yaml files
$ git status
```