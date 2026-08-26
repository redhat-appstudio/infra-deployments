# Test: `verify-clusterpolicy-is-ready`

Tests that the ClusterPolicy for generating konflux-support-ai-konflux-user-support
ClusterRoleBinding is applied successfully and becomes Ready.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-viewer-clusterrole-exists](#step-given-viewer-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-group-crd-exists](#step-given-group-crd-exists) | 0 | 2 | 0 | 0 | 0 |
| 4 | [Apply Kyverno ClusterPolicy and assert it exists](#step-Apply Kyverno ClusterPolicy and assert it exists) | 0 | 2 | 0 | 0 | 0 |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-viewer-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-group-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Apply Kyverno ClusterPolicy and assert it exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

---

# Test: `clusterrolebinding-created-with-users`

Tests that when an ai-konflux-user-support and konflux-sre Groups
are created with users, a ClusterRoleBinding for each group is generated.
The ClusterRoleBindings have Group's User subjects, the correct label,
and they reference the konflux-viewer-user-actions ClusterRole.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-viewer-clusterrole-exists](#step-given-viewer-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-group-crd-exists](#step-given-group-crd-exists) | 0 | 2 | 0 | 0 | 0 |
| 4 | [Apply Kyverno ClusterPolicy](#step-Apply Kyverno ClusterPolicy) | 0 | 2 | 0 | 0 | 0 |
| 5 | [Create ai-konflux-user-support Group with users](#step-Create ai-konflux-user-support Group with users) | 0 | 1 | 0 | 0 | 0 |
| 6 | [Assert ai-konflux-user-support's support ClusterRoleBinding is created correctly](#step-Assert ai-konflux-user-support's support ClusterRoleBinding is created correctly) | 0 | 1 | 0 | 0 | 0 |
| 7 | [Create konflux-sre Group with users](#step-Create konflux-sre Group with users) | 0 | 1 | 0 | 0 | 0 |
| 8 | [Assert konflux-sre's support ClusterRoleBinding is created correctly](#step-Assert konflux-sre's support ClusterRoleBinding is created correctly) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-viewer-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-group-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Apply Kyverno ClusterPolicy`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create ai-konflux-user-support Group with users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `Assert ai-konflux-user-support's support ClusterRoleBinding is created correctly`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

### Step: `Create konflux-sre Group with users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `Assert konflux-sre's support ClusterRoleBinding is created correctly`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `clusterrolebinding-updated-when-users-added`

Tests that when users are added to the ai-konflux-user-support Group,
the ClusterRoleBinding subjects are updated to include the new users.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-viewer-clusterrole-exists](#step-given-viewer-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-group-crd-exists](#step-given-group-crd-exists) | 0 | 2 | 0 | 0 | 0 |
| 4 | [Apply Kyverno ClusterPolicy](#step-Apply Kyverno ClusterPolicy) | 0 | 2 | 0 | 0 | 0 |
| 5 | [Create ai-konflux-user-support Group with initial users](#step-Create ai-konflux-user-support Group with initial users) | 0 | 1 | 0 | 0 | 0 |
| 6 | [Verify initial ClusterRoleBinding](#step-Verify initial ClusterRoleBinding) | 0 | 1 | 0 | 0 | 0 |
| 7 | [Update Group with more users](#step-Update Group with more users) | 0 | 1 | 0 | 0 | 0 |
| 8 | [Assert ClusterRoleBinding is updated with new users](#step-Assert ClusterRoleBinding is updated with new users) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-viewer-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-group-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Apply Kyverno ClusterPolicy`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create ai-konflux-user-support Group with initial users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `Verify initial ClusterRoleBinding`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

### Step: `Update Group with more users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Assert ClusterRoleBinding is updated with new users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `clusterrolebinding-updated-when-users-removed`

Tests that when users are removed from the ai-konflux-user-support Group,
the ClusterRoleBinding subjects are updated to remove those users.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-viewer-clusterrole-exists](#step-given-viewer-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-group-crd-exists](#step-given-group-crd-exists) | 0 | 2 | 0 | 0 | 0 |
| 4 | [Apply Kyverno ClusterPolicy](#step-Apply Kyverno ClusterPolicy) | 0 | 2 | 0 | 0 | 0 |
| 5 | [Create ai-konflux-user-support Group with multiple users](#step-Create ai-konflux-user-support Group with multiple users) | 0 | 1 | 0 | 0 | 0 |
| 6 | [Verify initial ClusterRoleBinding](#step-Verify initial ClusterRoleBinding) | 0 | 1 | 0 | 0 | 0 |
| 7 | [Update Group with fewer users](#step-Update Group with fewer users) | 0 | 1 | 0 | 0 | 0 |
| 8 | [Assert ClusterRoleBinding is updated with fewer users](#step-Assert ClusterRoleBinding is updated with fewer users) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-viewer-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-group-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Apply Kyverno ClusterPolicy`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create ai-konflux-user-support Group with multiple users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `Verify initial ClusterRoleBinding`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

### Step: `Update Group with fewer users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `Assert ClusterRoleBinding is updated with fewer users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

# Test: `clusterrolebinding-with-empty-group`

Tests that when the ai-konflux-user-support Group has no users,
the ClusterRoleBinding is still created but with empty subjects.


## Steps

| # | Name | Bindings | Try | Catch | Finally | Cleanup |
|:-:|---|:-:|:-:|:-:|:-:|:-:|
| 1 | [given-kyverno-has-permission-on-resources](#step-given-kyverno-has-permission-on-resources) | 0 | 1 | 0 | 0 | 0 |
| 2 | [given-viewer-clusterrole-exists](#step-given-viewer-clusterrole-exists) | 0 | 1 | 0 | 0 | 0 |
| 3 | [given-group-crd-exists](#step-given-group-crd-exists) | 0 | 2 | 0 | 0 | 0 |
| 4 | [Apply Kyverno ClusterPolicy](#step-Apply Kyverno ClusterPolicy) | 0 | 2 | 0 | 0 | 0 |
| 5 | [Create ai-konflux-user-support Group with no users](#step-Create ai-konflux-user-support Group with no users) | 0 | 1 | 0 | 0 | 0 |
| 6 | [Assert ClusterRoleBinding exists with correct metadata](#step-Assert ClusterRoleBinding exists with correct metadata) | 0 | 1 | 0 | 0 | 0 |

### Step: `given-kyverno-has-permission-on-resources`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-viewer-clusterrole-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |

### Step: `given-group-crd-exists`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Apply Kyverno ClusterPolicy`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `apply` | 0 | 0 | *No description* |
| 2 | `assert` | 0 | 0 | *No description* |

### Step: `Create ai-konflux-user-support Group with no users`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `create` | 0 | 0 | *No description* |

### Step: `Assert ClusterRoleBinding exists with correct metadata`

*No description*

#### Try

| # | Operation | Bindings | Outputs | Description |
|:-:|---|:-:|:-:|---|
| 1 | `assert` | 0 | 0 | *No description* |

---

