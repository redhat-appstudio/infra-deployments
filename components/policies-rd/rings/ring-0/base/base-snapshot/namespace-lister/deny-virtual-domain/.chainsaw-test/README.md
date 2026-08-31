# Test: `allow-namespace-without-virtual-label-and-annotation`

Tests that the creation of a tenant namespace neither labeled
nor annotated with domain `virtual.konflux-ci.dev` is allowed


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a tenant namespace without virtual domain label](#step-Create a tenant namespace without virtual domain label) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a tenant namespace without virtual domain label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `allow-tenant-namespace-without-virtual-label-and-annotation`

Tests that creation of a namespace neither labeled nor annotated
with `virtual.konflux-ci.dev` is allowed


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a namespace without labels](#step-Create a namespace without labels) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a namespace without labels`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `allow-namespace-with-virtual-annotation`

Tests that creation of a namespace annotated with
`virtual.konflux-ci.dev` is allowed


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a namespace with virtual label](#step-Create a namespace with virtual label) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a namespace with virtual label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `deny-tenant-namespace-with-virtual-annotation`

Tests that creation of a tenant namespace annotated with
`virtual.konflux-ci.dev` is denied


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a tenant namespace with virtual annotation](#step-Create a tenant namespace with virtual annotation) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a tenant namespace with virtual annotation`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `deny-promoted-tenant-namespace-with-virtual-annotation`

Tests that a namespace annotated with `virtual.konflux-ci.dev` can not
be promoted to tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a namespace with virtual annotation](#step-Create a namespace with virtual annotation) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Promote namespace with virtual annotation to tenant namespace](#step-Promote namespace with virtual annotation to tenant namespace) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a namespace with virtual annotation`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Promote namespace with virtual annotation to tenant namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 1 | 0 | *No description* |

---

# Test: `deny-update-existing-tenant-namespace-with-virtual-annotation`

Tests that an existing tenant namespace cannot be updated to add
an annotation with `virtual.konflux-ci.dev` domain.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a tenant namespace without virtual annotations](#step-Create a tenant namespace without virtual annotations) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Update existing tenant namespace with virtual annotation](#step-Update existing tenant namespace with virtual annotation) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a tenant namespace without virtual annotations`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Update existing tenant namespace with virtual annotation`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 1 | 0 | *No description* |

---

# Test: `allow-namespace-with-virtual-label`

Tests that creation of a namespace labeled with
`virtual.konflux-ci.dev` is allowed


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a namespace with virtual label](#step-Create a namespace with virtual label) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a namespace with virtual label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `deny-tenant-namespace-with-virtual-label`

Tests that creation of a tenant namespace labeled with
`virtual.konflux-ci.dev` is denied


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a tenant namespace with virtual labels](#step-Create a tenant namespace with virtual labels) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a tenant namespace with virtual labels`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `deny-promoted-tenant-namespace-with-virtual-label`

Tests that a namespace labeled with `virtual.konflux-ci.dev` can not
be promoted to tenant namespace


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a namespace with virtual labels](#step-Create a namespace with virtual labels) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Promote namespace with virtual labels to tenant namespace](#step-Promote namespace with virtual labels to tenant namespace) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a namespace with virtual labels`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Promote namespace with virtual labels to tenant namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 1 | 0 | *No description* |

---

# Test: `deny-update-existing-tenant-namespace-with-virtual-label`

Tests that an existing tenant namespace cannot be updated to add
a label with `virtual.konflux-ci.dev` domain.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Apply Kyverno Cluster Policy and assert it exists](#step-Apply Kyverno Cluster Policy and assert it exists) | 0 | 2 | 0 | 0 | 0 |
| 2 | [Create a tenant namespace without virtual labels](#step-Create a tenant namespace without virtual labels) | 0 | 1 | 0 | 0 | 0 |
| 3 | [Update existing tenant namespace with virtual label](#step-Update existing tenant namespace with virtual label) | 0 | 1 | 0 | 0 | 0 |

### Step: `Apply Kyverno Cluster Policy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create a tenant namespace without virtual labels`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

### Step: `Update existing tenant namespace with virtual label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 1 | 0 | *No description* |

---

