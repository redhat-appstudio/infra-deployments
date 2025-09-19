# Conforma Knative Service Component

A Kubernetes-native, event-driven service that automatically triggers enterprise contract verification for application snapshots using Tekton bundles.

## Overview

The Conforma Knative Service is a CloudEvents-based service that monitors for the creation of Snapshot resources and automatically triggers compliance verification workflows. It implements an event-driven architecture to bridge CloudEvents with Tekton pipelines, using bundle resolution to dynamically fetch verification tasks from container registries.

## Architecture

### Event-Driven Processing
- Listens for CloudEvents of type `dev.knative.apiserver.resource.add`
- Processes Snapshot resources from the `appstudio.redhat.com/v1alpha1` API
- Automatically creates Tekton TaskRuns for compliance verification

### Bundle Resolution
- Uses Tekton's bundle resolver to fetch tasks from `quay.io/conforma/tekton-task:latest`
- Eliminates the need for pre-installed tasks in the cluster
- Enables dynamic task updates without redeploying the service

## Deployment

### Prerequisites
- Kubernetes cluster with Tekton Pipelines installed
- Knative Serving installed and configured
- Knative Eventing installed and configured
- Access to the bundle registry (`quay.io/conforma/tekton-task:latest`)

### Staging Environment

The staging configuration includes:
- **Dedicated namespace**: `conforma`
- **VSA attestation support**: Enabled for staging compliance
- **Enhanced RBAC**: Additional permissions for attestation workflows
- **Optimized scaling**: Minimum 1 replica, maximum 3 replicas, target 10 concurrent requests
- **Debug logging**: Enabled for troubleshooting

### Deployment Structure

The component is organized as follows:
- **`base/`**: Core Kubernetes manifests
  - `configmap.yaml`: Configuration settings
  - `knative-service.yaml`: Knative Service definition
  - `rbac.yaml`: ServiceAccount, Role, and RoleBinding
  - `event-source.yaml`: ApiServerSource for Snapshot events
  - `trigger.yaml`: Event trigger configuration
  - `kustomization.yaml`: Base resource definitions
- **`staging/`**: Environment-specific overrides
  - `kustomization.yaml`: Staging configuration with enhanced features
  - `staging-patches.yaml`: Resource patches for staging environment

### Configuration

The service reads configuration from a ConfigMap named `taskrun-config`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: taskrun-config
  namespace: conforma
data:
  POLICY_CONFIGURATION: "github.com/enterprise-contract/config//slsa3"
  PUBLIC_KEY: |
    -----BEGIN PUBLIC KEY-----
    MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEZP/0htjhVt2y0ohjgtIIgICOtQtA
    naYJRuLprwIv6FDhZ5yFjYUEtsmoNcW7rx2KM6FOXGsCX3BNc7qhHELT+g==
    -----END PUBLIC KEY-----
  IGNORE_REKOR: "true"
```

### Environment Variables (Staging)

The staging environment includes additional environment variables:
- **`LOG_LEVEL`**: `debug` - Enhanced logging for troubleshooting
- **`ENVIRONMENT`**: `staging` - Environment identifier

### Resource Configuration (Staging)

The staging environment includes resource limits and requests:
- **Requests**: 128Mi memory, 100m CPU
- **Limits**: 512Mi memory, 500m CPU
- **Container Concurrency**: 10 concurrent requests per pod

## Features

- **Automated Compliance**: Triggers verification workflows without manual intervention
- **Multi-Namespace Support**: Handles snapshots across different namespaces
- **Configurable Policies**: Supports custom policy configurations and public keys
- **Cloud-Native**: Stateless, horizontally scalable, and Kubernetes-native
- **Bundle-Based**: Dynamic task resolution from container registries
- **VSA Attestation**: Full support for Verification Summary Attestation in staging

## Source Code

The service source code is maintained in: https://github.com/conforma/knative-service

## Container Images

- **Registry**: `quay.io/conforma/knative-service`
- **Tags**: `latest` (staging), versioned tags for releases

## Security

### RBAC Permissions

**Base Configuration:**
- **ConfigMaps**: get, list (for configuration)
- **TaskRuns**: create (for verification workflows)
- **Snapshots**: get, list, watch (for processing snapshot events)

**Staging Configuration (Enhanced):**
- **ConfigMaps**: get, list (for configuration)
- **TaskRuns**: create, get, list, watch (for verification workflows)
- **Snapshots**: get, list, watch (for processing snapshot events)
- **PipelineRuns**: get, list, watch (for VSA attestation support)

### VSA Attestation
- **Always Enabled**: VSA attestation is enabled by default in all environments
- Tekton Chains integration for automated signing (`chains.tekton.dev/signed: "true"`)
- Enhanced RBAC for attestation workflows in staging
- Configurable public keys for verification

## Monitoring

### Health Checks
The service includes health monitoring endpoints:
- **Health**: Service health status
- **Ready**: Service readiness for traffic

### Observability
- Structured logging with configurable levels
- CloudEvents processing with detailed logging
- Integration with Knative serving metrics
