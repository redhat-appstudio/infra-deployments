# External Secrets Operator Helm Deployment for stone-stage-p01

This directory contains the Helm-based deployment configuration for External Secrets Operator (ESO) on the `stone-stage-p01` cluster, replacing the OLM-based deployment.

## Migration Overview

This migration switches from OLM Subscription-based deployment to Helm chart deployment while maintaining:
- Same version (0.11.0)
- Same resource limits and requests
- Same Prometheus monitoring configuration
- Same namespace (external-secrets-operator)
- Same labels and ServiceMonitor configuration

## Files

- `helm-values.yaml`: Helm chart values matching the OLM OperatorConfig settings
- `custom-resources.yaml`: Custom Service and ServiceMonitor manifests to ensure exact label matching with OLM deployment
- `kustomization.yaml`: Kustomize configuration for applying custom resources

## Migration Process

The migration happens automatically via ArgoCD:

1. **Helm ApplicationSet** (`external-secrets-operator-helm-stone-stage-p01`) deploys the Helm chart first (sync-wave: -1)
2. **Custom Resources ApplicationSet** (`external-secrets-operator-custom-resources-stone-stage-p01`) applies custom Service/ServiceMonitor (sync-wave: 0)
3. **OLM ApplicationSet** (`external-secrets-operator`) automatically removes OLM resources for stone-stage-p01 via pruning (sync-wave: implicit)

## Configuration Details

### Resource Limits (matching OLM OperatorConfig)

- **Controller**: 100m CPU / 512Mi memory requests, 512Mi memory limit
- **Cert Controller**: 100m CPU / 128Mi memory requests, 128Mi memory limit
- **Webhook**: 100m CPU / 128Mi memory requests, 128Mi memory limit

### Prometheus Monitoring

- Enabled on port 8080
- ServiceMonitor configured with matching labels
- Metrics Service: `cluster-external-secrets-metrics`

## Verification

After migration, verify:

```bash
# Check Helm release
helm list -n external-secrets-operator

# Check operator pods
kubectl get pods -n external-secrets-operator

# Verify ExternalSecret resources reconcile
kubectl get externalsecrets -A

# Check ServiceMonitor
kubectl get servicemonitor -n external-secrets-operator

# Verify metrics endpoint
kubectl port-forward -n external-secrets-operator svc/cluster-external-secrets-metrics 8080:8080
curl http://localhost:8080/metrics
```

## Rollback

If issues occur:

1. Remove Helm ApplicationSets from ArgoCD
2. Revert the OLM ApplicationSet exclusion (remove selector matchExpressions)
3. ArgoCD will restore OLM deployment automatically

