# Test: `outside-tenant-create-valid-taskrun`

creates a TaskRun with a PLATFORM parameter and a corresponding
tekton-kueue's annotation in a non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [then-valid-taskrun-can-be-created](#step-then-valid-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

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

# Test: `outside-tenant-create-valid-taskrun-complex-platform`

creates a TaskRun with a complex PLATFORM parameter and a corresponding
tekton-kueue's annotation in a non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [then-valid-taskrun-can-be-created](#step-then-valid-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

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

# Test: `outside-tenant-create-skipped-taskrun`

creates a TaskRun with no PLATFORM parameter and no corresponding
tekton-kueue's annotation in a non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [then-skipped-taskrun-can-be-created](#step-then-skipped-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

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

# Test: `outside-tenant-create-invalid-wrong-annotation-taskrun`

creates a TaskRun with a PLATFORM parameter and
a wrong corresponding tekton-kueue's annotation
in a non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [then-invalid-taskrun-can-be-created](#step-then-invalid-taskrun-can-be-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

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

### Step: `then-invalid-taskrun-can-be-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `outside-tenant-create-invalid-no-annotations-taskrun`

creates a TaskRun with a PLATFORM parameter and no
corresponding tekton-kueue's annotation in a
non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-taskruns-crd-exists](#step-given-taskruns-crd-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-validatingadmissionpolicy-is-installed](#step-given-validatingadmissionpolicy-is-installed) | 0 | 3 | 0 | 0 | 0 |
| 3 | [then-invalid-taskrun-is-created](#step-then-invalid-taskrun-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-taskruns-crd-exists`

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

### Step: `then-invalid-taskrun-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

