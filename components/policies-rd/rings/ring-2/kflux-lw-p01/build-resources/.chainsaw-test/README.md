# Test: `mutate-pod-remote-platform`

Tests that a buildah-remote-oci-ta Pod with a remote platform (not in
local-platforms) gets its build step resources adjusted to cpu=1, memory=2Gi.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-test-namespace-is-tenant-labeled](#step-given-test-namespace-is-tenant-labeled) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 5 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 6 | [when-pod-with-remote-platform-is-created](#step-when-pod-with-remote-platform-is-created) | 0 | 1 | 0 | 0 | 0 |
| 7 | [then-build-step-resources-are-adjusted](#step-then-build-step-resources-are-adjusted) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-test-namespace-is-tenant-labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-pod-with-remote-platform-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-build-step-resources-are-adjusted`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `skip-mutation-local-platform`

Tests that a buildah-remote-oci-ta Pod with a local platform (listed in
local-platforms) does NOT get its build step resources adjusted.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-test-namespace-is-tenant-labeled](#step-given-test-namespace-is-tenant-labeled) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 5 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 6 | [when-pod-with-local-platform-is-created](#step-when-pod-with-local-platform-is-created) | 0 | 1 | 0 | 0 | 0 |
| 7 | [then-build-step-resources-are-not-adjusted](#step-then-build-step-resources-are-not-adjusted) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-test-namespace-is-tenant-labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-pod-with-local-platform-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-build-step-resources-are-not-adjusted`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `skip-mutation-no-platform`

Tests that a buildah-remote-oci-ta Pod without a PLATFORM environment
variable does NOT get its build step resources adjusted.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-test-namespace-is-tenant-labeled](#step-given-test-namespace-is-tenant-labeled) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 5 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 6 | [when-pod-without-platform-env-is-created](#step-when-pod-without-platform-env-is-created) | 0 | 1 | 0 | 0 | 0 |
| 7 | [then-build-step-resources-are-not-adjusted](#step-then-build-step-resources-are-not-adjusted) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-test-namespace-is-tenant-labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-pod-without-platform-env-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-build-step-resources-are-not-adjusted`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `skip-mutation-no-matching-label`

Tests that a Pod without the tekton.dev/task=buildah-remote-oci-ta label
does NOT get its build step resources adjusted, even when the platform
is remote.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-test-namespace-is-tenant-labeled](#step-given-test-namespace-is-tenant-labeled) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 5 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 6 | [when-pod-without-matching-label-is-created](#step-when-pod-without-matching-label-is-created) | 0 | 1 | 0 | 0 | 0 |
| 7 | [then-build-step-resources-are-not-adjusted](#step-then-build-step-resources-are-not-adjusted) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-test-namespace-is-tenant-labeled`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-pod-without-matching-label-is-created`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-build-step-resources-are-not-adjusted`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `skip-mutation-matching-pod-non-tenant-namespace`

Tests that a buildah-remote-oci-ta Pod with a remote platform does NOT
get mutated when the namespace is not labeled as a tenant namespace.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [when-matching-pod-is-created-in-non-tenant-namespace](#step-when-matching-pod-is-created-in-non-tenant-namespace) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-build-step-resources-are-not-adjusted](#step-then-build-step-resources-are-not-adjusted) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-matching-pod-is-created-in-non-tenant-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-build-step-resources-are-not-adjusted`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `skip-mutation-non-matching-pod-non-tenant-namespace`

Tests that a Pod without the buildah-remote-oci-ta label does NOT get
mutated when the namespace is not labeled as a tenant namespace.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [when-non-matching-pod-is-created-in-non-tenant-namespace](#step-when-non-matching-pod-is-created-in-non-tenant-namespace) | 0 | 1 | 0 | 0 | 0 |
| 6 | [then-build-step-resources-are-not-adjusted](#step-then-build-step-resources-are-not-adjusted) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `when-non-matching-pod-is-created-in-non-tenant-namespace`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `then-build-step-resources-are-not-adjusted`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `verify-failure-policy-ignore`

Tests that the ClusterPolicy has failurePolicy: Ignore set in its spec.
This ensures any changes to the failurePolicy field are intentional.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-multi-platform-controller-namespace-exists](#step-given-multi-platform-controller-namespace-exists) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-host-config-configmap-exists](#step-given-host-config-configmap-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 4 | [given-cluster-policy-is-ready](#step-given-cluster-policy-is-ready) | 0 | 2 | 0 | 0 | 0 |
| 5 | [then-cluster-policy-has-failure-policy-ignore](#step-then-cluster-policy-has-failure-policy-ignore) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-multi-platform-controller-namespace-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-host-config-configmap-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-cluster-policy-is-ready`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `then-cluster-policy-has-failure-policy-ignore`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

