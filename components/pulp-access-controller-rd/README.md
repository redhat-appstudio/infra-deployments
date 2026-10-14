# Pulp Access Controller

The `pulp-access-controller` is a Kubernetes operator that automates the creation of secrets for accessing Red Hat Pulp services.

## Overview

Similar to `image-controller` which provisions Quay repos and creates secrets for user pipelines, the Pulp Access Controller provisions Pulp domains and provides secrets for user pipelines to access those domains.

## Features

- **Automated Secret Creation**: Creates `pulp-access` secrets containing credentials and configuration files
- **Domain Management**: Automatically creates Pulp domains with the naming convention `konflux-<namespace>`
- **Certificate-Based Authentication**: Uses mTLS with client certificates for all Pulp API operations
- **Quay Integration**: Optional OCI storage backend configuration with Quay.io
- **Status Tracking**: Reports completion status via Kubernetes status conditions

## Usage

Users create a `PulpAccessRequest` custom resource in their namespace, referencing a secret that contains their credentials:

```yaml
apiVersion: pulp.konflux-ci.dev/v1alpha1
kind: PulpAccessRequest
metadata:
  name: my-pulp-access
  namespace: my-namespace
spec:
  credentialsSecretName: pulp-credentials  # Reference to existing credentials secret
  use_quay_backend: false  # Optional: Enable Quay.io OCI backend
```

The referenced credentials secret should contain TLS certificates:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pulp-credentials
  namespace: my-namespace
type: Opaque
stringData:
  tls.crt: |
    -----BEGIN CERTIFICATE-----
    ... your certificate content ...
    -----END CERTIFICATE-----
  tls.key: |
    -----BEGIN PRIVATE KEY-----
    ... your private key content ...
    -----END PRIVATE KEY-----
```

The controller will then:
1. Read the TLS certificate and key from the referenced secret
2. Create a Pulp domain named `konflux-<namespace>` (e.g., `konflux-my-namespace`) via mTLS API
3. Generate a `pulp-access` secret with all necessary configuration files
4. Optionally configure Quay.io as OCI storage backend (if `use_quay_backend: true`)
5. Update the status to indicate completion

## Status Monitoring

Check if the request completed successfully:

```bash
kubectl get pulpaccessrequest my-pulp-access -o yaml
```

The status will show a `Ready` condition with `True` when processing is complete.

For detailed documentation, see the [main README](../../../pulp-access-controller/README.md).
