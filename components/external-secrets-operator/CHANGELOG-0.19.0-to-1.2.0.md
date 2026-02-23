# ESO Upgrade: v0.19.0 → v1.2.0 Summary

**Date:** 2026-02-13
**Deployment:** Staging Environment

## Overview

This document summarizes the main changes between External Secrets Operator v0.19.0 and v1.2.0.

---

## Breaking Changes

### None Affecting Our Use Case

No breaking changes were introduced between v0.19.0 and v1.2.0 that impact our Vault-based deployment.

**Notable changes that don't affect us:**
- **Image registry migration (v1.1.0):** The project migrated from `oci.external-secrets.io` to `ghcr.io/external-secrets/external-secrets`. Our `values.yaml` already specifies the image explicitly.
- **AWS SDK v1 removal (v1.1.0):** Not using AWS providers.

---

## New Features & Capabilities

### 1. GA Release (v1.0.0)

**Milestone:** First stable release with semver guarantees.

- **Semver commitment:** API stability guarantees begin from v1.0.0 onwards
- **Go module separation:** Restructured Go module organization for better maintainability
- **Dynamic target implementation:** Enhanced external secrets source targeting

### 2. Operational Improvements

#### **Liveness Probe (v0.20.0)**
- Liveness probe added to the ESO controller
- Improves health monitoring and automatic restart on failures
- **Impact on us:** ✅ Better pod health management out of the box

#### **Secure Metrics Serving (v0.20.0)**
- Metrics endpoint can now be served over HTTPS
- HTTP/2 configurability option added
- **Impact on us:** ✅ Enhanced security for metrics scraping

#### **Force Sync Annotation (v0.20.0)**
- New annotation to force immediate reconciliation of an ExternalSecret
- **Impact on us:** ✅ Useful for troubleshooting

```bash
# Force an immediate sync
kubectl annotate externalsecret <name> -n <namespace> force-sync="$(date +%s)" --overwrite
```

#### **Store Deprecation Mechanism (v1.2.0)**
- Formal process for deprecating SecretStores
- **Impact on us:** ✅ Helpful for phasing out old stores in the future

#### **SecretStore Reconcile Flag (v1.2.0)**
- New flag to enable/disable SecretStore reconciliation
- **Impact on us:** ✅ Useful for maintenance windows

### 3. Vault Provider Enhancements

#### **Pod Identity Authentication (v0.20.0)**
- New authentication method for Vault using Kubernetes Pod Identity
- **Impact on us:** Not using currently (we use AppRole), but available as a future option

#### **Check-and-Set for Push Secrets (v0.20.0)**
- CAS (Check-and-Set) functionality for Vault push secrets
- **Impact on us:** Not using PushSecrets, but available if needed

#### **GCP Workload Identity for Vault (v1.1.0)**
- Authenticate to Vault using GCP Workload Identity
- **Impact on us:** Not using GCP, but available as a future option

#### **SecretStore Finalizers (v0.20.0)**
- Finalizers added for SecretStores with PushSecrets deletion policy
- Prevents accidental deletion of stores that are in use
- **Impact on us:** ✅ Additional safety for store management

### 4. New Providers & Generators

These don't affect our Vault-only use case but are available:

| Version | New Provider/Generator | Description |
|---------|----------------------|-------------|
| v0.20.0 | Volcengine provider | Volcengine secret management |
| v0.20.0 | Cloudsmith generator | Container registry authentication tokens |
| v1.0.0 | esoctl bootstrap generators | CLI utility improvements |
| v1.1.0 | ECDSA SSH key support | Additional SSH key type for generator |
| v1.2.0 | Barbican provider | OpenStack secret management |
| v1.2.0 | Doppler OIDC auth | OIDC-based authentication for Doppler |

### 5. SSH Key Generator Enhancement (v1.1.0)
- Added ECDSA key type support alongside RSA and Ed25519
- **Impact on us:** Not using SSH key generator

---

## CRD Changes

### New CRD

| CRD | Version | Description |
|-----|---------|-------------|
| `cloudsmithaccesstokens.generators.external-secrets.io` | v0.20.0 | Cloudsmith registry access token generator |

### Modified CRDs

CRDs continue to grow with new provider fields. Server-side apply (already enabled) handles the size requirements.

---

## Helm Chart Changes

### Global Values (v1.2.0)
- New `global` values block for common deployment configurations
- Supports shared `nodeSelector`, `tolerations`, `topologySpreadConstraints`, `affinity`
- OpenShift compatibility via `global.compatibility.openshift.adaptSecurityContext`
- **Impact on us:** ✅ Cleaner configuration for shared settings

### Image Registry (v1.1.0)
- Default image repository changed from `oci.external-secrets.io/external-secrets/external-secrets` to `ghcr.io/external-secrets/external-secrets`
- **Impact on us:** Our `values.yaml` overrides the repository explicitly, so no change needed

### Certificate Duration (v1.0.0)
- Normalized certificate duration default value in Helm chart
- **Impact on us:** ✅ Better defaults for webhook certificates

### ProcessClusterGenerator (v0.20.0)
- New boolean to control cluster generator processing
- **Impact on us:** Default behavior unchanged

### Bitwarden Dependency (v1.2.0)
- Bitwarden SDK server chart dependency updated to v0.5.2
- **Impact on us:** Bitwarden is disabled in our deployment

---

## Version-Specific Changes

### v0.20.0 (September 22, 2024)
**Theme:** Operational Improvements & New Providers

