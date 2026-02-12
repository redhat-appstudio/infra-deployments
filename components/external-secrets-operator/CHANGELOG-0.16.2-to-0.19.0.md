# ESO Upgrade: v0.16.2 → v0.19.0 Summary

**Date:** 2026-02-02
**Deployment:** Staging Environment

## Overview

This document summarizes the main changes between External Secrets Operator v0.16.2 and v0.19.0.

---

## Critical Breaking Changes

### 1. Server-Side Apply Required (v0.19.0)
**Impact:** HIGH - Deployment blocker

The CRDs in v0.19.0 have grown significantly larger (see CRD size increases below) and now exceed the annotation size limit for standard `kubectl apply`.

**Error you'll see without this:**
```
The CustomResourceDefinition "externalsecrets.external-secrets.io" is invalid:
metadata.annotations: Too long: must have at most 262144 bytes
```

**Mitigation:**
- ArgoCD: Added `ServerSideApply=true` to sync options ✅
- Manual: Use `kubectl apply --server-side`

### 2. v1beta1 API Removed (v0.17.0)
**Impact:** NONE for us - already migrated to v1 API ✅

All `ExternalSecret`, `SecretStore`, and `ClusterSecretStore` resources must use `apiVersion: external-secrets.io/v1`.

---

## New Features & Capabilities

### 1. New Generator CRDs

#### **MFA Generator (v0.19.0)**
- **Purpose:** Generate TOTP (Time-based One-Time Password) tokens
- **Use Case:** Multi-factor authentication token generation
- **CRD:** `mfas.generators.external-secrets.io`
- **Status:** v1alpha1
- **RFC Compliance:** RFC 6238

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: MFA
metadata:
  name: example-mfa
spec:
  # Generates TOTP tokens
```

#### **SSHKey Generator (v0.19.0)**
- **Purpose:** Generate SSH key pairs (public/private keys)
- **Use Case:** Automated SSH key generation for services
- **CRD:** `sshkeys.generators.external-secrets.io`
- **Status:** v1alpha1
- **Key Types:** RSA, ECDSA, Ed25519

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: SSHKey
metadata:
  name: example-ssh-key
spec:
  # Generates SSH key pairs
```

### 2. New Provider Support

#### **Infisical Provider**
- **Added in:** v0.17.0, enhanced in v0.18.0
- **Authentication Methods:**
  - Azure Auth Credentials (new)
  - Universal Auth
  - Service Token Auth
- **Purpose:** Native integration with Infisical secret management platform
- **Impact on us:** Not used - we use Vault only

### 3. Enhanced Existing Features

#### **Vault Provider Enhancements (v0.17.0)**
- **Namespace-scoped caching:** Improved performance for Vault secrets per namespace
- **Better token management:** Enhanced token renewal and lifecycle
- **Impact on us:** ✅ Automatic performance improvements for our Vault usage

#### **1Password SDK Provider (v0.17.0, v0.18.0)**
- GetSecretMap support
- Better error handling
- **Impact on us:** Not used

---

## CRD Size Increases

The following CRDs saw significant size increases (this is why ServerSideApply is required):

| CRD | Lines Added | Main Reason |
|-----|-------------|-------------|
| `ClusterSecretStore` | +680 | New provider fields (Infisical, IBM enhancements) |
| `SecretStore` | +680 | New provider fields (Infisical, IBM enhancements) |
| `ClusterGenerator` | +76 | Support for new generator types |
| `ClusterExternalSecret` | +44 | Enhanced metadata and status fields |
| `ExternalSecret` | +40 | Enhanced status reporting |
| **NEW:** `MFA` | +100 | TOTP generator CRD |
| **NEW:** `SSHKey` | +77 | SSH key generator CRD |

**Total CRD additions:** ~1,674 lines

---

## Helm Chart Changes

### PodDisruptionBudget
**Change:** Logic reordering for `maxUnavailable` vs `minAvailable`

**Before (v0.16.2):**
```yaml
minAvailable: <value>
maxUnavailable: <value>
```

**After (v0.19.0):**
```yaml
# Prioritizes maxUnavailable if both are set
maxUnavailable: <value>
# Falls back to minAvailable
minAvailable: <value>
```

**Impact:** Better handling of disruption budgets during rolling updates

### Deployment Enhancements
- **Init Containers Support:** New `extraInitContainers` support for custom initialization logic
- **Better Templating:** Improved conditional logic for optional features

### Grafana Dashboard
- Enhanced monitoring dashboards with new metrics
- Better visualization of secret sync status
- Additional panels for generator metrics

---

## Provider Field Additions

### Enhanced Certificate Authority (CA) Support
New `caProvider` field structure for better TLS certificate validation:

```yaml
caProvider:
  type: Secret  # or ConfigMap
  name: ca-cert-bundle
  key: ca.crt
  namespace: cert-namespace  # ClusterSecretStore only
```

**Benefits:**
- Flexible CA certificate sources
- Support for ConfigMaps in addition to Secrets
- Better namespace isolation

### Enhanced Secret Reference Fields
Improved validation and constraints:

