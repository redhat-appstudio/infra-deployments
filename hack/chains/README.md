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
the 'build' application by setting DEPLOY_ONLY. (The DEPLOY_ONLY var takes a
comma separated list, so you can add other applications if required.)

    DEPLOY_ONLY=build

To workaround a Gitops problem causing the 'pvc-cleaner' component to never
complete syncing when installing just the 'build' application, create the
'application-service' project in advance. (This may become unnecessary once
[#214](https://github.com/redhat-appstudio/infra-deployments/pull/241) is
completed.)

    oc new-project application-service

Then, assuming you're working in a branch in your own fork
of this repo:

    hack/bootstrap-cluster.sh preview

A "go make some coffee" one liner for CRC:

    cd $(git rev-parse --show-toplevel); crc delete; crc start; `crc console --credentials | tail -1 | cut -d\' -f2`; oc new-project application-service; DEPLOY_ONLY=build,enterprise-contract hack/bootstrap-cluster.sh preview

Wait until you see healthy/synced at [Argo CD](https://openshift-gitops-server-openshift-gitops.apps-crc.testing/applications).

If you installed all the applications, there are other steps required to get
them all to go green. See [this
guide](https://coreos.slack.com/files/T027F3GAJ/F036QJ81LLU) for details.

### **Deprecated** - (Optional) Deploy local cluster rekor server

    cd hack/chains
    ./deploy-local-rekor.sh

To enable the use of a locally deployed rekor instance:

    ./config.sh rekor-local

To return to using the default, public rekor instance:

   ./config.sh rekor-default

### Local cluster rekor server

After running the `bootstrap-cluster.sh` script, a rekor application is deployed to the local cluster. This rekor instance has an internal hostname and ingress configured ( `rekor.enterprise-contract-service.svc` ) which can be used in conjunction with chains. This hostname will not be accessible by default from your local development machine.

Execute the following commands to update the chains configmap to allow chains to utilize this internal hostname instead of the public Sigstore instance:

    # turn off gitops syncing (otherwise, the configmap is overwritten to ensure proper sync state)
    $ ./hack/chains/gitops-sync.sh off

    # patch the configmap
    $ kubectl patch configmap chains-config -n tekton-chains \
    -p='{"data":{"transparency.url": "http://rekor-server.rekor.svc.cluster.local"}}'

### External access for local cluster rekor server

By default, executing the `boostrap-cluster.sh` script with the `preview` parameter after executing the `boostrap-cluster.sh` script will do the following:
* Reconfigure the local cluster rekor server with a hostname that is externally accessible in the format of `rekor.<cluster_domain>`
* Patch the chains config to use this hostname in the format of `https://rekor.<cluster_domain>`
* Restart the chains controller pod

At this point you may access the local cluster rekor server via API or with the `rekor-cli`.

To access via `rekor-cli`, ensure that you provide the `--rekor_server` flag such as in the example below:

    $ rekor-cli --rekor_server https://rekor.<cluster_domain> loginfo

### YAML Validation

To validate any modified or untracked YAML files, execute the `check-yaml.sh` script in the `hack/chains` directory. This script uses `yamllint` and will notify you if it isn't installed on your system. [This](https://yamllint.readthedocs.io/en/stable/quickstart.html#installing-yamllint) URL is provided for installation details.

### Trust the cluster's SSL cert (optional)

Some of the demos expect that cosign running on your workstation can use SSL
to connect to the cluster's internal registry. This script should will help
make that work.

    ./trust-local-cert.sh


Chains Demos
------------

### Required prerequisites

You must have the following prerequisites installed to run any of the demos in
this section.
- [tkn](https://github.com/tektoncd/cli)
- [docker](https://docs.docker.com/get-docker/)
- [skopeo](https://github.com/containers/skopeo)
- [rekor-cli](https://docs.sigstore.dev/rekor/installation/)
- [cosign](https://docs.sigstore.dev/cosign/installation/)

The script `./install-demo-pre-req.sh` in the `hack/chains/setup` directory will
install these prerequisites for you, if they're not already installed.

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

This uses the default configuration so there's no need to change the config if
your Argo CD is in sync.

    ./gitops-sync.sh off
    ./config.sh default
    kubectl create -f your-downloaded-quay-secret.yml
    ./pipeline-quay-demo.sh quay.io/your-user/your-repo your-quay-secret-name


Enterprise Contract demos
-------------------------

### Release pipeline demo

This demo will create and run an example release pipeline including the
Enterprise Contract task.

See also the [task
definition](https://github.com/redhat-appstudio/build-definitions/blob/main/tasks/verify-enterprise-contract-v2.yaml)
and [related scripts](https://github.com/redhat-appstudio/build-definitions/tree/main/appstudio-utils/util-scripts)
in the [build-definitions](https://github.com/redhat-appstudio/build-definitions) repo.

    ./release-pipeline-with-ec-demo.sh <src-image-ref> <dst-image-ref>

See the comments in that script for more details on how to use it.

### End to end demo

This will run a build pipeline, then run a QE pipeline, and then use the above
release pipeline demo to verify that build against the Enterprise Contract.

    ./end-to-end-demo.sh

There is a [video showing this in action](https://drive.google.com/file/d/1DEqAVhqNhu2L1t8w3_fkxAY860YOnx6M/view?usp=sharing)
(8 minutes, Apr 28, 2022. Red Hat internal only).
