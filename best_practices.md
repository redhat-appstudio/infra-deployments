
<b>Pattern 1: Add automated tests alongside policy/configuration changes (Kyverno policies, CEL/tekton-kueue config, admission policies) to lock in expected behavior across both matching and non-matching cases, and adjust tests when cluster-scoped resources or ordering constraints require it (e.g., disable concurrency).
</b>

Example code before:
```
# policy.yaml (changed validation/mutation behavior)
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: restrict-something
spec:
  rules:
    - name: rule
      validate:
        message: "..."
        pattern:
          spec: {}
# (no tests added/updated)
```

Example code after:
```
# policy.yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: restrict-something
spec:
  rules:
    - name: rule
      validate:
        message: "..."
        pattern:
          spec: {}
---
# .chainsaw-test/chainsaw-test.yaml
apiVersion: chainsaw.kyverno.io/v1alpha1
kind: Test
metadata:
  name: restrict-something-matching-and-non-matching
spec:
  concurrent: false
  steps:
    - try:
        - apply: { file: ../policy.yaml }
        - apply: { file: resources/object-matching.yaml }
        - assert: { file: resources/expected-matching.yaml }
    - try:
        - apply: { file: resources/object-not-matching.yaml }
        - assert: { file: resources/expected-not-matching.yaml }
```

<details><summary>Examples for relevant past discussions:</summary>

- https://github.com/redhat-appstudio/infra-deployments/pull/10649#discussion_r2876906831
- https://github.com/redhat-appstudio/infra-deployments/pull/10481#discussion_r2795707981
- https://github.com/redhat-appstudio/infra-deployments/pull/10414#discussion_r2799263802
- https://github.com/redhat-appstudio/infra-deployments/pull/10369#discussion_r2775868841
</details>


___

<b>Pattern 2: Keep changes correctly scoped to the intended environment/overlay (dev vs staging vs prod) and prefer cluster- or environment-specific patches/overlays when a change is not globally applicable, including verifying whether a modified base is shared by production.
</b>

Example code before:
```
# components/foo/base/kustomization.yaml
images:
  - name: foo
    newTag: dev-only-test-tag
# (affects all environments consuming base)
```

Example code after:
```
# components/foo/base/kustomization.yaml
images:
  - name: foo
    newTag: stable-tag
---
# components/foo/development/patches/foo-image-dev.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: foo
    newTag: dev-only-test-tag
```

<details><summary>Examples for relevant past discussions:</summary>

- https://github.com/redhat-appstudio/infra-deployments/pull/10613#discussion_r2828682383
- https://github.com/redhat-appstudio/infra-deployments/pull/10452#discussion_r2789409750
- https://github.com/redhat-appstudio/infra-deployments/pull/10481#discussion_r2795706961
- https://github.com/redhat-appstudio/infra-deployments/pull/10511#discussion_r2802904315
</details>


___

<b>Pattern 3: Reduce duplication by extracting repeated configuration into shared files or base overlays, and keep kustomize patch paths/structure consistent and maintainable (prefer reuse over copy/paste, and align patch referencing conventions within a component).
</b>

Example code before:
```
# overlay-a/patch.yaml
data:
  config.yaml: |
    receivers: { ...large block... }
    processors: { ...large block... }

# overlay-b/patch.yaml
data:
  config.yaml: |
    receivers: { ...same large block... }
    processors: { ...same large block... }
```

Example code after:
```
# base/config/otel-config.yaml
receivers: { ... }
processors: { ... }

# overlay-a/kustomization.yaml
patches:
  - path: ../base/patches/apply-otel-config.yaml

# base/patches/apply-otel-config.yaml
target:
  kind: Secret
  name: otel-config
patch: |
  - op: replace
    path: /data/config.yaml
    value: |
      # included from base/config/otel-config.yaml
      receivers: { ... }
      processors: { ... }
```

<details><summary>Examples for relevant past discussions:</summary>

- https://github.com/redhat-appstudio/infra-deployments/pull/10667#discussion_r2863683589
- https://github.com/redhat-appstudio/infra-deployments/pull/10549#discussion_r2819617458
</details>


___

<b>Pattern 4: Prefer simple, intention-revealing configuration over complex expressions/regexes, and ensure resource/file naming matches the component purpose to avoid confusion and miswiring (e.g., wrong maintainer filenames, unnecessary regex use in policies).
</b>

Example code before:
```
# kustomization.yaml
resources:
  - registry-maintainers.yaml

# policy.yaml (overly complex)
- key: "{{ regex_match('^(docker\\.io|index\\.docker\\.io)/', '{{element.image}}') }}"
  operator: Equals
  value: true
```

Example code after:
```
# kustomization.yaml
resources:
  - cardinality-maintainers.yaml

# policy.yaml (simpler matching)
validate:
  foreach:
    - list: "request.object.spec.[initContainers, ephemeralContainers, containers][]"
      deny:
        conditions:
          any:
            - key: "{{ element.image }}"
              operator: Equals
              value: "docker.io/*"
            - key: "{{ element.image }}"
              operator: Equals
              value: "index.docker.io/*"
            - key: "{{ element.image }}"
              operator: NotEquals
              value: "*/*"
```

<details><summary>Examples for relevant past discussions:</summary>

- https://github.com/redhat-appstudio/infra-deployments/pull/10778#discussion_r2907518479
- https://github.com/redhat-appstudio/infra-deployments/pull/10778#discussion_r2907689779
- https://github.com/redhat-appstudio/infra-deployments/pull/10649#discussion_r2876942230
</details>


___
