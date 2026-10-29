# IBM FinOps Agent for Konflux

This directory contains Kubernetes manifests for deploying the IBM FinOps Agent to Konflux clusters for cost monitoring and analysis with IBM Cloudability.

## Overview

The **IBM FinOps Agent** is a unified, modern solution that collects comprehensive cluster cost and resource metrics and uploads them to IBM Cloudability. Here are some of it's capabilities:

- Built-in metrics database with configurable retention
- Support for both Cloudability and Kubecost
- Prometheus metrics export
- Advanced authentication (access/secret keys + environment ID)
- Custom upload destinations (S3, Azure Blob)
- Outbound proxy support
- Metric data parsing and filtering


## Generated from Helm Charts

The manifests in this directory were generated from official Helm charts:

### IBM FinOps Agent
- **Chart:** `ibm-finops/finops-agent`
- **Repository:** https://kubecost.github.io/finops-agent-chart


### IBM FinOps Agent Credentials

Managed by: `ibm-finops-agent-secrets` ExternalSecret

Required vault fields:
- `access_key` - Cloudability Access Key ID
- `secret_key` - Cloudability Secret Key  
- `env_id` - Cloudability Environment ID
- `csp_pricing_api_key` - Cloud Service Provider Pricing API Key

## Verification

After deployment:

```bash
# Check namespaces
oc get namespace ibm-finops-agent

# Check secrets created
oc get secret -n ibm-finops-agent ibm-finops-agent-secrets
# Verify deployments
oc get deployment -n ibm-finops-agent

# Check logs
oc logs -n ibm-finops-agent -l app.kubernetes.io/name=finops-agent
```