---
title: Prerequisites
---

## Required prerequisites

The prerequisites are:

- You must have the following tools install:
  - `kubectl` - that matches your cluster version.
  - `oc` - that matches your cluster version.
  - `jq` >= `1.6`
  - [`yq`](https://github.com/mikefarah/yq) >= `v4.23.1`
  - `openssl` >= v3.0.2
- The script `./hack/setup/install-pre-req.sh` will install these prerequisites for you, if they're not already installed.
- You must have `kubectl` and `oc` pointing to an existing OpenShift cluster, that you wish to deploy to.

**Note - Mac OS**

If you're using Mac OS, make sure you are using GNU version of `sed` (`sed --version` -> **GNU sed 4.8**), openssl `openssl version` >= v3.0.2 and bash (`bash --version` >= **GNU bash, version 5.2**), because some of the configuration scripts in this repository depend on them.

You can install correct versions of these tools with:
```bash
brew install openssl@3 gnu-sed bash
```
Then make sure the $PATH is updated to point to those tools' binaries (by updating your .bashrc/.zshrc file):
```bash
export PATH="/usr/local/opt/openssl@3/bin:$PATH"
export PATH="/usr/local/opt/gnu-sed/libexec/gnubin:$PATH"
export PATH="/usr/local/bin/bash:${PATH}"
```
After opening a new terminal window you should be using correct versions of these tools by default.

## Bootstrapping a cluster

Any Openshift cluster with a configured default Storage Class can be used for the deployment.
If you don't have a cluster, you can try the methods below for allocating one.

### OpenShift Local Setup

If you don't already have a test OpenShift cluster available, OpenShift Local is a popular option. It runs a small OpenShift cluster in a single VM on your local workstation.

1. Create or log in using your free Red Hat account, and [install OpenShift Local](https://console.redhat.com/openshift/create/local).

2. Make sure you have the latest version of CRC: `crc version`

3. Run `./hack/setup/prepare-crc.sh` to configure OpenShift Local with the recommended minimum memory (16 GiB) and CPUs (6) for StoneSoup. The script has optional parameters for customizing `memory` and `cpu` allowance. It also supports `force delete` of existing cluster. Run `./hack/setup/prepare-crc.sh --help` to see the options. The script will also enable cluster monitoring and log you in as the cluster administrator.

### QuickCluster storage setup for clusters

If you are using QuickCluster to provision your development cluster, you will need to setup storage prior to running the bootstrap script.

See [Configuring NFS storage provisioner on QuickCluster clusters](../../hack/quickcluster/README.html)
