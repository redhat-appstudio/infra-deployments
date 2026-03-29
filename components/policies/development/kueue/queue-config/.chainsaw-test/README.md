# Test: `kueue-bootstrap-queue-new-tenant-labeled-namespace`

Tests that a LocalQueue is created in a namespace labeled with
`konflux-ci.dev/type=tenant`.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-tenant-labeled-namespace-is-created](#step-when-tenant-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

Apply the queue bootstrap ClusterPolicy and assert Kyverno reports the policy as ready before exercising generation.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-tenant-labeled-namespace-is-created`

Create a namespace labeled konflux-ci.dev/type=tenant so the policy should generate a LocalQueue.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-unlabeled-namespace-negative`

Tests that a LocalQueue is NOT created in an unlabeled namespace
that does not match the special names.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-unlabeled-namespace-is-created](#step-when-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-not-created](#step-then-localqueue-is-not-created) | 0 | 3 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

Apply the queue bootstrap ClusterPolicy and assert Kyverno reports the policy as ready before exercising generation.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-unlabeled-namespace-is-created`

Create a namespace without the tenant label and without a special name so generation must not run.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-not-created`

Confirm pipelines-queue LocalQueue is absent (NotFound) and stays absent after a short wait.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `script` | 0 | 0 | *No description* |
| 2 | `sleep` | 0 | 0 | *No description* |
| 3 | `script` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-label-added-to-namespace`

Tests that a LocalQueue is created when an unlabeled namespace is
updated to have the label `konflux-ci.dev/type=tenant`.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "updated-labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [given-unlabeled-namespace-exists](#step-given-unlabeled-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 5 | [when-tenant-label-is-added](#step-when-tenant-label-is-added) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

Apply the queue bootstrap ClusterPolicy and assert Kyverno reports the policy as ready before exercising generation.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `given-unlabeled-namespace-exists`

Create an unlabeled namespace first so a later label update can trigger the policy on change.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-tenant-label-is-added`

Apply the tenant-labeled namespace manifest so konflux-ci.dev/type=tenant is set on the existing namespace.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-labeled-namespace-before-policy`

Tests that a LocalQueue is created for an existing tenant-labeled
namespace when the ClusterPolicy is applied (generateExisting).


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-tenant-labeled-namespace-exists](#step-given-tenant-labeled-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-tenant-labeled-namespace-exists`

Create the tenant-labeled namespace before the ClusterPolicy exists to exercise generateExisting on policy install.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policy-is-ready`

Apply the ClusterPolicy after the target namespace exists so generateExisting can backfill the LocalQueue.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-mintmaker-namespace-before-policy`

Tests that a LocalQueue is created for an existing `mintmaker`
namespace when the ClusterPolicy is applied (generateExisting,
name-based match).


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-mintmaker-namespace-exists](#step-given-mintmaker-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-mintmaker-namespace-exists`

Create the mintmaker namespace before policy install so name-based matching is satisfied for generateExisting.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policy-is-ready`

Apply the ClusterPolicy after the target namespace exists so generateExisting can backfill the LocalQueue.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-kanary-namespace-before-policy`

Tests that a LocalQueue is created for an existing
`appstudio-kanary-exporter` namespace when the ClusterPolicy is
applied (generateExisting, name-based match).


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kanary-namespace-exists](#step-given-kanary-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kanary-namespace-exists`

Create the appstudio-kanary-exporter namespace before policy install for name-based generateExisting.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policy-is-ready`

Apply the ClusterPolicy after the target namespace exists so generateExisting can backfill the LocalQueue.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-mintmaker-by-name`

Tests that a LocalQueue is created for the `mintmaker` namespace
without the tenant label (name-based match).


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-mintmaker-namespace-is-created](#step-when-mintmaker-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

Apply the queue bootstrap ClusterPolicy and assert Kyverno reports the policy as ready before exercising generation.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-mintmaker-namespace-is-created`

Create the mintmaker namespace after the policy is ready to verify name-based generation without the tenant label.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `kueue-bootstrap-queue-kanary-by-name`

Tests that a LocalQueue is created for the `appstudio-kanary-exporter`
namespace without the tenant label (name-based match).


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-localqueue-crd-exists](#step-given-localqueue-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-kanary-namespace-is-created](#step-when-kanary-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-localqueue-is-created](#step-then-localqueue-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-localqueue-crd-exists`

Install the Kueue LocalQueue CRD so the API server accepts LocalQueue objects in the test cluster.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

Apply Kyverno RBAC so the policy engine can generate and reconcile LocalQueue resources.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

Apply the queue bootstrap ClusterPolicy and assert Kyverno reports the policy as ready before exercising generation.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-kanary-namespace-is-created`

Create the appstudio-kanary-exporter namespace after the policy is ready to verify name-based generation.


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-localqueue-is-created`

Assert the expected pipelines-queue LocalQueue exists and matches the fixture (name and spec).


#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

