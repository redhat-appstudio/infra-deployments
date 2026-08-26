# Test: `in-rhtap-releng-tenant-invalid-rolebinding`

tests that the a invalid RoleBinding can NOT be created
in a tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 3 | [given-tenant-namespace-exists](#step-given-tenant-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-invalid-rolebindings-can-not-be-created](#step-then-invalid-rolebindings-can-not-be-created) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `given-tenant-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-invalid-rolebindings-can-not-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

---

# Test: `in-rhtap-releng-tenant-valid-rolebinding`

tests that the a valid RoleBinding can be created in a
tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 3 | [given-tenant-namespace-exists](#step-given-tenant-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-valid-rolebindings-can-be-created](#step-then-valid-rolebindings-can-be-created) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `given-tenant-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-valid-rolebindings-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

---

# Test: `out-of-rhtap-releng-tenant-rolebinding`

tests that the whatever RoleBinding can be created in a
non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 3 | [then-rolebindings-can-be-created-in-a-nontenant-namespace](#step-then-rolebindings-can-be-created-in-a-nontenant-namespace) | 0 | 2 | 0 | 0 | 0 |

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

### Step: `then-rolebindings-can-be-created-in-a-nontenant-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |

---

