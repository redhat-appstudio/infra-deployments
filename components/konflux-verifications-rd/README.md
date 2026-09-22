# Verification environments

This component provisions persistent namespaces and access for team-owned test
suites. Kargo launches verification through Tekton on the target cluster; the test
namespaces survive individual runs.

A **suite** is the ownership and concurrency boundary. A team can own several
suites. Each suite has one runner namespace and one or more target namespaces.
Different suites must not share mutable test resources if they run concurrently.

## Repository layout

```text
base/
  profiles/
    runner/                   # Namespace, launcher and runner SAs, launcher RBAC
    target/                   # Persistent tenant namespace; no test privileges
  suites/
    konflux-vanguard/conformance/      # Existing conformance resources, unchanged
examples/team-suite/          # Buildable onboarding example; not deployed
rings/
  ring-0/base/                # Empty: no development deployment
  ring-1/base/base-snapshot/  # Self-contained snapshot of base
  ring-1/<cluster>/           # Cluster overlay selecting the snapshot
```

`base/kustomization.yaml` explicitly selects enabled suites. The profiles and
example are opt-in; adding a directory alone does not deploy it. Ring snapshots
include their profiles so changes to `base` cannot bypass promotion into a ring.

## Existing conformance suite

| Resource | Purpose |
| --- | --- |
| `konflux-managed-tests` | Runner namespace |
| `konflux-conformance-tests` | Additional test namespace |
| `konflux-managed-tests/konflux-bot-0` | Remote launcher identity, bound to `konflux-builder-bot-actions` |
| `konflux-managed-tests/conformance-test-runner` | Test execution identity with suite-specific bindings |
| `konflux-managed-tests/release-pipeline` | Existing release execution identity |

Both namespaces have the `konflux-ci.dev/type: tenant` label. The word "managed"
in the namespace name does not give it a different namespace type. This
reorganization preserves all existing names, annotations, permissions and targets.
Conformance keeps its legacy RBAC; new suites use the smaller runner profile.

The staging ApplicationSet selects `stone-stage-p01`, `stone-stg-rh01` and
`lightwell-dev`. Other selected clusters use the empty fallback. These manifests
do not add production targets. See the
[ApplicationSet](../../argo-cd-apps/overlays/rd-staging/konflux-verifications-rd/konflux-verifications-rd-appset.yaml)
and [staging cluster list](../../argo-cd-apps/k-components/deploy-to-staging-tenant-clusters/tenant-clusters-list-patch.yaml).

## Add a suite

1. Copy `examples/team-suite/` into `base/suites/<team>/<suite>/` and add an
   `OWNERS` file listing the team's actual approvers and reviewers.
2. Set unique runner and target namespace names. Update the explicit namespaces
   in `target-access.yaml`, including the runner ServiceAccount's subject
   namespace. Set the team and suite labels and the required billing labels.
3. Adjust the profile references for the deeper directory:
   - `runner/kustomization.yaml`: `../../../../profiles/runner`
   - `targets/tenant/kustomization.yaml`: `../../../../../profiles/target`
4. Add target namespaces by creating more target overlays. Keep the suite root
   free of a `namespace:` transformer: it would collapse the namespace boundary
   and can rewrite cross-namespace ServiceAccount subjects.
5. Replace the example pod-read Role with the suite's reviewed permissions.
   Add the team's Pipeline/Task resources and required Secrets or ExternalSecrets
   under the runner overlay. Pipelines must use the `verification-runner`
   ServiceAccount. Use immutable test image/source versions.
6. Review quotas, network access and any tenant policies that add permissions or
   resources automatically. The profiles do not impose workload-specific quotas
   or network policy. Cluster-wide test access requires separate review.
7. Add `suites/<team>/<suite>` to `base/kustomization.yaml`. Copy the complete base,
   including `profiles/`, into the intended ring's `base-snapshot` through the
   normal promotion/review flow. All cluster overlays referencing that snapshot
   receive its enabled suites; split the snapshot/overlay selection if targeting
   only a subset of those clusters.
8. Configure the corresponding Kargo AnalysisTemplate and credential delivery in
   `infra-common-deployments`. Namespace provisioning alone does not enable tests.

The launcher Role permits PipelineRun submission, observation, cancellation and
deletion, plus TaskRun/pod/log reads within its runner namespace. It grants no
Secret reads or target-namespace permissions. Test permissions belong in explicit
RoleBindings from the runner ServiceAccount into the suite's target namespaces.
For additional operations inside the runner namespace, add explicit suite RBAC.

Creating arbitrary PipelineRuns can indirectly exercise the runner's permissions
and access its Secrets. These identities are separate operational roles, not a
security boundary against an untrusted submitter. Limit launch access to trusted
callers; enforce approved pipelines and ServiceAccounts with admission policy
before granting it to less-trusted callers.

## Credentials and persistent state

Launcher tokens are minted through Kubernetes TokenRequest and delivered to Kargo
through the existing credential system. This component does **not** mint tokens,
create bound token Secrets, or publish them to Vault. Provisioning and rotation
need an explicit owner; an ExternalSecret refresh does not renew a token.

For new suites, keep test credentials in the remote runner namespace and use
`valueFrom.secretKeyRef` in test steps. Do not insert token values into PipelineRun
specifications. Give each authorized Kargo Project only its suite/cluster launcher
credentials. Existing conformance credential delivery is unchanged by this PR.

Namespaces use `Delete=false,Prune=false` because tests reuse them. Run cleanup
must remove only run-owned resources and preserve infrastructure and fixtures.
Removing a suite from Git does not delete its protected namespaces; retirement
requires deliberate cleanup. New profile RBAC remains prunable so obsolete access
can be removed normally.

## Execution contract for the launcher follow-up

The current conformance launcher is in `infra-common-deployments`. This foundation
neither replaces it nor implements serialization. The shared launcher must enforce:

- One active execution per `(cluster, suite)` across all Kargo stages and callers.
- A durable run identity, recovery of existing runs, and bounded queue/execution
  deadlines. A retry must not accidentally create a second execution.
- Coordination covering both tests and cleanup. Lease expiry alone must never
  authorize overlapping runs; ambiguous recovery must block and report why.
- Cancellation/timeout handling for remote runs and retained failure diagnostics.
- Recorded suite, cluster, verification identity, tested revision and test version.

Add coordination permissions with that implementation, once the lock/state
protocol is defined. Do not assume this namespace profile already enforces it.

## Validate

From the repository root:

```sh
kustomize build components/konflux-verifications-rd/base
kustomize build components/konflux-verifications-rd/examples/team-suite
kustomize build components/konflux-verifications-rd/rings/ring-1/stone-stage-p01
kustomize build components/konflux-verifications-rd/rings/ring-1/stone-stg-rh01
kustomize build components/konflux-verifications-rd/rings/ring-1/lightwell-dev
```

Before enabling a suite, inspect the rendered namespaces, RoleBindings and runner
identities together. The example intentionally grants only pod-read access and
does not include an executable test pipeline.
