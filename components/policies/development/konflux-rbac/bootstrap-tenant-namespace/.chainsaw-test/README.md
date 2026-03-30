# Test: `konfluxci-bootstrap-ns-mutate-new-namespace-konfluxcidev`

tests that the resources are created in a namespace
labeled with `konflux-ci.dev/type=tenant`


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-appstudio-pipeline-clusterrole-exists](#step-given-appstudio-pipeline-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policies-are-ready](#step-given-cluster-policies-are-ready) | 0 | 7 | 0 | 0 | 0 |
| 4 | [when-konfluxcidev-labeled-namespace-is-created](#step-when-konfluxcidev-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-resources-are-created](#step-then-resources-are-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-appstudio-pipeline-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policies-are-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `apply` | 0 | 0 | *No description* |
| 4 | `apply` | 0 | 0 | *No description* |
| 5 | `apply` | 0 | 0 | *No description* |
| 6 | `apply` | 0 | 0 | *No description* |
| 7 | `assert` | 0 | 0 | *No description* |

### Step: `when-konfluxcidev-labeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-resources-are-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `konfluxci-bootstrap-ns-mutate-updated-namespace-konfluxcidev`

tests that the resources are created in an unlabeled namespace
when it is updated to have the label `konflux-ci.dev/type=tenant`


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "updated-labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-appstudio-pipeline-clusterrole-exists](#step-given-appstudio-pipeline-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policies-are-ready](#step-given-cluster-policies-are-ready) | 0 | 7 | 0 | 0 | 0 |
| 4 | [given-unlabeled-namespace-exists](#step-given-unlabeled-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 5 | [when-konfluxcidev-label-is-added](#step-when-konfluxcidev-label-is-added) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-resources-are-created](#step-then-resources-are-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-appstudio-pipeline-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policies-are-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `apply` | 0 | 0 | *No description* |
| 4 | `apply` | 0 | 0 | *No description* |
| 5 | `apply` | 0 | 0 | *No description* |
| 6 | `apply` | 0 | 0 | *No description* |
| 7 | `assert` | 0 | 0 | *No description* |

### Step: `given-unlabeled-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-konfluxcidev-label-is-added`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-resources-are-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `konfluxci-bootstrap-ns-mutate-new-namespace-unlabeled`

tests that the resources are NOT created in an unlabeled namespace


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-appstudio-pipeline-clusterrole-exists](#step-given-appstudio-pipeline-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policies-are-ready](#step-given-cluster-policies-are-ready) | 0 | 7 | 0 | 0 | 0 |
| 4 | [when-unlabeled-namespace-is-created](#step-when-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-resources-are-not-created](#step-then-resources-are-not-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-appstudio-pipeline-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policies-are-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `apply` | 0 | 0 | *No description* |
| 4 | `apply` | 0 | 0 | *No description* |
| 5 | `apply` | 0 | 0 | *No description* |
| 6 | `apply` | 0 | 0 | *No description* |
| 7 | `assert` | 0 | 0 | *No description* |

### Step: `when-unlabeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-resources-are-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

# Test: `konfluxci-bootstrap-ns-mutate-existing-namespace`

tests that the RoleBinding is NOT created in an already existing
namespace labeled with `konflux-ci.dev/type=tenant`, whereas
other resources are.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konfluxci-labeled-namespace-is-created](#step-given-konfluxci-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-appstudio-pipeline-clusterrole-exists](#step-given-appstudio-pipeline-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policies-are-ready](#step-when-cluster-policies-are-ready) | 0 | 7 | 0 | 0 | 0 |
| 5 | [then-resources-are-created](#step-then-resources-are-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konfluxci-labeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-appstudio-pipeline-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policies-are-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `apply` | 0 | 0 | *No description* |
| 4 | `apply` | 0 | 0 | *No description* |
| 5 | `apply` | 0 | 0 | *No description* |
| 6 | `apply` | 0 | 0 | *No description* |
| 7 | `assert` | 0 | 0 | *No description* |

### Step: `then-resources-are-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `konfluxci-bootstrap-ns-mutate-existing-namespace-unlabeled`

tests that the RoleBinding is NOT created in an
existing unlabeled namespace


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-appstudio-pipeline-clusterrole-exists](#step-given-appstudio-pipeline-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-unlabeled-namespace-is-created](#step-given-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policies-are-ready](#step-when-cluster-policies-are-ready) | 0 | 7 | 0 | 0 | 0 |
| 5 | [then-resources-are-not-created](#step-then-resources-are-not-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-appstudio-pipeline-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-unlabeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policies-are-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `apply` | 0 | 0 | *No description* |
| 4 | `apply` | 0 | 0 | *No description* |
| 5 | `apply` | 0 | 0 | *No description* |
| 6 | `apply` | 0 | 0 | *No description* |
| 7 | `assert` | 0 | 0 | *No description* |

### Step: `then-resources-are-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

# Test: `konfluxci-bootstrap-ns-mutate-existing-namespace-konfluxcidev-existing-resources`

tests that resources are not updated in an already existing
namespace labeled with `konflux-ci.dev/type=tenant` where the
RoleBinding already exists


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konfluxcidev-labeled-namespace-is-created](#step-given-konfluxcidev-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 2 | [then-resources-are-not-changed](#step-then-resources-are-not-changed) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-appstudio-pipeline-clusterrole-exists](#step-given-appstudio-pipeline-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 5 | [when-cluster-policies-are-ready](#step-when-cluster-policies-are-ready) | 0 | 7 | 0 | 0 | 0 |
| 6 | [then-resources-are-not-changed](#step-then-resources-are-not-changed) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konfluxcidev-labeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-resources-are-not-changed`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-appstudio-pipeline-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policies-are-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `apply` | 0 | 0 | *No description* |
| 4 | `apply` | 0 | 0 | *No description* |
| 5 | `apply` | 0 | 0 | *No description* |
| 6 | `apply` | 0 | 0 | *No description* |
| 7 | `assert` | 0 | 0 | *No description* |

### Step: `then-resources-are-not-changed`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

