# hack/chains

Some bash scripts to help configure and demonstrate [Tekton
Chains](https://github.com/tektoncd/chains) in an infra-deployments managed
cluster.

## To set up an environment from scratch

### Bootstrap

For CRC users:

    crc delete
    crc start
    `crc console --credentials | tail -1 | cut -d\' -f2`

Then (assuming you're in this branch on your own fork):

    hack/bootstrap-cluster.sh preview

A "go make some coffee" one liner:

    cd $(git rev-parse --show-toplevel); crc delete; crc start; `crc console --credentials | tail -1 | cut -d\' -f2`; hack/bootstrap-cluster.sh preview

Wait a while until you see mostly healthy/synced at [Argo CD](https://openshift-gitops-server-openshift-gitops.apps-crc.testing/applications).

(I have degraded 'has' and 'spi' related to GitHub auth and I don't think it
matters for the basic pipeline functionality needed to do builds and
demonstrate chains.)

### Create a key-pair signing secret for chains

    cd hack/chains
    ./create-signing-secret.sh

### Apply some CA cert related hacks to make SSL work for the internal registry

    ./trust-local-cert.sh

## Demos

Note: All these demos have been tested in local CRC cluster. They haven't yet
been confirmed working in a cluster-bot cluster.

### Kaniko build demo

- Trigger a taskrun that builds an image using kaniko and pushes it to
    the cluster's internal registry
- Verify the image in the internal registry using cosign
- Verify the image's attestation in the internal registry using cosign
- Using rekor-cli verify the rekor record of the taskrun (and/or image?)

Because we're using the cluster registry, the trust-local-cert and
setup-controller-certs are needed.

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

    ./gitops-sync.sh off
    ./config.sh quay
    kubectl create -f your-downloaded-quay-secret.yml
    ./pipeline-quay-demo.sh quay.io/your-user/your-repo your-quay-secret-name
