---
title: Setting up observability stack
---

A script exists to create the secrets and datasource used to connect Grafana with the Prometheus instance, required for observability ([hack/setup-monitoring.sh](hack/setup-monitoring.sh)).  
This script requires few things:
* [Github oauth](https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps) tokens for authentication of the components
* [Github Cookie secret](https://oauth2-proxy.github.io/oauth2-proxy/docs/configuration/overview)
* [oc](https://docs.openshift.com/container-platform/4.11/cli_reference/openshift_cli/getting-started-cli.html) binary installed and configured to have admin access to the cluster

for running the `hack/setup-monitoring.sh` script
1. Copy `hack/setup-monitoring.sh` to `hack/monitoring.sh`

2. Update the values for the variables in `hack/monitoring.sh` from github oauth

3. ```$ ./hack/monitoring.sh```
