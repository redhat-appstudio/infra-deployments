# Test: `outside-tenant-create-legacy-serviceaccount-token-secret-allowed`

Creates a Secret of type kubernetes.io/service-account-token in a namespace that is not
labeled as a tenant; the binding must not apply.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 2 | [given-serviceaccount-exists](#step-given-serviceaccount-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [then-legacy-sa-token-secret-can-be-created](#step-then-legacy-sa-token-secret-can-be-created) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `then-legacy-sa-token-secret-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

