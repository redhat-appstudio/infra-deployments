# Test: `allow-namespace-with-cost-center`

Tests that a namespace labeled as `konflux-ci.dev/type: tenant`
is allowed only if it has the `cost-center` label.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 3 | [Create a tenant namespace with cost-center label](#step-Create a tenant namespace with cost-center label) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `Create a tenant namespace with cost-center label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 2 | 0 | *No description* |

---

# Test: `allow-namespace-without-tenant-label`

Tests that a namespace without the `konflux-ci.dev/type: tenant` label
is allowed regardless of the `cost-center` label.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 3 | [Create a namespace without tenant label](#step-Create a namespace without tenant label) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `Create a namespace without tenant label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `deny-namespace-with-empty-cost-center`

Tests that a namespace labeled as `konflux-ci.dev/type: tenant`
is denied if the `cost-center` label is empty.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 3 | [Attempt to create a tenant namespace with empty cost-center label](#step-Attempt to create a tenant namespace with empty cost-center label) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `Attempt to create a tenant namespace with empty cost-center label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `deny-namespace-without-cost-center`

Tests that a namespace labeled as `konflux-ci.dev/type: tenant`
is denied if it does not have the `cost-center` label.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 3 | [Attempt to create a tenant namespace without cost-center label](#step-Attempt to create a tenant namespace without cost-center label) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `Attempt to create a tenant namespace without cost-center label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `deny-namespace-with-invalid-cost-center`

Tests that a namespace labeled as `konflux-ci.dev/type: tenant`
is denied if the `cost-center` label is invalid.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply RBAC](#step-Apply RBAC) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 3 | [Attempt to create a tenant namespace with invalid cost-center label](#step-Attempt to create a tenant namespace with invalid cost-center label) | 0 | 1 | 0 | 0 | 0 |

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

### Step: `Attempt to create a tenant namespace with invalid cost-center label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 2 | 0 | *No description* |

---

