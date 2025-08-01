# 🚀 konflux-info Repository Guide

## 📂 Directory Structure

The `KONFLUX-INFO` directory contains:

```bash
.
├── base/                   # Common resources (e.g., RBAC)
├── production/             # Production cluster configurations
├── staging/                # Staging cluster configurations
├── banner-schema.json      # JSON schema definition for validating banner-content.yaml files

```

Each cluster directory contains:

```bash
.
├── system-alerts # The directory manages auto-generated alerts content shown in the UI
├── banner-content.yaml # The banner content shown in the UI
├── info.json # Metadata about the cluster
└── kustomization.yaml # Kustomize configuration for this cluster, including base, system-alerts, and other configs

```

---

## ✅ Banner Content Validation

To maintain consistency, a GitHub workflow named **`banner-validate`** automatically validates all `banner-content.yaml` files against the schema defined in [`banner-schema.json`](./banner-schema.json).

**When does it run?**

- On any pull request that changes:
  - `banner-schema.json` (schema definition)
  - Any `banner-content.yaml` file (banner configurations)

**What does it check?**

- Ensures the YAML structure matches the schema (e.g., required fields, allowed values, date/time format).
- Prevents invalid or misconfigured banners from being merged.

**How to fix validation errors?**

- Review the error message in the PR checks.
- Compare your changes with the [schema](./banner-schema.json) and [examples in README](#usage-scenarios--examples).

## ✅ Banner Content Specification

The `banner-content.yaml` file defines one or more banners displayed in the Konflux UI. Each cluster has its own `banner-content.yaml` under its directory (e.g., `staging/stone-stage-p01/banner-content.yaml`).

### **Schema**

The schema for banner content is defined in [`banner-schema.json`](./banner-schema.json) and validated automatically by the `banner-validate` GitHub workflow on every PR.

The file must contain a **YAML list** where each item represents a banner configuration.

---

### **Important Behavior**

- The **UI displays only the first valid active banner** from the list, based on current date, time, and optional recurrence settings.
- If multiple banners are configured, **order matters**. Place the highest-priority banner **at the top of the list**.

---

### **Required and Optional Fields for Each Banner**

📎 For the full schema used in CI validation, see banner-schema.json. This table is a human-friendly reference for banner authors.

| Field        | Type   | Required | Description                                                               |
| ------------ | ------ | -------- | ------------------------------------------------------------------------- |
| `summary`    | string | ✅       | Banner text (5–500 chars). **Supports Markdown** (e.g., bold, links).     |
| `type`       | string | ✅       | Banner type: `info`, `warning`, or `danger`.                              |
| `startTime`  | string | ⚠️\*     | Start time in `HH:mm` (24-hour). Required if date-related fields are set. |
| `endTime`    | string | ⚠️\*     | End time in `HH:mm` (24-hour). Required if date-related fields are set.   |
| `timeZone`   | string | ❌       | Optional IANA timezone (e.g., `UTC`, `Asia/Shanghai`). Defaults to UTC.   |
| `year`       | number | ❌       | Year (1970–9999) for one-time banners.                                    |
| `month`      | number | ❌       | Month (1–12).                                                             |
| `dayOfWeek`  | number | ❌       | Day of week (0=Sunday, 6=Saturday) for weekly recurrence.                 |
| `dayOfMonth` | number | ❌       | Day of month (1–31). Required if `year` or `month` is specified.          |

⚠️ **If any of `year`, `month`, `dayOfWeek`, or `dayOfMonth` is specified, both `startTime` and `endTime` are required.**

---

### **Usage Scenarios & Examples**

#### ✅ **1. Multiple Banners**

Example of a `banner-content.yaml` with multiple banners (first active one is shown in UI):

```yaml
- summary: "Scheduled downtime on July 25"
  type: "warning"
  year: 2025
  month: 7
  dayOfMonth: 25
  startTime: "10:00"
  endTime: "14:00"
  timeZone: "UTC"

- summary: "Maintenance every Sunday"
  type: "info"
  dayOfWeek: 0
  startTime: "02:00"
  endTime: "04:00"
  timeZone: "UTC"
```

#### ✅ **2. One-Time Banner**

For a single event on a specific date:

```yaml
- summary: "Scheduled downtime on July 25"
  type: "warning"
  year: 2025
  month: 7
  dayOfMonth: 25
  startTime: "10:00"
  endTime: "14:00"
  timeZone: "UTC"
```

For a single event in today

```yaml
- summary: "Scheduled downtime on July 25"
  type: "warning"
  startTime: "10:00"
  endTime: "14:00"
  timeZone: "UTC"
```

#### ✅ **2. Weekly Recurring Banner**

For an event that repeats every week:

```yaml
- summary: "Maintenance every Sunday"
  type: "info"
  dayOfWeek: 0
  startTime: "02:00"
  endTime: "04:00"
  timeZone: "UTC"
```

#### ✅ **3. Monthly Recurring Banner**

For an event that happens on the same day each month:

```yaml
- summary: "Patch release on 1st of every month"
  type: "info"
  dayOfMonth: 1
  startTime: "01:00"
  endTime: "03:00"
  timeZone: "Asia/Shanghai"
```

#### ✅ **4. Always-On Banner**

For an event that requires immediate notification:

```yaml
- summary: "New feature: Pipeline Insights is live!"
  type: "info"
```

#### ✅ **5. Empty Banner**

When there are no events to announce:

```
[]
```

---

## 📝 How to submit a PR for Banner

1. Locate the target cluster directory:

- For staging: `staging/<cluster-name>/banner-content.yaml`
- For production: `production/<cluster-name>/banner-content.yaml`

2. Edit banner-content.yaml:

- Insert the new banner at the top of the list (highest priority).
- Remove obsolete banners to keep the list clean.

  Example:

  ```yaml
  # New banner on top
  - summary: "New feature rollout on July 30"
    type: "info"
    year: 2025
    month: 7
    dayOfMonth: 30
    startTime: "09:00"
    endTime: "17:00"
    timeZone: "UTC"

  # Keep other active banners below
  - summary: "Maintenance every Sunday"
    type: "info"
    dayOfWeek: 0
    startTime: "02:00"
    endTime: "04:00"
    timeZone: "UTC"
  ```

3. Submit a Pull Request:

- Modify only the target cluster’s banner-content.yaml.
  In the PR description, include:
- Target cluster (e.g., kflux-ocp-p01)
- Type of change (e.g., new banner / update / remove obsolete)
- Purpose of change (e.g., release announcement, downtime notice)

  Example:

  ```yaml
  Target cluster: kflux-ocp-p01
  Type: New banner
  Purpose: Release announcement for Konflux 1.2
  ```

---

## 📢 System Alerts

We enables the infrastructure team to automatically surface specific operational issues or warnings in the Konflux UI.

These alerts would be auto-generated from monitoring systems or automation scripts, written as Kubernetes ConfigMaps, and automatically picked up by the Konflux UI to inform users of system-wide conditions.

### ✅ Alert YAML Format

Each file under `system-alerts/` must be a valid Kubernetes ConfigMap and would be considered as one alert.

We use below simple JSON format inside the ConfigMap to make generation via scripts easier and more reliable:

- Only two fields are supported in the alert content:

  - summary: A short, user-facing message.
  - type: One of info, warning, or danger.

- JSON is used instead of YAML for ease of script generation.

- The label `konflux.system.alert: "true"` is required — this is how the Konflux UI discovers and filters alert ConfigMaps.

Here is an example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: konflux-system-alert-xyz
  namespace: konflux-info
  labels:
    konflux.system.alert: "true"
data:
  alert-content.json: |
    {
      "summary": "Builds are delayed due to maintenance",
      "type": "warning"
    }
```

Note there is no CI validation for the `system-alert-content.json` payload.

### Folder Structure

When there are active alerts, the folder structure is:

```bash

system-alerts/   # Alert ConfigMaps (one file = one alert)
.
├── alert-1.yaml        # A example valid ConfigMap containing alert content
├── alert-2.yaml
└── kustomization.yaml  # Auto-generated, includes all alert YAMLs

```

When there are no alerts, the `system-alerts/` folder and its reference in the parent `kustomization.yaml` should be both removed automatically to avoid kustomize errors.

### When to Add or Remove an Alert

These ConfigMaps are automatically generated by monitoring or scripting systems based on the current system state.

- Add an alert

  - A new YAML file should be generated under `system-alerts` when a new system condition or issue needs to be surfaced to users.
  - `kustomization.yaml` under `system-alerts` should be refreshed to ensure the new YAML file is covered.
  - If the `system-alerts/` directory does not exist (e.g. no previous alerts), you must:
    - Create the `system-alerts/` directory.
    - Generate its local `kustomization.yaml`
    - Ensure the parent kustomization.yaml includes a reference to `system-alerts/`.

- To remove an alert:
  - Delete the corresponding alert YAML file from `system-alerts/.`
  - `kustomization.yaml` under `system-alerts` should be refreshed to ensure the deleted YAML has been removed.
  - If no alerts remain:
    - Delete the `system-alerts/` directory.
    - Remove its reference from the parent `kustomization.yaml`.

⚙️ These add/remove actions are expected to be handled by automation. Manual edits are discouraged unless for emergency override or debugging purposes.
