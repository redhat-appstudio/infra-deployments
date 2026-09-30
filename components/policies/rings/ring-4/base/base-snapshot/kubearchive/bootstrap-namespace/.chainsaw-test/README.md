# Test: `mutate-new-namespace-konfluxcidev`

tests that the KubeArchiveConfig is created in a namespace
labelled with `konflux-ci.dev/type=tenant`


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konfluxcidev" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kubearchiveconfig-crd-exists](#step-given-kubearchiveconfig-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-konfluxcidev-labeled-namespace-is-created](#step-when-konfluxcidev-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-kubearchiveconfig-is-created](#step-then-kubearchiveconfig-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kubearchiveconfig-crd-exists`

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

### Step: `then-kubearchiveconfig-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `mutate-new-namespace-unlabeled`

tests that the KubeArchiveConfig is NOT created in an unlabeled namespace


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kubearchiveconfig-crd-exists](#step-given-kubearchiveconfig-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 4 | [when-unlabeled-namespace-is-created](#step-when-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-kubearchiveconfig-is-created](#step-then-kubearchiveconfig-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kubearchiveconfig-crd-exists`

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

### Step: `then-kubearchiveconfig-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-konfluxcidev`

tests that the KubeArchiveConfig is created in an already existing
namespace labelled with `konflux-ci.dev/type=tenant`


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konflux" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kubearchiveconfig-crd-exists](#step-given-kubearchiveconfig-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-konfluxci-labeled-namespace-is-created](#step-given-konfluxci-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-kubearchiveconfig-is-created](#step-then-kubearchiveconfig-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kubearchiveconfig-crd-exists`

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

### Step: `then-kubearchiveconfig-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-unlabeled`

tests that the KubeArchiveConfig is NOT created in an
existing unlabeled namespace


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "unlabeled" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kubearchiveconfig-crd-exists](#step-given-kubearchiveconfig-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-unlabeled-namespace-is-created](#step-given-unlabeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-kubearchiveconfig-is-created](#step-then-kubearchiveconfig-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kubearchiveconfig-crd-exists`

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

### Step: `then-kubearchiveconfig-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

---

# Test: `mutate-existing-namespace-konfluxcidev-existing-kubearchiveconfig`

tests that the KubeArchiveConfig is not updated in an already existing
namespace labelled with `konflux-ci.dev/type=tenant` where the
KubeArchiveConfig already exists


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konfluxcidev-existing-kubearchive" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kubearchiveconfig-crd-exists](#step-given-kubearchiveconfig-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-konfluxcidev-labeled-namespace-is-created](#step-given-konfluxcidev-labeled-namespace-is-created) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-kubearchiveconfig-is-created](#step-given-kubearchiveconfig-is-created) | 0 | 1 | 0 | 0 | 0 |
| 5 | [when-cluster-policy-is-ready](#step-when-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 6 | [then-kubearchiveconfig-is-not-changed](#step-then-kubearchiveconfig-is-not-changed) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kubearchiveconfig-crd-exists`

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

### Step: `given-konfluxcidev-labeled-namespace-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kubearchiveconfig-is-created`

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

### Step: `then-kubearchiveconfig-is-not-changed`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

