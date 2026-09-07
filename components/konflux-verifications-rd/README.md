# konflux-verifications-rd

Infrastructure component for Kargo verification conformance tests.

## What this deploys

- **Namespaces**: `konflux-conformance-tests` (tenant) and `konflux-managed-tests` (managed)
- **ServiceAccount**: `konflux-bot-0` in `konflux-managed-tests`, with a bound token Secret
- **RoleBindings**: Grants the SA `konflux-admin-user-actions` in both namespaces and `release-pipeline-resource-role` in the managed namespace

## Purpose

Kargo uses this ServiceAccount to run conformance tests against staging clusters as a promotion verification gate. The SA token is pushed to Vault via PushSecret and consumed by Kargo clusters through ExternalSecrets.

## Cluster targeting

Currently deployed to `stone-stage-p01` only. Other staging clusters use the `empty-base` fallback (no resources deployed). To add a new cluster, create a directory under `staging/<cluster-name>/` with a `kustomization.yaml` pointing to `../../base` and add it to the ApplicationSet generator list.

## Owners

See [OWNERS](OWNERS).
