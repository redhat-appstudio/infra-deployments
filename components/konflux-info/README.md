_Follow links by Ctrl + click (or Cmd + click on Mac)_

- 1. [📂 Directory Structure](#DirectoryStructure)
- 2. [📢 Konflux Banner](#KonfluxBanner)
  - 2.1. [✅ Banner Content Validation](#BannerContentValidation)
  - 2.2. [✅ Banner Content Specification](#BannerContentSpecification)
    - 2.2.1. [**Schema**](#Schema)
    - 2.2.2. [**Required and Optional Fields for Each Banner**](#RequiredandOptionalFieldsforEachBanner)
  - 2.3. [Usage Scenarios & Examples](#UsageScenariosExamples)
    - 2.3.1. [✅ **1. Multiple Banners**](#1.MultipleBanners)
    - 2.3.2. [✅ **2. One-Time Banner**](#2.One-TimeBanner)
    - 2.3.3. [✅ **3. Weekly Recurring Banner**](#3.WeeklyRecurringBanner)
    - 2.3.4. [✅ **4. Monthly Recurring Banner**](#4.MonthlyRecurringBanner)
    - 2.3.5. [✅ **5. Always-On Banner**](#5.Always-OnBanner)
    - 2.3.6. [✅ **6. Empty Banner**](#6.EmptyBanner)
  - 2.4. [📝 How to submit a PR for Banner](#HowtosubmitaPRforBanner)
  - 2.5. [✅ UI Behavior](#UIBehavior)
  - 2.6. [❓ Frequently Asked Questions](#FrequentlyAskedQuestions)
- 3. [📢 System Notifications](#SystemNotifications)
  - 3.1. [✅ **Notification JSON Format**](#NotificationJSONFormat)
  - 3.2. [✅ **Example Notification ConfigMap**](#ExampleNotificationConfigMap)
  - 3.3. [✅ **Notification Content Validation**](#NotificationContentValidation)
  - 3.4. [✅ **Folder Structure**](#FolderStructure)
  - 3.5. [✅ **UI Behavior**](#UIBehavior-1)
  - 3.6. [✅ **When to Add or Remove Notifications**](#WhentoAddorRemoveNotifications)

# 🚀 konflux-info Repository Guide

## 1. <a name='DirectoryStructure'></a>📂 Directory Structure

The `KONFLUX-INFO` directory contains:

```bash
.
├── base/                   # Common resources (e.g., RBAC)
├── production/             # Production cluster configurations
├── staging/                # Staging cluster configurations
├── banner-schema.json      # JSON schema definition for validating banner-content.yaml files
├── notification-schema.json      # JSON schema definition for validating notification-content.json
```

Each cluster directory contains:

```bash
.
├── system-notifications # The directory manages auto-generated notifications content shown in the UI
├── banner-content.yaml # The banner content shown in the UI
├── info.json # Metadata about the cluster
└── kustomization.yaml # Kustomize configuration for this cluster, including base, system-notifications, and other configs

```

---

## 2. <a name='KonfluxBanner'></a>📢 Konflux Banner

### 2.1. <a name='BannerContentValidation'></a>✅ Banner Content Validation

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

### 2.2. <a name='BannerContentSpecification'></a>✅ Banner Content Specification

The `banner-content.yaml` file defines one or more banners displayed in the Konflux UI. Each cluster has its own `banner-content.yaml` under its directory (e.g., `staging/stone-stage-p01/banner-content.yaml`).

#### 2.2.1. <a name='Schema'></a>**Schema**

The schema for banner content is defined in [`banner-schema.json`](./banner-schema.json) and validated automatically by the `banner-validate` GitHub workflow on every PR.

The file must contain a **YAML list** where each item represents a banner configuration.

---

#### 2.2.2. <a name='RequiredandOptionalFieldsforEachBanner'></a>**Required and Optional Fields for Each Banner**

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

### 2.3. <a name='UsageScenariosExamples'></a>Usage Scenarios & Examples

#### 2.3.1. <a name='1.MultipleBanners'></a>✅ **1. Multiple Banners**

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

#### 2.3.2. <a name='2.One-TimeBanner'></a>✅ **2. One-Time Banner**

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

#### 2.3.3. <a name='3.WeeklyRecurringBanner'></a>✅ **3. Weekly Recurring Banner**

For an event that repeats every week:

```yaml
- summary: "Maintenance every Sunday"
  type: "info"
  dayOfWeek: 0
  startTime: "02:00"
  endTime: "04:00"
```

#### 2.3.4. <a name='4.MonthlyRecurringBanner'></a>✅ **4. Monthly Recurring Banner**

For an event that happens on the same day each month:

```yaml
- summary: "Patch release on 1st of every month"
  type: "info"
  dayOfMonth: 1
  startTime: "01:00"
  endTime: "03:00"
  timeZone: "Asia/Shanghai"
```

#### 2.3.5. <a name='5.Always-OnBanner'></a>✅ **5. Always-On Banner**

For an event that requires immediate notification:

```yaml
- summary: "New feature: Pipeline Insights is live!"
  type: "info"
```

#### 2.3.6. <a name='6.EmptyBanner'></a>✅ **6. Empty Banner**

When there are no events to announce:

```
[]
```

---

### 2.4. <a name='HowtosubmitaPRforBanner'></a>📝 How to submit a PR for Banner

1. Locate the target cluster directory:

- For staging: `staging/<cluster-name>/banner-content.yaml`
- For production: `production/<cluster-name>/banner-content.yaml`

2. Edit banner-content.yaml:

- <strong style="color: red;">Insert the new banner at the top of the list</strong>.
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

### 2.5. <a name='UIBehavior'></a>✅ UI Behavior

- The <strong style="color: red;">UI displays only the first valid active banner</strong> from the list, based on current date, time, and optional recurrence settings.
- If multiple banners are configured, <strong style="color: red;">order matters</strong>.
- <strong style="color: red;">Time-related fields like `startTime` and `endTime` are not displayed in the UI</strong>; they only control when the banner is active.

  <strong>To convey duration or timing details, please include them within the `summary`.</strong>

- <strong style="color: red;">The `type` and `summary` fields are displayed directly in the UI</strong>.
- We enjoyed leveraging the [PatternFly Banner component (v5)](https://v5-archive.patternfly.org/components/banner/) to implement the UI, following its design principles for clarity and consistency.

### 2.6. <a name='FrequentlyAskedQuestions'></a>❓ Frequently Asked Questions

- Why is only one banner shown even when multiple are configured?

  <strong style="color: red;">We follow the [PatternFly design guidelines](https://www.patternfly.org/components/banner/design-guidelines) for banners</strong>, which emphasize simplicity and clarity. Showing just one banner line at a time helps avoid overwhelming users and ensures that important messages aren't lost in clutter.

- What does “first active” actually mean?

  <strong style="color: red;">The term 'first' doesn’t imply priority or severity</strong> — it simply refers to the first banner that is currently active based on time and repeat configuration.

  If a banner was scheduled in the past, it should already have been displayed.

  If it's scheduled in the future, it will show when its time comes.

  At any given moment, the system checks which banner is active right now, and picks the first one that matches the criteria.

  🕒 Banners use fields like `startTime`, `endTime`, `dayOfWeek`, etc., to precisely define when they should appear.

  <strong style="color: red;">📝 If multiple messages need to be shared at the same time, consider combining them into a well-written summary inside a single banner.</strong>

## 3. <a name='SystemNotifications'></a>📢 System Notifications

The infrastructure team uses System Notifications to automatically surface important operational notifications in the Konflux UI.

These notifications are generated from monitoring systems or automation scripts as Kubernetes ConfigMaps.

The Konflux UI detects and displays these notifications to inform users about system-wide conditions.

### 3.1. <a name='NotificationJSONFormat'></a>✅ **Notification JSON Format**

System notifications are stored as Kubernetes ConfigMaps in the `system-notifications/` directory.

Each ConfigMap contains notification data in the `notification-content.json` field as JSON.

<strong>Key points:</strong>

- The JSON payload supports these fields for each notification object:

  - <strong>title (optional)</strong>: A short heading.
  - <strong>summary (required)</strong>: A brief, user-facing message displayed as the notification content.
  - <strong>type (required)</strong>: It sets the bell icon. Allowed values: `info`, `warning`, or `danger`.
  - <strong>activeTimestamp (optional)</strong>: The time when the notification should be shown, specified as an ISO 8601 timestamp (e.g., `"2025-08-11T11:08:17Z"`).
    - If set to a future time, the UI will delay displaying the notification until that time is reached.
    - If not set, the system falls back to using the resource’s raw `creationTimestamp` to decide when to show the notification.

- Payload structure:

  We recommend using <strong>a single JSON object representing one notification</strong> in `notification-content.json`.

  However, <strong>a JSON list (array) of notification objects is also allowed</strong> if multiple notifications need to be included in one ConfigMap.

### 3.2. <a name='ExampleNotificationConfigMap'></a>✅ **Example Notification ConfigMap**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: konflux-system-notification-xyz
  namespace: konflux-info
  labels:
    konflux.system.notification: "true"
data:
  notification-content.json: |-
    {
      "summary": "Builds are delayed due to maintenance",
      "type": "warning",
      "title": "From Builds Service",
      "activeTimestamp": "2025-08-12T14:30:00Z"
    }
```

⚠️ `title` and `activeTimestamp` are optional fields. Include them only when necessary.

### 3.3. <a name='NotificationContentValidation'></a>✅ **Notification Content Validation**

To ensure consistency, a GitHub workflow named `notification-validate` automatically checks that all ConfigMaps in the `system-notifications` folder include the required labels and validates all `notification-content.json` objects against the schema defined in [notification-schema.json](./notification-schema.json).

**When does it run?**

- On any pull request that changes:
  - `notification-schema.json` (schema definition)
  - Any `yaml` file defined under `system-notifications`

**What does it check?**

- Ensures the JSON structure matches the schema (e.g., required fields, allowed values).
- Ensure the configmaps are labelled well.
- Prevents invalid or misconfigured notifications from being merged.

**How to fix validation errors?**

- Review the error message in the PR checks.
- Compare your changes with the [schema](./notification-schema.json) and your labels.

### 3.4. <a name='FolderStructure'></a>✅ **Folder Structure**

Notifications are organized under the `system-notifications/` directory:

```bash

system-notifications/
.
├── notification-1.yaml  # A ConfigMap representing one notification
├── notification-2.yaml
└── kustomization.yaml   # Auto-generated, includes all notifications YAMLs

```

When there are no active notifications, both the `system-notifications/` folder and its reference in the parent `kustomization.yaml` should be removed automatically to avoid kustomize errors.

### 3.5. <a name='UIBehavior-1'></a>✅ **UI Behavior**

- The UI discovers and filters notifications by detecting ConfigMaps labeled with
  `konflux.system.notification: "true".`
- When a valid notification ConfigMap exists, its notification will be shown in the UI only if the active time has been reached.
- The UI respects the `activeTimestamp` field to control when notifications are displayed:
  - If `activeTimestamp` is set to a future time, the notification remains hidden until that time arrives.
  - If `activeTimestamp` is not set, the notification uses the resource's `creationTimestamp` to determine when to show.
- To remove a notification from the UI, the corresponding ConfigMap must be deleted or renamed so it no longer matches the label.
- When multiple notifications exist, the UI lists them ordered by their active time, <strong>showing the most recent first</strong>.
- The `type` field control the notification icon shown before the title in the Notification Drawer.
- If the `title` field is omitted in the JSON, it falls back to using component.metadata.name as the default in the UI.
- We leveraged [PatternFly Notification Drawer (v5)](https://v5-archive.patternfly.org/components/notification-drawer/html/#ws-core-c-notification-drawer-basic) and [Notification Badge (v5)](https://v5-archive.patternfly.org/components/notification-badge) components to implement the UI, following their design principles for consistency and usability.
- All notifications are always shown as unread. There is no backend tracking for notification state, so <strong>`read/unread` functionality is not supported</strong>.

### 3.6. <a name='WhentoAddorRemoveNotifications'></a>✅ **When to Add or Remove Notifications**

<strong>These notification ConfigMaps are automatically generated or removed</strong> by monitoring or scripting systems based on current system status.

- Add a notification:

  1. Generate a new ConfigMap YAML file under `system-notifications/`.
  2. Refresh the `kustomization.yaml` in that folder to include the new file.
  3. If the folder does not exist (e.g., no prior notifications), create it and its `kustomization.yaml`, and ensure the parent kustomization includes it.

- Remove a notification:

  1. Delete the corresponding ConfigMap YAML file from `system-notifications/`.
  2. Refresh `kustomization.yaml` to remove the reference.
  3. If no notifications remain, delete the `system-notifications/` directory and remove its reference from the parent kustomization.

⚠️ <strong>These add/remove operations are expected to be automated</strong>. Manual edits should only be done in emergencies or for debugging.
