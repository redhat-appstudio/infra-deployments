---
title: Toolchain (Sandbox) Operators
---

## Installation

This part is automated if you use `--toolchain` parameter of `hack/bootstrap-cluster.sh`

There are two scripts which you can use:

- `./hack/sandbox-development-mode.sh` for development mode
- `./hack/sandbox-e2e-mode.sh` for E2E mode

Both of the scripts will:

1. Automatically reduce the resources.requests.cpu values in argocd/openshift-gitops resource.

2. Install & configure the Toolchain (Sandbox) operators in the corresponding mode.

3. Print:
    - The landing-page URL that you can use for signing-up for the Sandbox environment that is running in your cluster.
    - Proxy URL.

### SSO

This part is automated if you use `--toolchain --keycloak` parameters of `hack/bootstrap-cluster.sh`. These parameters install toolchain operators (`./hack/sandbox-development-mode.sh`) and configure them to use keycloak, which is automatically deployed as part of `development` overlay.

In development mode, the Toolchain Operators are configured to use Keycloak instance that is internally used by the Sandbox team. If you want to reconfigure it to use your own Keycloak instance, you need to add a few parameters to `ToolchainConfig` resource in `toolchain-host-operator` namespace.
This is an example of the needed parameters and their values:

```yaml
spec:
  host:
    registrationService:
      auth:
        authClientConfigRaw: '{
                  "realm": "sandbox-dev",
                  "auth-server-url": "https://sso.devsandbox.dev/auth",
                  "ssl-required": "none",
                  "resource": "sandbox-public",
                  "clientId": "sandbox-public",
                  "public-client": true
                }'
        authClientLibraryURL: https://sso.devsandbox.dev/auth/js/keycloak.js
        authClientPublicKeysURL: https://sso.devsandbox.dev/auth/realms/sandbox-dev/protocol/openid-connect/certs
      registrationServiceURL: <The landing page URL>
```