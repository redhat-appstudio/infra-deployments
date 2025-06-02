# 🚀 konflux-info Repository Guide

## 📂 Directory Structure

```bash
.
├── base/                   # Common resources (e.g., RBAC)
├── production/             # Production cluster configurations
├── staging/                # Staging cluster configurations
├── banner-schema.json      # JSON schema definition for validating banner-content.yaml files
```

Each cluster directory contains:

- `banner-content.yaml` 👉 The banner content shown in the UI
- `info.json` 👉 Metadata about the cluster
- `kustomization.yaml` 👉 Kustomize configuration for this cluster

---

## ✅ Banner Content Validation

A GitHub workflow named `banner-validate` automatically checks that each `banner-content.yaml` file conforms to the schema defined in `banner-schema.json`.  
This workflow runs whenever either the schema or any `banner-content.yaml` file is changed.  
The schema (`banner-schema.json`) specifies the required structure and fields for banner content, ensuring consistency and correctness across environments.

---

## 📝 How to submit a PR

1. Modify only the files relevant to your target cluster, e.g.: `staging/stone-stage-p01/banner-content.yaml` or `production/kflux-ocp-p01/banner-content.yaml`
2. In your PR description, include:
  - Target cluster (e.g. kflux-ocp-p01)
  - Type of change (e.g. new banner / update info / typo fix)
  - Purpose of change (e.g. downgrade notification / release announcement)

---

