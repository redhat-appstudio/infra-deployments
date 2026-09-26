# Test: `in-tenant-create-valid-taskrun`

creates a TaskRun with a PLATFORM parameter and a corresponding
tekton-kueue's annotation


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 4 | [then-valid-taskrun-can-be-created](#step-then-valid-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

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

### Step: `then-valid-taskrun-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `in-tenant-create-valid-taskrun-complex-platform`

creates a TaskRun with a complex PLATFORM parameter and a corresponding
tekton-kueue's annotation


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 4 | [then-valid-taskrun-can-be-created](#step-then-valid-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

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

### Step: `then-valid-taskrun-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `in-tenant-create-skipped-taskrun`

creates a TaskRun with no PLATFORM parameter and no corresponding
tekton-kueue's annotation


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 4 | [then-skipped-taskrun-can-be-created](#step-then-skipped-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

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

### Step: `then-skipped-taskrun-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `in-tenant-create-invalid-wrong-annotation-taskrun`

fails to create a TaskRun with a PLATFORM parameter and
a wrong corresponding tekton-kueue's annotation


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 4 | [then-ivalid-taskrun-can-not-be-created](#step-then-ivalid-taskrun-can-not-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

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

### Step: `then-ivalid-taskrun-can-not-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `error` | 0 | 0 | *No description* |

---

# Test: `in-tenant-create-invalid-no-annotations-taskrun`

fails to create a TaskRun with a PLATFORM parameter
and no corresponding tekton-kueue's annotation


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [ensure-namespace-is-labeled](#step-ensure-namespace-is-labeled) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 4 | [then-invalid-taskrun-can-not-be-created](#step-then-invalid-taskrun-can-not-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

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

### Step: `then-invalid-taskrun-can-not-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `error` | 0 | 0 | *No description* |

---

