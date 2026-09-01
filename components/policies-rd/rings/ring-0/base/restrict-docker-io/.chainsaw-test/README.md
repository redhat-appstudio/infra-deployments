# Test: `deny-docker-io-image`

Tests that a Pod with an explicit docker.io image is denied.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 2 | [then-docker-io-pod-is-denied](#step-then-docker-io-pod-is-denied) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-docker-io-pod-is-denied`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `deny-index-docker-io-image`

Tests that a Pod with an explicit index.docker.io image is denied.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 2 | [then-index-docker-io-pod-is-denied](#step-then-index-docker-io-pod-is-denied) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-index-docker-io-pod-is-denied`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `deny-unqualified-image`

Tests that a Pod with an unqualified image (no registry, e.g. nginx:latest)
is denied, since it defaults to docker.io.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 2 | [then-unqualified-image-pod-is-denied](#step-then-unqualified-image-pod-is-denied) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-unqualified-image-pod-is-denied`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

# Test: `allow-non-docker-registry`

Tests that a Pod with a fully qualified image from an allowed
registry (registry.access.redhat.com) is accepted.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 2 | [then-allowed-registry-pod-is-created](#step-then-allowed-registry-pod-is-created) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-allowed-registry-pod-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

---