```yaml
secretRef:
  name: my-secret
    # Validation: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
    # MaxLength: 253
  key: my-key
    # MaxLength: 253
  namespace: my-namespace
    # MaxLength: 63
    # ClusterSecretStore only
```

**Benefits:**
- Stricter validation prevents misconfigurations
- Better error messages when validation fails
- Kubernetes DNS-1123 compliance

---

## Version-Specific Changes

### v0.17.0 (May 2025)
**Theme:** API Cleanup & Provider Enhancements

- ❌ **BREAKING:** Removed v1beta1 API support (safe for us - already on v1)
- ✅ New 1Password SDK provider
- ✅ Vault caching improvements per namespace
- ✅ Enhanced Infisical support
- ✅ Better error messages and logging

### v0.18.0 (June 2025)
**Theme:** AWS Migration & Feature Additions

- ⚠️ **BREAKING:** AWS SDK V2 migration (doesn't affect us - not using AWS)
- ✅ 1Password GetSecretMap improvements
- ✅ MFA token generator support (new CRD)
- ✅ PodDisruptionBudget chart enhancements
- ✅ Better handling of secret refresh failures

### v0.19.0 (~August 2025)
**Theme:** CRD Expansion & Generators

- ⚠️ **BREAKING:** CRD size increase requires server-side apply
- ✅ SSH key generator support (new CRD)
- ✅ Enhanced Grafana dashboards
- ✅ Improved status reporting in ExternalSecret
- ✅ Better cluster-scoped resource handling
- ✅ Enhanced provider validation

---

## Impact Assessment for Our Deployment

### ✅ Direct Benefits

1. **Performance Improvements**
   - Vault namespace-scoped caching → faster secret retrieval
   - Better token renewal logic → fewer authentication failures
   - Improved reconciliation efficiency

2. **Better Monitoring**
   - Enhanced Grafana dashboards for secret sync visualization
   - Improved metrics for troubleshooting
   - Better status reporting in ExternalSecret resources

3. **Stability Improvements**
   - Better error handling and retry logic
   - Improved secret refresh reliability
   - Enhanced certificate validation

### 🔶 Available But Unused

1. **New Generators**
   - MFA/TOTP generation capability
   - SSH key pair generation
   - Could be useful for future automation needs

2. **New Providers**
   - Infisical integration
   - Enhanced 1Password support
   - Not needed for current Vault-only setup

### ⚠️ No Impact

1. **AWS Changes**
   - SDK V2 migration doesn't affect us
   - We only use Vault provider

2. **v1beta1 Removal**
   - Already migrated to v1 API
   - No action needed

---

## Testing Recommendations

Based on the changes, focus testing on:

### 1. CRD Application
```bash
# Verify CRDs apply successfully with server-side apply
kubectl get crds | grep external-secrets
# Should show all CRDs with no errors
```

### 2. Vault Connection
```bash
# Test Vault provider still works
kubectl get externalsecrets --all-namespaces
# All should show Ready=True

# Check for improved caching behavior
kubectl logs -n external-secrets-operator deployment/external-secrets | grep -i cache
```

### 3. Secret Refresh
```bash
# Verify secrets still refresh on schedule
# Test both 1h and 5m refresh intervals
kubectl describe externalsecret <name> -n <namespace>
```

### 4. Status Reporting
```bash
# Check enhanced status fields
kubectl get externalsecret <name> -n <namespace> -o yaml | grep -A20 status
# Should show more detailed status information
```

### 5. Performance
```bash
# Monitor for performance improvements
kubectl top pods -n external-secrets-operator
# Memory should be stable or slightly improved
```

---

## Rollback Considerations

If issues are found:

1. **CRDs remain at v0.19.0 version** even after rollback
   - This is normal and safe
   - v0.16.2 operator can work with v0.19.0 CRDs

2. **Server-Side Apply can stay enabled**
   - No harm in keeping it enabled for v0.16.2
   - Makes future upgrades easier

3. **Generated secrets persist**
   - Kubernetes secrets created by ESO remain unchanged
   - No data loss during rollback

---

## Documentation & References

### Release Notes
- [v0.17.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v0.17.0)
- [v0.18.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v0.18.0)
- [v0.19.0 Release](https://github.com/external-secrets/external-secrets/releases/tag/v0.19.0)

### Provider Documentation
- [Vault Provider Docs](https://external-secrets.io/latest/provider/hashicorp-vault/)
- [Infisical Provider Docs](https://external-secrets.io/latest/provider/infisical/)

### Generator Documentation
- [MFA Generator](https://external-secrets.io/latest/api/generator/mfa/)
- [SSHKey Generator](https://external-secrets.io/latest/api/generator/sshkey/)

---

## Summary

**Bottom Line:**
- Main changes are new generators (MFA, SSHKey) and provider enhancements (Infisical)
- CRD size increased significantly → requires ServerSideApply ✅
- Vault provider improved with better caching and performance
- No breaking changes that affect our Vault-based deployment
- All changes are backward compatible for our use case

**Risk Level:** LOW
**Expected Benefit:** Medium (performance improvements, better monitoring)
**Required Action:** Monitor staging for 2-3 days before production rollout