- ✅ Liveness probe added to controller
- ✅ Secure metrics serving (HTTPS)
- ✅ Force sync annotation support
- ✅ SecretStore finalizers for PushSecrets
- ✅ Pod Identity auth for Vault (optional)
- ✅ Check-and-set for Vault push secrets
- New Volcengine provider
- New Cloudsmith generator
- GCP Workload Identity Federation support

### v1.0.0 (November 7, 2024)
**Theme:** GA Release & Stability

- ✅ Semver guarantees begin
- ✅ Go module separation architecture
- ✅ Dynamic target implementation
- ✅ esoctl bootstrap generator commands
- Certificate duration normalization in Helm

### v1.1.0 (November 21, 2024)
**Theme:** Registry Migration & Provider Enhancements

- ⚠️ Image registry migrated to ghcr.io (our values.yaml overrides this)
- ✅ ECDSA SSH key support
- ✅ GCP Workload Identity auth for Vault
- ✅ Provider build tags (compile-time provider selection)
- AWS SDK v1 usage fully removed
- Darwin ARM64 binary releases

### v1.2.0 (December 19, 2024)
**Theme:** Store Management & New Providers

- ✅ Store deprecation mechanism
- ✅ SecretStore reconcile enable/disable flag
- ✅ Global Helm values for common configurations
- New Barbican provider (OpenStack)
- Doppler OIDC-based authentication
- SecretServer provider promoted to beta

---

## Impact Assessment for Our Deployment

### ✅ Direct Benefits

1. **Operational Improvements**
   - Liveness probes → automatic restart on controller failures
   - Secure metrics serving → HTTPS for metrics endpoint
   - Force sync annotation → faster troubleshooting
   - SecretStore finalizers → prevents accidental store deletion

2. **Stability & Maturity**
   - GA release with semver guarantees
   - Battle-tested by community since November 2024
   - Improved error handling and reconciliation

3. **Configuration Management**
   - Store deprecation mechanism for phasing out old stores
   - Global Helm values for cleaner configuration
   - SecretStore reconcile flag for maintenance windows

### 🔶 Available But Unused

1. **Vault Auth Methods**
   - Pod Identity authentication (we use AppRole)
   - GCP Workload Identity for Vault

2. **New Providers**
   - Barbican, Volcengine, Doppler OIDC
   - Not needed for current Vault-only setup

3. **New Generators**
   - Cloudsmith access token generator
   - ECDSA SSH key support

### ⚠️ No Impact

1. **Registry Migration**
   - Our `values.yaml` explicitly sets the image repository
   - No action needed

2. **AWS Changes**
   - AWS SDK v1 fully removed
   - We only use Vault provider

---

## Testing Recommendations

Based on the changes, focus testing on:

### 1. Pod Health & Liveness
```bash
# Verify liveness probe is active
kubectl get deployment -n external-secrets-operator external-secrets -o yaml | grep -A 5 livenessProbe

# Check pod restarts (should be 0)
kubectl get pods -n external-secrets-operator
```

### 2. Vault Connection
```bash
# Test Vault provider still works
kubectl get externalsecrets --all-namespaces
# All should show Ready=True

# Verify AppRole authentication
kubectl logs -n external-secrets-operator deployment/external-secrets --tail=50 | grep -i vault
```

### 3. Secret Refresh
```bash
# Verify secrets still refresh on schedule
# Test both 1h and 5m refresh intervals
kubectl describe externalsecret <name> -n <namespace> | grep -A 5 "Last Transition"
```

### 4. Metrics Endpoint
```bash
# Verify metrics are still accessible
kubectl port-forward -n external-secrets-operator svc/cluster-external-secrets-metrics 8080:8080 &
curl http://localhost:8080/metrics | head -20
```

### 5. CRD Validation
```bash
# Verify all CRDs are present and healthy
kubectl get crds | grep external-secrets

# Check for new Cloudsmith CRD
kubectl get crds cloudsmithaccesstokens.generators.external-secrets.io
```

### 6. Resource Usage
```bash
# Monitor memory and CPU usage
kubectl top pods -n external-secrets-operator
# Compare with pre-upgrade baseline
```

---

## Rollback Considerations

If issues are found:

1. **CRDs remain at v1.2.0 version** even after rollback
   - This is normal and safe
   - v0.19.0 operator can work with v1.2.0 CRDs

2. **Staging rollback is straightforward**
   - Revert `staging/kustomization.yaml` to reference `../base`
   - Remove staging-specific files
   - ArgoCD will sync back to v0.19.0 from base

3. **Generated secrets persist**
   - Kubernetes secrets created by ESO remain unchanged
   - No data loss during rollback

4. **Server-Side Apply stays enabled**
   - Already required for v0.19.0
   - No change needed for rollback

---

## Documentation & References

### Release Notes
- [v0.20.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v0.20.0)
- [v1.0.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v1.0.0)
- [v1.1.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v1.1.0)
- [v1.2.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v1.2.0)

### Provider Documentation
- [Vault Provider Docs](https://external-secrets.io/latest/provider/hashicorp-vault/)
- [Barbican Provider Docs](https://external-secrets.io/latest/provider/barbican/)

### Stability & Support
- [ESO Stability and Support Policy](https://external-secrets.io/latest/introduction/stability-support/)

---

## Summary

**Bottom Line:**
- This is primarily a stability and maturity upgrade (v0.19.0 → GA v1.2.0)
- Key operational improvements: liveness probes, secure metrics, force sync, store deprecation
- Vault provider remains stable with full backward compatibility
- No breaking changes affect our Vault/AppRole deployment
- New features (store deprecation, global values) improve manageability

**Risk Level:** LOW
**Expected Benefit:** Medium (operational improvements, GA stability guarantees)
**Required Action:** Monitor staging for 2-3 days before production rollout
