# Test: `mutate-new-namespace-konfluxcidev`

tests that the ServiceAccount and RoleBinding are created in a namespace
labelled with `konflux-ci.dev/type=tenant`


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konfluxcidev" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konflux-integration-runner-clusterrole-exists](#step-given-konflux-integration-runner-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-konfluxcidev-labeled-namespace-is-created](#step-when-konfluxcidev-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-serviceaccount-is-created](#step-then-serviceaccount-is-created) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-rolebinding-is-created](#step-then-rolebinding-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konflux-integration-runner-clusterrole-exists`

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

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-konfluxcidev-labeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-serviceaccount-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

### Step: `then-rolebinding-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `mutate-new-namespace-unlabeled`

tests that the ServiceAccount and RoleBinding are NOT created in an unlabeled namespace


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konflux-integration-runner-clusterrole-exists](#step-given-konflux-integration-runner-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-unlabeled-namespace-is-created](#step-when-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-serviceaccount-is-not-created](#step-then-serviceaccount-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-rolebinding-is-not-created](#step-then-rolebinding-is-not-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konflux-integration-runner-clusterrole-exists`

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

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-unlabeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-serviceaccount-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `then-rolebinding-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-konfluxcidev`

tests that the ServiceAccount and RoleBinding are created in an already existing
namespace labelled with `konflux-ci.dev/type=tenant`


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konflux" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konflux-integration-runner-clusterrole-exists](#step-given-konflux-integration-runner-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-konfluxci-labeled-namespace-is-created](#step-given-konfluxci-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-serviceaccount-is-created](#step-then-serviceaccount-is-created) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-rolebinding-is-created](#step-then-rolebinding-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konflux-integration-runner-clusterrole-exists`

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

### Step: `given-konfluxci-labeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `when-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-serviceaccount-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

### Step: `then-rolebinding-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-unlabeled`

tests that the ServiceAccount and RoleBinding are NOT created in an
existing unlabeled namespace


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konflux-integration-runner-clusterrole-exists](#step-given-konflux-integration-runner-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-unlabeled-namespace-is-created](#step-given-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-serviceaccount-is-not-created](#step-then-serviceaccount-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-rolebinding-is-not-created](#step-then-rolebinding-is-not-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konflux-integration-runner-clusterrole-exists`

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

### Step: `when-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-serviceaccount-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `then-rolebinding-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-unlabeled-to-labeled`

tests that the ServiceAccount and RoleBinding are created in an
existing unlabeled namespace when it is labeled


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "to-labeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konflux-integration-runner-clusterrole-exists](#step-given-konflux-integration-runner-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-unlabeled-namespace-is-created](#step-given-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [given-serviceaccount-is-not-created](#step-given-serviceaccount-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 6 | [given-rolebinding-is-not-created](#step-given-rolebinding-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 7 | [when-konfluxci-namespace-is-labeled-namespace](#step-when-konfluxci-namespace-is-labeled-namespace) | 0 | 1 | 0 | 0 | 0 |
| 8 | [then-serviceaccount-is-created](#step-then-serviceaccount-is-created) | 0 | 1 | 0 | 0 | 0 |
| 9 | [then-rolebinding-is-created](#step-then-rolebinding-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konflux-integration-runner-clusterrole-exists`

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

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `given-serviceaccount-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `given-rolebinding-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `when-konfluxci-namespace-is-labeled-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-serviceaccount-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

### Step: `then-rolebinding-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-unlabeled-to-unlabeled`

tests that the ServiceAccount and RoleBinding are not created in an
existing unlabeled namespace when it is updated but still found unlabeled


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "to-unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-konflux-integration-runner-clusterrole-exists](#step-given-konflux-integration-runner-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-unlabeled-namespace-is-created](#step-given-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [given-serviceaccount-is-not-created](#step-given-serviceaccount-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 6 | [given-rolebinding-is-not-created](#step-given-rolebinding-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 7 | [when-konfluxci-namespace-is-updated-to-unlabeled-namespace](#step-when-konfluxci-namespace-is-updated-to-unlabeled-namespace) | 0 | 1 | 0 | 0 | 0 |
| 8 | [then-serviceaccount-is-not-created](#step-then-serviceaccount-is-not-created) | 0 | 1 | 0 | 0 | 0 |
| 9 | [then-rolebinding-is-not-created](#step-then-rolebinding-is-not-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-konflux-integration-runner-clusterrole-exists`

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

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `given-serviceaccount-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `given-rolebinding-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `when-konfluxci-namespace-is-updated-to-unlabeled-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-serviceaccount-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `then-rolebinding-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

