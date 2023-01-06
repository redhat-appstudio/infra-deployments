---
title: Service Provider Integration Deployment
---

## Post-bootstrap Service Provider Integration(SPI) Configuration

SPI requires Service Provider to have configured OAuth application so it can process the OAuth flow. Follow [Configuring Service Providers](https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/docs/ADMIN.md#configuring-service-providers) in SPI admin documentation.

> Authorization URL of staging server: `https://spi-oauth-spi-system.apps.appstudio-stage.x99m.p1.openshiftapps.com`  
Callback URL of staging server: `https://spi-oauth-spi-system.apps.appstudio-stage.x99m.p1.openshiftapps.com/oauth/callback`

> **NOTE:**  Following process is automated in `preview` mode

SPI components will fail to start right after the bootstrap as additional manual configuration is required before they are healthy.

1. Edit `./components/spi/base/config.yaml` [see SPI Configuration Documentation](https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/docs/ADMIN.md#configuration).

2. Create a `shared-configuration-file` Secret 

```bash
kubectl create secret generic shared-configuration-file --from-file=components/spi/base/config.yaml -n spi-system
```

3. In few moments, SPI pods should start

SPI Vault instance has to be manually initialized. There is a script to help with that:

1. Make sure that your cluster user has at least [permissions](../../components/authentication/spi-vault-admin.yaml).

2. Clone SPI operator repo 

```bash
git clone https://github.com/redhat-appstudio/service-provider-integration-operator && cd service-provider-integration-operator
```

3. run `vault-init.sh` script from repo root directory 

```bash
`./hack/vault-init.sh`
```