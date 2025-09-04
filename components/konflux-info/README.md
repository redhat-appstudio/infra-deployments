_Follow links by Ctrl + click (or Cmd + click on Mac)_

- [📂 Directory Structure](#DirectoryStructure)
- [📢 Konflux Banner](#KonfluxBanner)
  - [✅ Banner Content Validation](#BannerContentValidation)
  - [✅ Banner Content Specification](#BannerContentSpecification)
    - [**Schema**](#Schema)
    - [**Required and Optional Fields for Each Banner**](#RequiredandOptionalFieldsforEachBanner)
  - [Usage Scenarios & Examples](#UsageScenariosExamples)
    - [✅ **1. Multiple Banners**](#1.MultipleBanners)
    - [✅ **2. One-Time Banner**](#2.One-TimeBanner)
    - [✅ **3. Weekly Recurring Banner**](#3.WeeklyRecurringBanner)
    - [✅ **4. Monthly Recurring Banner**](#4.MonthlyRecurringBanner)
    - [✅ **5. Always-On Banner**](#5.Always-OnBanner)
    - [✅ **6. Empty Banner**](#6.EmptyBanner)
  - [📝 How to submit a PR for Banner](#HowtosubmitaPRforBanner)
  - [✅ UI Behavior](#UIBehavior)
  - [❓ Frequently Asked Questions](#FrequentlyAskedQuestions)

# 🚀 konflux-info Repository Guide

## <a name='DirectoryStructure'></a>📂 Directory Structure

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
├── banner-content.yaml # The banner content shown in the UI
├── info.json # Metadata about the cluster
└── kustomization.yaml # Kustomize configuration for this cluster

```

---

## <a name='KonfluxBanner'></a>📢 Konflux Banner

### <a name='BannerContentValidation'></a>✅ Banner Content Validation

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

### <a name='BannerContentSpecification'></a>✅ Banner Content Specification

The `banner-content.yaml` file defines one or more banners displayed in the Konflux UI. Each cluster has its own `banner-content.yaml` under its directory (e.g., `staging/stone-stage-p01/banner-content.yaml`).

#### <a name='Schema'></a>**Schema**

The schema for banner content is defined in [`banner-schema.json`](./banner-schema.json) and validated automatically by the `banner-validate` GitHub workflow on every PR.

The file must contain a **YAML list** where each item represents a banner configuration.

---

#### <a name='RequiredandOptionalFieldsforEachBanner'></a>**Required and Optional Fields for Each Banner**

📎 For the full schema used in CI validation, see banner-schema.json. This table is a human-friendly reference for banner authors.

| Field        | Type   | Required | Description                                                               |
| ------------ | ------ | -------- | ------------------------------------------------------------------------- |
| `summary`    | string | ✅       | Banner text (5–500 chars). **Supports Markdown** (e.g., bold, links).     |
| `type`       | string | ✅       | Banner type: `info`, `warning`, or `danger`.                              |
| `startTime`  | string | ⚠️\*     | Start time in `HH:mm` (24-hour). Required if date-related fields are set. |
| `endTime`    | string | ⚠️\*     | End time in `HH:mm` (24-hour). Required if date-related fields are set.   |
| `timeZone`   | string | ❌       | Optional IANA time zone (e.g., Asia/Shanghai). Omit for UTC (default).    |
| `year`       | number | ❌       | Year (1970–9999) for one-time banners.                                    |
| `month`      | number | ❌       | Month (1–12).                                                             |
| `dayOfWeek`  | number | ❌       | Day of week (0=Sunday, 6=Saturday) for weekly recurrence.                 |
| `dayOfMonth` | number | ❌       | Day of month (1–31). Required if `year` or `month` is specified.          |

⚠️ **If any of `year`, `month`, `dayOfWeek`, or `dayOfMonth` is specified, both `startTime` and `endTime` are required.**

---

### <a name='UsageScenariosExamples'></a>Usage Scenarios & Examples

#### <a name='1.MultipleBanners'></a>✅ **1. Multiple Banners**

Example of a `banner-content.yaml` with multiple banners (first active one is shown in UI):

```yaml
- summary: "Scheduled downtime on July 25"
  type: "warning"
  year: 2025
  month: 7
  dayOfMonth: 25
  startTime: "10:00"
  endTime: "14:00"
  timeZone: "America/Los_Angeles"

- summary: "Maintenance every Sunday"
  type: "info"
  dayOfWeek: 0
  startTime: "02:00"
  endTime: "04:00"
  # No timezone is needed when you expect it's UTC.
```

#### <a name='2.One-TimeBanner'></a>✅ **2. One-Time Banner**

For a single event on a specific date:

```yaml
- summary: "Scheduled downtime on July 25"
  type: "warning"
  year: 2025
  month: 7
  dayOfMonth: 25
  startTime: "10:00"
  endTime: "14:00"
```

For a single event in today

```yaml
- summary: "Scheduled downtime on July 25"
  type: "warning"
  startTime: "10:00"
  endTime: "14:00"
```

#### <a name='3.WeeklyRecurringBanner'></a>✅ **3. Weekly Recurring Banner**

For an event that repeats every week:

```yaml
- summary: "Maintenance every Sunday"
  type: "info"
  dayOfWeek: 0
  startTime: "02:00"
  endTime: "04:00"
```

#### <a name='4.MonthlyRecurringBanner'></a>✅ **4. Monthly Recurring Banner**

For an event that happens on the same day each month:

```yaml
- summary: "Patch release on 1st of every month"
  type: "info"
  dayOfMonth: 1
  startTime: "01:00"
  endTime: "03:00"
  timeZone: "Asia/Shanghai"
```

#### <a name='5.Always-OnBanner'></a>✅ **5. Always-On Banner**

For an event that requires immediate notification:

```yaml
- summary: "New feature: Pipeline Insights is live!"
  type: "info"
```

#### <a name='6.EmptyBanner'></a>✅ **6. Empty Banner**

When there are no events to announce:

```
[]
```

---

### <a name='HowtosubmitaPRforBanner'></a>📝 How to submit a PR for Banner

1. Locate the target cluster directory:

- For staging: `staging/<cluster-name>/banner-content.yaml`
- For production: `production/<cluster-name>/banner-content.yaml`

2. Edit banner-content.yaml:

- **Insert the new banner at the top of the list**.
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

  # Keep other active banners below
  - summary: "Maintenance every Sunday"
    type: "info"
    dayOfWeek: 0
    startTime: "02:00"
    endTime: "04:00"
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

### <a name='UIBehavior'></a>✅ UI Behavior

- The **UI displays only the first valid active banner** from the list, based on current date, time, and optional recurrence settings.
- If multiple banners are configured, **order matters**.
- **Time-related fields like `startTime` and `endTime` are not displayed in the UI**; they only control when the banner is active.

  **To convey duration or timing details, please include them within the `summary`.**

- **The `type` and `summary` fields are displayed directly in the UI**.
- We enjoyed leveraging the [PatternFly Banner component (v5)](https://v5-archive.patternfly.org/components/banner/) to implement the UI, following its design principles for clarity and consistency.

---

### <a name='FrequentlyAskedQuestions'></a>❓ Frequently Asked Questions

- Why is only one banner shown even when multiple are configured?

  **We follow the [PatternFly design guidelines](https://www.patternfly.org/components/banner/design-guidelines) for banners**, which emphasize simplicity and clarity. Showing just one banner line at a time helps avoid overwhelming users and ensures that important messages aren't lost in clutter.

- What does “first active” actually mean?

  **The term 'first' doesn't imply priority or severity** — it simply refers to the first banner that is currently active based on time and repeat configuration.

  If a banner was scheduled in the past, it should already have been displayed.

  If it's scheduled in the future, it will show when its time comes.

  At any given moment, the system checks which banner is active right now, and picks the first one that matches the criteria.

  🕒 Banners use fields like `startTime`, `endTime`, `dayOfWeek`, etc., to precisely define when they should appear.

  **📝 If multiple messages need to be shared at the same time, consider combining them into a well-written summary inside a single banner.**
