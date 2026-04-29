# Test: `in-tenant-create-opaque-secret-allowed`

Creates an Opaque Secret in a tenant namespace while the policy is active.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [then-opaque-secret-can-be-created](#step-then-opaque-secret-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `ensure-namespace-is-labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-validatingadmissionpolicy-is-installed`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `then-opaque-secret-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `in-tenant-deny-legacy-serviceaccount-token-secret`

Fails to create a Secret of type kubernetes.io/service-account-token in a tenant namespace.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [given-serviceaccount-exists](#step-given-serviceaccount-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-legacy-sa-token-secret-can-not-be-created](#step-then-legacy-sa-token-secret-can-not-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `ensure-namespace-is-labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-validatingadmissionpolicy-is-installed`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `given-serviceaccount-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `then-legacy-sa-token-secret-can-not-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `error` | 0 | 0 | *No description* |

---

