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

    crc delete; crc start; `crc console --credentials | tail -1 | cut -d\' -f2`; hack/bootstrap-cluster.sh preview

Wait a while until you see mostly healthy/synced at [Argo CD](https://openshift-gitops-server-openshift-gitops.apps-crc.testing/applications).

(I have degraded 'has' and 'spi' related to GitHub auth and I don't think it
matters for the basic pipeline functionality needed to do builds and
demonstrate chains.)

### Create a key-pair signing secret for chains

    cd hack/chains
    ./create-signing-secret.sh

### Apply some CA cert related hacks to make SSL work

    ./trust-local-cert.sh
    ./setup-controller-certs.sh

## Demos

### Kaniko build demo

Currently this works well in a local CRC cluster, but has some problems
running on a cluster-bot cluster in AWS.

    ./kaniko-demo-build.sh
    ./kaniko-demo-cosign.sh
    ./kaniko-demo-rekor.sh

### S2I pipeline build demo

Todo

### Buildah pipeline build demo

Todo
