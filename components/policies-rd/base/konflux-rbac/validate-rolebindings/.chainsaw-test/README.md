# Test: `disallow-tenant-groups`

Asserts that rolebindings to restricted groups are denied in tenant namespaces


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Given namespace has tenant label](#step-Given namespace has tenant label) | 0 | 1 | 0 | 0 | 0 |
| 2 | [Given cluster policy is ready](#step-Given cluster policy is ready) | 0 | 2 | 0 | 0 | 0 |
| 3 | [When invalid rolebinding is applied, assert an error](#step-When invalid rolebinding is applied, assert an error) | 0 | 3 | 0 | 0 | 0 |

### Step: `Given namespace has tenant label`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Given cluster policy is ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `When invalid rolebinding is applied, assert an error`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 1 | 0 | *No description* |
| 2 | `apply` | 1 | 0 | *No description* |
| 3 | `apply` | 1 | 0 | *No description* |

---

# Test: `allow-system-groups-in-non-tenant-namespaces`

Asserts that rolebindings restricted groups are allowed in non-tenant namespaces


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [Given cluster policy is ready](#step-Given cluster policy is ready) | 0 | 2 | 0 | 0 | 0 |
| 2 | [When valid rolebinding is applied, assert no error](#step-When valid rolebinding is applied, assert no error) | 0 | 3 | 0 | 0 | 0 |

### Step: `Given cluster policy is ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `When valid rolebinding is applied, assert no error`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 1 | 0 | *No description* |
| 2 | `apply` | 1 | 0 | *No description* |
| 3 | `apply` | 1 | 0 | *No description* |

---

