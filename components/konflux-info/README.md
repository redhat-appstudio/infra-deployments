# 🚀 konflux-info Repository Guide

## 📂 Directory Structure

```bash
.
├── base/                   # Common resources (e.g., RBAC)
├── production/              # Production cluster configurations
├── staging/                 # Staging cluster configurations
├── validator                # Local validation support for banner content
├── Makefile                 # For easier local validation
```

Each cluster directory contains:

- `banner-content.yaml` 👉 The banner content shown in the UI
- `info.json` 👉 Metadata about the cluster
- `kustomization.yaml` 👉 Kustomize configuration for this cluster

---

## 📝 How to submit a PR

1. Modify only the files relevant to your target cluster, e.g.: `staging/stone-stage-p01/banner-content.yaml` or `production/kflux-ocp-p01/banner-content.yaml`
2. Before submitting, make sure your changes pass local validation (see below).
3. In your PR description, include:
   - Target cluster (e.g. kflux-ocp-p01)
   - Type of change (e.g. new banner / update info / typo fix)
   - Purpose of change (e.g. downgrade notification / release announcement)

---

## 🛠 How to validate locally(Optional)

This repository supports two types of validation:
✅ Banner content validation — ensures banner-content.yaml meets the JSON schema and contains no unsafe HTML content.
✅ Kustomize validation — ensures kustomization.yaml files can be built successfully with kustomize.

Make sure you have:

```bash
kustomize (version >= v3.8)
kubectl (optional, for local apply tests)
make (optinal, for easier validation)
```

Validate a single cluster:

```bash
kustomize build production/kflux-ocp-p01
validator/validator validator/banner-schema.json production/kflux-ocp-p01
```

Validate all production/staging clusters, take production as the example:

```bash
for d in production/*/ ; do
  echo ">>> Verifying $d"
  kustomize build "$d" >/dev/null || exit 1
  validator/validator validator/banner-schema.json production/
done
```

---

## ⚡ Makefile for easier validation

A Makefile is provided to simplify validation:

```bash
make verify-production-banner
make verify-production-kustomize
make verify-staging-banner
make verify-staging-kustomize
make verify-production
make verify-staging
make verify-all
```
