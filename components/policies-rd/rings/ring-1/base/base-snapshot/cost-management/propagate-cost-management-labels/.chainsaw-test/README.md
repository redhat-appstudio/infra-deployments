# Test: `label-propagation-valid-cost-center`

tests that the labels are correctly set on pods in tenant namespace
that have the `cost-center` label


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply GlobalContextEntry](#step-Apply GlobalContextEntry) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Create namespaces for testing](#step-Create namespaces for testing) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 4 | [Apply kyverno Cluster Policy and assert it exists](#step-Apply kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 5 | [create pods in tenant](#step-create pods in tenant) | 0 | 1 | 0 | 0 | 0 |
| 6 | [assert pods in the tenant are labeled](#step-assert pods in the tenant are labeled) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply GlobalContextEntry`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Create namespaces for testing`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 2 | 0 | *No description* |

### Step: `Apply RBAC`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Apply kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 1 | 0 | *No description* |

### Step: `create pods in tenant`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `assert pods in the tenant are labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 2 | 0 | *No description* |

---

# Test: `label-not-applied-random-ns`

tests that the label is not applied to pods in a non-tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply GlobalContextEntry](#step-Apply GlobalContextEntry) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Create namespaces for testing](#step-Create namespaces for testing) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 4 | [Apply kyverno Cluster Policy and assert it exists](#step-Apply kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 5 | [create pods in random-ns](#step-create pods in random-ns) | 0 | 1 | 0 | 0 | 0 |
| 6 | [assert pods in random-ns are not labeled](#step-assert pods in random-ns are not labeled) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply GlobalContextEntry`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Create namespaces for testing`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `Apply RBAC`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Apply kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 1 | 0 | *No description* |

### Step: `create pods in random-ns`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `assert pods in random-ns are not labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 1 | 0 | *No description* |

---

# Test: `rule-not-applied-to-rhtap-releng-tenant`

Tests that the Kyverno policy does not apply to pods in managed tenant namespaces.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply GlobalContextEntry](#step-Apply GlobalContextEntry) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Create a managed namespace](#step-Create a managed namespace) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 4 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 5 | [Create a pod in the namespace](#step-Create a pod in the namespace) | 0 | 1 | 0 | 0 | 0 |
| 6 | [Assert pod in namespace is not labeled](#step-Assert pod in namespace is not labeled) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply GlobalContextEntry`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Create a managed namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Apply RBAC`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 1 | 0 | *No description* |

### Step: `Create a pod in the namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Assert pod in namespace is not labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `create-pod-in-tenant-namespace-without-cost-center`

Tests that it is possible to create a pod in an existing tenant namespace
that does not have the `cost-center` label.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply GlobalContextEntry](#step-Apply GlobalContextEntry) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Create a tenant namespace without cost-center label](#step-Create a tenant namespace without cost-center label) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 4 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 5 | [Create a pod in the tenant namespace without cost-center label](#step-Create a pod in the tenant namespace without cost-center label) | 0 | 1 | 0 | 0 | 0 |
| 6 | [Assert pod in tenant namespace is created successfully](#step-Assert pod in tenant namespace is created successfully) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply GlobalContextEntry`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Create a tenant namespace without cost-center label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Apply RBAC`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 1 | 0 | *No description* |

### Step: `Create a pod in the tenant namespace without cost-center label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Assert pod in tenant namespace is created successfully`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 1 | 0 | *No description* |

---

