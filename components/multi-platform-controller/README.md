# Multi-Platform Controller Host Configuration

This directory contains the configuration for the multi-platform-controller, which manages dynamic and static hosts across different environments and clusters.

## Overview

The multi-platform-controller uses a Helm chart to generate host configuration from values files specific to each cluster. This approach provides:
- Centralized template management in `base/host-config-chart/`
- Environment-specific configuration through values files
- Consistent configuration structure across all clusters
- Easy AMI updates and configuration changes

## Directory Structure

```
components/multi-platform-controller/
├── base/
│   ├── host-config-chart/          # Helm chart for generating host configs
│   │   ├── Chart.yaml              # Chart metadata
│   │   └── templates/
│   │       └── host-config.yaml    # ConfigMap template
│   └── ...
├── production/
│   ├── stone-prd-rh01/
│   │   ├── host-values.yaml        # Values file for this cluster
│   │   └── kustomization.yaml
│   ├── kflux-prd-rh02/
│   └── kflux-prd-rh03/
├── staging/
│   ├── host-values.yaml
│   └── kustomization.yaml
└── production-downstream/
    └── ...
```

## Generating Host Configuration

### Basic Command

From within a cluster-specific directory (e.g., `production/stone-prd-rh01/`), run:

```bash
helm template ../../base/host-config-chart/ \
  --namespace multi-platform-controller \
  -f host-values.yaml > out.yaml
```

### Command Breakdown

- `helm template` - Renders the chart locally without installing to a cluster
- `../../base/host-config-chart/` - Path to the Helm chart directory
- `--namespace multi-platform-controller` - Sets the namespace for the generated resources
- `-f host-values.yaml` - Specifies the values file for this cluster
- `> out.yaml` - Redirects output to a file for review

