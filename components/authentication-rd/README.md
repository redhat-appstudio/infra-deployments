# Authentication

The Authentication component contains GitOps manifests for Konflux cluster **authentication and authorization**: OpenShift RBAC (**ClusterRoles**, **ClusterRoleBindings**) for Konflux LDAP/Rover groups, baseline view permissions for authenticated users, and the **admin-checker** job that audits cluster admin group membership.

## What gets deployed

### RBAC roles and bindings

| Manifest | Kind | Purpose |
| --- | --- | --- |
| `konflux-admins.yaml` | `ClusterRole` / `ClusterRoleBinding` | Broad cluster permissions **without** `secrets` or `internalrequests` |
| `konflux-admins-pod-admin.yaml` | `ClusterRole` + `RoleBinding` | Pod create/exec/attach in selected namespaces only |
| `konflux-sre.yaml` | `ClusterRole` / `ClusterRoleBinding` | Read-only pod access; delete allowed for cleanup |
| `component-maintainer.yaml` | `ClusterRole`  | OLM `installplans`, Tekton Results, limited SA patch |
| `grafana-view-only.yaml` | `ClusterRole` / `RoleBinding` | Read `appstudio-grafana` namespace |
| `test-platform-ci-admins-can-view.yaml` | `ClusterRole` / `ClusterRoleBinding` | View test platform / Crossplane resources |
| `everyone-can-view.yaml` + patch | `ClusterRole` / `ClusterRoleBinding` | Shared view roles for App Studio, monitoring, cluster version, compute |

`everyone-can-view-patch.yaml` centralizes the list of Konflux Rover groups that receive the “everyone can view” bindings so the same group list is not duplicated across multiple bindings.

### Admin checker

Runs in namespace `admin-checker` as CronJob `check-cluster-admins` (Mondays 11:30 UTC). It:

1. Reads OpenShift `Group` objects `cluster-admins` and `dedicated-admins`.
2. POSTs membership to the workflow URL in secret `rhtap-infra-secrets` (`admin-checker-workflow-url`).

The service account has read-only access to `user.openshift.io/groups`. Credentials are synced from Vault via External Secrets (`appsre-stonesoup-vault` ClusterSecretStore).

## Related documentation

- [Extending the service — Authentication](../../docs/deployment/extending-the-service.md#authentication)
- [k8s-groups component](https://github.com/redhat-appstudio/internal-infra-deployments/tree/main/components/k8s-groups)

