# Kueue VM Quotas Management

This directory contains a script to automatically update Kueue ClusterQueue resources with VM quotas based on multi-platform-controller host configurations.

## Overview

The `update-kueue-vm-quotas.py` script reads a `host-config.yaml` file from multi-platform-controller and updates a `cluster-queue.yaml` file with appropriate resource groups and quotas for each platform.

## How It Works

The script extracts platform quotas from host-config.yaml:

1. **Dynamic Platforms**: Uses `max-instances` values from keys like `dynamic.linux-arm64.max-instances`
2. **Static Platforms**: Uses `concurrency` values from keys like `host.s390x-static-1.concurrency`

Platform quotas are distributed across resource groups respecting Kueue's constraints:
- Maximum 16 resource groups total
- Maximum 16 resources per group
- Each resource group has a single unique flavor

## Usage

### Dry Run (Recommended First)

```bash
python3 hack/kueue-vm-quotas/update-kueue-vm-quotas.py \
    components/multi-platform-controller/staging/host-config.yaml \
    components/kueue/development/queue-config/cluster-queue.yaml \
    --dry-run
```

### Update the Cluster Queue

```bash
python3 hack/kueue-vm-quotas/update-kueue-vm-quotas.py \
    components/multi-platform-controller/staging/host-config.yaml \
    components/kueue/development/queue-config/cluster-queue.yaml
```

## Example Output

The script shows platform discovery and quota assignment:

```
Found 34 platforms:
  linux-arm64: 160
  linux-c2xlarge-amd64: 10
  linux-c4xlarge-arm64: 10
  linux-mlarge-arm64: 160
  linux-ppc64le: 64
  linux-s390x: 60
  ...
```

During updates, it shows resource group creation:

```
Preserved base resource group with basic resources
Created resource group 1 with 16 platforms: linux-arm64, linux-c2xlarge-amd64, ...
Created resource group 2 with 16 platforms: linux-m2xlarge-arm64, linux-m4xlarge-amd64, ...
Created resource group 3 with 2 platforms: linux-s390x, linux-ppc64le

Updated cluster-queue.yaml
Total resource groups: 4
Base resource groups: 1, Platform resource groups: 3
Total platforms processed: 34
```

## Generated Structure

### Base Resource Group (Preserved)
Contains fundamental resources - preserved from existing configuration:

```yaml
resourceGroups:
  - coveredResources:
      - tekton.dev/pipelineruns
      - cpu
      - memory
    flavors:
      - name: default-flavor
        resources:
          - name: tekton.dev/pipelineruns
            nominalQuota: '500'
          - name: cpu
            nominalQuota: 1k
          - name: memory
            nominalQuota: 500Ti
```

### Platform Resource Groups
Each group contains up to 16 platforms with unique flavor names:

```yaml
  - coveredResources:
      - linux-arm64
      - linux-c2xlarge-amd64
      - linux-c4xlarge-arm64
      # ... up to 16 platforms
    flavors:
      - name: platform-group-1
        resources:
          - name: linux-arm64
            nominalQuota: '160'
          - name: linux-c2xlarge-amd64
            nominalQuota: '10'
          - name: linux-c4xlarge-arm64
            nominalQuota: '10'
          # ... individual platform quotas
```

## Platform Naming Convention

- **Dynamic platforms**: Use platform name as-is (e.g., `linux-arm64`)
- **Static platforms**: Convert from `linux/s390x` format to `linux-s390x`
- **Multiple hosts**: Quotas are aggregated (e.g., multiple s390x hosts sum their concurrency)

## Development Workflow

When updating multi-platform-controller host configurations:

1. **Review changes**: Run with `--dry-run` to preview updates
2. **Validate quotas**: Ensure platform quotas are reasonable
3. **Update cluster queue**: Run without `--dry-run` to apply changes
4. **Commit together**: Include both host-config.yaml and cluster-queue.yaml changes

## Requirements

- Python 3.6+
- PyYAML (`pip install PyYAML`)

## Files

- **Input**: `components/multi-platform-controller/*/host-config.yaml`
- **Output**: `components/kueue/*/queue-config/cluster-queue.yaml`
- **Generated**: ResourceFlavor objects for each platform group
