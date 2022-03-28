hack/chains
===========

Some bash scripts to help configure and demonstrate [Tekton
Chains](https://github.com/tektoncd/chains) in an infra-deployments managed
cluster.


To set up an environment from scratch
-------------------------------------

### Bootstrap

For CRC:

    crc delete
    crc start
    `crc console --credentials | tail -1 | cut -d\' -f2`

For cluster-bot (in slack):

    launch 4.9.15 $PLATFORM
    # Use one of the following tested platforms: ovirt, openstack, azure, aws
    # ...and follow the instructions to authenticate once it's ready

The full Gitops/ArgoCD install includes applications that are not needed if
you're just testing build pipelines and tasks with tekton chains. So you can
save some time and reduce cluster resourcing requirements by installing only
the 'build' application.

    DEPLOY_ONLY=build

(The DEPLOY_ONLY var takes a comma separated list, so you can add other
applications if required.)

Then (assuming you're working in a branch in your own fork
of this repo):

    hack/bootstrap-cluster.sh preview

A "go make some coffee" one liner for CRC:

    cd $(git rev-parse --show-toplevel); crc delete; crc start; `crc console --credentials | tail -1 | cut -d\' -f2`; DEPLOY_ONLY=build hack/bootstrap-cluster.sh preview

Wait until you see healthy/synced at [Argo CD](https://openshift-gitops-server-openshift-gitops.apps-crc.testing/applications).

If you installed all the applications, there are other steps required to get
them all to go green. See [this
guide](https://coreos.slack.com/files/T027F3GAJ/F036QJ81LLU) for details.


### (Optional) Deploy local cluster rekor server

    cd hack/chains
    ./deploy-local-rekor.sh

To enable the use of a locally deployed rekor instance:

    ./config.sh rekor-local

To return to using the default, public rekor instance:

   ./config.sh rekor-default


### Trust the cluster's SSL cert (optional)

Some of the demos expect that cosign running on your workstation can use SSL
to connect to the cluster's internal registry. This script should will help
make that work.

    ./trust-local-cert.sh


Demos
-----

### Kaniko build demo

- Trigger a taskrun that builds an image using kaniko and pushes it to
    the cluster's internal registry
- Verify the image in the internal registry using cosign
- Verify the image's attestation in the internal registry using cosign
- Use rekor-cli to verify the rekor record of the taskrun

Because we're using the cluster's internal registry, the demo will work better
if your workstation trusts the cluster CA cert. This can be achieved by
running `trust-local-cert.sh`.

This uses the default configuration so there's no need to change the config if
your Argo CD is in sync.

    ./gitops-sync.sh off
    ./config.sh default
    ./kaniko-demo.sh


### Simple demo

- Trigger a taskrun
- Verify the taskrun using cosign verify-blob

The taskrun does not build an image but it does create a fake digest that
chains seems to interact with. I don't understand what's going on with that
digest, but the focus here is the taskrun verification using tekton storage.

Note that it uses the deprecated tekon format instead of in-toto, like the
official basic getting started guide.

    ./gitops-sync.sh off
    ./config.sh simple
    ./simple-demo.sh

You can also run this with `in-toto` taskrun format.

    ./config.sh 'artifacts.taskrun.format: in-toto'
    ./simple-demo.sh

And you can run it with the transparency log enabled and look at the log
entry.

    ./config.sh rekor-on
    ./simple-demo.sh
    ./rekor-verify-taskrun.sh


### Pipeline S2I build and push to quay.io demo

- Configure a clustertask that can build an nodejs image using S2I
- Configure a pvc and a pipeline with that task
- Trigger a pipeline run which builds an image and pushes it to the
    (probably) quay.io registry specified
- Use cosign to verify the image in the registry
- Use rekor-cli to verify the taskrun that built the image

For this demo you need to provide your own registry image url and a k8s
secret. To do that, go to 'User and Robot Permissions' to create a read/write
robot account permission for your repo in quay.io, and then download the
kubernetes secret for the bot.

Since we're not using the internal registry, this demo should work without
running the trust-local-cert.sh and setup-controller-certs.sh scripts.

Note that quay.io doesn't currently support storing attestations, but see
[PROJQUAY-3386](https://issues.redhat.com/browse/PROJQUAY-3386) which aims to
change that.

    ./gitops-sync.sh off
    ./config.sh quay
    kubectl create -f your-downloaded-quay-secret.yml
    ./pipeline-quay-demo.sh quay.io/your-user/your-repo your-quay-secret-name
