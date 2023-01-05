---
title: Setting up observability stack
---

Script to set up observability stack (i.e. prometheus and grafana)
- [hack/setup_observability.sh](hack/setup_observability.sh)
This script requires few things
* [Github oauth](https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps) tokens for authentication of the components
* [Github Cookie secret](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview)
* [oc](https://docs.openshift.com/container-platform/4.11/cli_reference/openshift_cli/getting-started-cli.html) binary installed and configured to have admin access to the cluster

for running the `hack/setup_observability.sh` script
1. Copy `hack/monitoring-template.env` to `hack/monitoring.env`

2. Update the values for the variables in `hack/monitoring.env` from github oauth

3. ```$ ./hack/setup_observability.sh```