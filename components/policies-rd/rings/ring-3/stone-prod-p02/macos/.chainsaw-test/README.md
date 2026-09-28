# Test: `deny-macos-in-non-allowed-namespace`

Ensure that pipelineruns requesting macos VMs are not admitted to non-allowed namespaces


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-pipelineruns-exist](#step-given-pipelineruns-exist) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-configmap-exists](#step-given-configmap-exists) | 1 | 3 | 0 | 0 | 0 |
| 3 | [given-admission-policy-exists](#step-given-admission-policy-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-pipelinerun-is-not-created](#step-then-pipelinerun-is-not-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-pipelineruns-exist`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-configmap-exists`

*No description*

#### Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `allowedNamespace` | "(join('-', [$namespace, 'foo']))" |

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `given-admission-policy-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-pipelinerun-is-not-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `allow-macos-in-valid-namespace`

Ensure that pipelineruns requesting macos VMs are admitted to macos-allowed namespaces.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konfluxcidev" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-pipelineruns-exist](#step-given-pipelineruns-exist) | 0 | 1 | 0 | 0 | 1 |
| 2 | [given-configmap-exists](#step-given-configmap-exists) | 1 | 3 | 0 | 0 | 0 |
| 3 | [given-admission-policy-exists](#step-given-admission-policy-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-pipelinerun-is-created](#step-then-pipelinerun-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-pipelineruns-exist`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

#### Cleanup

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `given-configmap-exists`

*No description*

#### Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `allowedNamespace` | "($namespace)" |

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `given-admission-policy-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-pipelinerun-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `multiple-namespaces`

Multiple namespaces can be in the allowlist, and they should all be allowed to acquire macos runners


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `secondNamespace` | "(join('-', [$namespace, 'foo']))" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-pipelineruns-exist](#step-given-pipelineruns-exist) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-second-namespace-exists](#step-given-second-namespace-exists) | 0 | 1 | 0 | 0 | 1 |
| 3 | [given-configmap-exists](#step-given-configmap-exists) | 1 | 3 | 0 | 0 | 0 |
| 4 | [given-admission-policy-exists](#step-given-admission-policy-exists) | 0 | 1 | 0 | 0 | 0 |
| 5 | [then-pipelinerun-is-created-in-first-namespace](#step-then-pipelinerun-is-created-in-first-namespace) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-pipelinerun-is-created-in-second-namespace](#step-then-pipelinerun-is-created-in-second-namespace) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-pipelineruns-exist`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-second-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

#### Cleanup

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `given-configmap-exists`

*No description*

#### Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `allowedNamespace` | "(join(`\"\\u000a\"`, [$namespace, $secondNamespace]))" |

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `given-admission-policy-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-pipelinerun-is-created-in-first-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `then-pipelinerun-is-created-in-second-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 1 | 0 | *No description* |

---

# Test: `allow-non-macos-pipelineruns-in-allowed-namespaces`

Ensure that pipelineruns not requesting macos VMs are admitted to the cluster.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konfluxcidev" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-pipelineruns-exist](#step-given-pipelineruns-exist) | 0 | 1 | 0 | 0 | 1 |
| 2 | [given-configmap-exists](#step-given-configmap-exists) | 1 | 3 | 0 | 0 | 0 |
| 3 | [given-admission-policy-exists](#step-given-admission-policy-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-pipelinerun-is-created](#step-then-pipelinerun-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-pipelineruns-exist`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

#### Cleanup

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `given-configmap-exists`

*No description*

#### Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `allowedNamespace` | "($namespace)" |

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `given-admission-policy-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-pipelinerun-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `allow-non-macos-pipelineruns-in-disallowed-namespaces`

Ensure that pipelineruns not requesting macos VMs in disallowed namespaces are admitted to the cluster.


## Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `suffix` | "konfluxcidev" |

## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-pipelineruns-exist](#step-given-pipelineruns-exist) | 0 | 1 | 0 | 0 | 1 |
| 2 | [given-configmap-exists](#step-given-configmap-exists) | 1 | 3 | 0 | 0 | 0 |
| 3 | [given-admission-policy-exists](#step-given-admission-policy-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [then-pipelinerun-is-created](#step-then-pipelinerun-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-pipelineruns-exist`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

#### Cleanup

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `delete` | 0 | 0 | *No description* |

### Step: `given-configmap-exists`

*No description*

#### Bindings

| # | Name | Value |
|:-:|---|---|
| 1 | `allowedNamespace` | "(join('-', [$namespace, 'foo']))" |

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `apply` | 0 | 0 | *No description* |
| 3 | `assert` | 0 | 0 | *No description* |

### Step: `given-admission-policy-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-pipelinerun-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

