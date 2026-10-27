package appset

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseApplicationSets_StandardTemplate(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: has
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/has
                environment: staging
                clusterDir: ""
          - list:
              elements: []
  template:
    metadata:
      name: has-{{nameNormalized}}
    spec:
      project: default
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
        targetRevision: main
      destination:
        namespace: application-service
        server: '{{server}}'
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))

	p := result.Paths[0]
	g.Expect(p.Path).To(Equal("components/has/staging"))
	g.Expect(p.ClusterDir).To(BeEmpty())
}

func TestParseApplicationSets_WithClusterOverrides(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: smee-client
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/smee-client
                environment: staging
                clusterDir: ""
          - list:
              elements:
                - nameNormalized: kflux-ocp-p01
                  values.clusterDir: kflux-ocp-p01
                - nameNormalized: stone-prod-p01
                  values.clusterDir: stone-prod-p01
  template:
    metadata:
      name: smee-client-{{nameNormalized}}
    spec:
      project: default
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
        targetRevision: main
      destination:
        namespace: smee-client
        server: '{{server}}'
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())

	// Should have base path + 2 cluster-specific paths
	g.Expect(result.Paths).To(HaveLen(3))

	// Check base path exists
	var foundBase bool
	for _, p := range result.Paths {
		if p.Path == "components/smee-client/staging" && p.ClusterDir == "" {
			foundBase = true
			break
		}
	}
	g.Expect(foundBase).To(BeTrue(), "expected base path 'components/smee-client/staging'")

	// Check cluster paths
	clusterDirs := make([]string, 0)
	for _, p := range result.Paths {
		if p.ClusterDir != "" {
			clusterDirs = append(clusterDirs, p.ClusterDir)
		}
	}
	g.Expect(clusterDirs).To(ConsistOf("kflux-ocp-p01", "stone-prod-p01"))

	// Check cluster name extraction
	g.Expect(result.Clusters).To(HaveKey("kflux-ocp-p01"))
	g.Expect(result.Clusters).To(HaveKey("stone-prod-p01"))
}

func TestParseApplicationSets_StaticPath(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: internal-services
spec:
  generators:
    - clusters: {}
  template:
    metadata:
      name: internal-services-{{nameNormalized}}
    spec:
      project: default
      source:
        path: components/internal-services
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
        targetRevision: main
      destination:
        namespace: internal-services
        server: '{{server}}'
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))

	p := result.Paths[0]
	g.Expect(p.Path).To(Equal("components/internal-services"))
}

func TestParseApplicationSets_StaticConfigPath(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: disable-self-provisioning
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters: {}
          - list:
              elements: []
  template:
    metadata:
      name: disable-self-provisioning-{{nameNormalized}}
    spec:
      project: default
      source:
        path: configs/disable-self-provisioning-for-all-cluster/
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
        targetRevision: main
      destination:
        namespace: openshift-config
        server: '{{server}}'
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	// Trailing slashes are stripped: paths are used as map keys and compared
	// with "<path>/" prefixes downstream.
	g.Expect(result.Paths[0].Path).To(Equal("configs/disable-self-provisioning-for-all-cluster"))
}

func TestParseApplicationSets_MultipleDocuments(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: has
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/has
                environment: production
                clusterDir: ""
          - list:
              elements: []
  template:
    metadata:
      name: has-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
---
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: internal-services
spec:
  generators:
    - clusters: {}
  template:
    metadata:
      name: internal-services-{{nameNormalized}}
    spec:
      source:
        path: components/internal-services
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(2))
}

// ---------------------------------------------------------------------------
// AppSetsByName
// ---------------------------------------------------------------------------

func TestAppSetsByName_Basic(t *testing.T) {
	g := NewWithT(t)

	yaml := `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: kanary
spec:
  generators:
    - merge:
        generators:
          - clusters:
              values:
                environment: staging
  template:
    metadata:
      name: kanary-{{nameNormalized}}
    spec:
      source:
        path: components/kanary/staging
        repoURL: https://example.com/repo.git
      destination:
        server: '{{server}}'
`
	result, err := AppSetsByName([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(HaveKey("kanary"))
}

func TestAppSetsByName_Empty(t *testing.T) {
	g := NewWithT(t)
	result, err := AppSetsByName([]byte{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestAppSetsByName_NonAppSetIgnored(t *testing.T) {
	g := NewWithT(t)
	result, err := AppSetsByName([]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: foo\n"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestAppSetsByName_NoName(t *testing.T) {
	g := NewWithT(t)
	// ApplicationSet with no metadata.name is skipped.
	yaml := `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
spec:
  generators: []
  template:
    metadata:
      name: foo
    spec:
      source:
        path: components/foo
        repoURL: https://example.com/repo.git
      destination:
        server: '{{server}}'
`
	result, err := AppSetsByName([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

func TestAppSetsByName_InvalidYAML(t *testing.T) {
	g := NewWithT(t)
	_, err := AppSetsByName([]byte("not: valid: yaml: [}"))
	g.Expect(err).To(HaveOccurred())
}

func TestEnvironmentFromAppSet_NonMapGenerator(t *testing.T) {
	g := NewWithT(t)
	// Generator element is a string, not a map — should be skipped, return ("", false).
	doc := map[string]interface{}{
		"spec": map[string]interface{}{
			"generators": []interface{}{
				"not-a-map",
			},
		},
	}
	_, ok := EnvironmentFromAppSet(doc)
	g.Expect(ok).To(BeFalse())
}

func TestEnvironmentFromAppSet_NonMapSubGenerator(t *testing.T) {
	g := NewWithT(t)
	// Sub-generator inside merge is a string — should be skipped, return ("", false).
	doc := map[string]interface{}{
		"spec": map[string]interface{}{
			"generators": []interface{}{
				map[string]interface{}{
					"merge": map[string]interface{}{
						"generators": []interface{}{
							"not-a-map",
						},
					},
				},
			},
		},
	}
	_, ok := EnvironmentFromAppSet(doc)
	g.Expect(ok).To(BeFalse())
}

// ---------------------------------------------------------------------------
// EnvironmentFromAppSet
// ---------------------------------------------------------------------------

func TestEnvironmentFromAppSet_MergeGenerator(t *testing.T) {
	g := NewWithT(t)

	doc := map[string]interface{}{
		"spec": map[string]interface{}{
			"generators": []interface{}{
				map[string]interface{}{
					"merge": map[string]interface{}{
						"generators": []interface{}{
							map[string]interface{}{
								"clusters": map[string]interface{}{
									"values": map[string]interface{}{
										"environment": "staging",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	env, ok := EnvironmentFromAppSet(doc)
	g.Expect(ok).To(BeTrue())
	g.Expect(env).To(Equal("staging"))
}

func TestEnvironmentFromAppSet_DirectClustersGenerator(t *testing.T) {
	g := NewWithT(t)

	doc := map[string]interface{}{
		"spec": map[string]interface{}{
			"generators": []interface{}{
				map[string]interface{}{
					"clusters": map[string]interface{}{
						"values": map[string]interface{}{
							"environment": "production",
						},
					},
				},
			},
		},
	}

	env, ok := EnvironmentFromAppSet(doc)
	g.Expect(ok).To(BeTrue())
	g.Expect(env).To(Equal("production"))
}

func TestEnvironmentFromAppSet_NoEnvironment(t *testing.T) {
	g := NewWithT(t)

	// clusters: {} with no values
	doc := map[string]interface{}{
		"spec": map[string]interface{}{
			"generators": []interface{}{
				map[string]interface{}{
					"clusters": map[string]interface{}{},
				},
			},
		},
	}

	_, ok := EnvironmentFromAppSet(doc)
	g.Expect(ok).To(BeFalse())
}

func TestEnvironmentFromAppSet_NoSpec(t *testing.T) {
	g := NewWithT(t)
	_, ok := EnvironmentFromAppSet(map[string]interface{}{})
	g.Expect(ok).To(BeFalse())
}

func TestParseApplicationSets_NonAppSetResourcesIgnored(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
---
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: has
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/has
                environment: staging
                clusterDir: ""
          - list:
              elements: []
  template:
    metadata:
      name: has-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
}

func TestParseApplicationSets_RingTemplate_ClusterDefaults(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: multi-platform-controller
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/multi-platform-controller
                ring: "empty-base"
                clusterDir: "empty-base"
          - list:
              elements: []
  template:
    metadata:
      name: multi-platform-controller-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/multi-platform-controller/rings/empty-base/empty-base"))
	g.Expect(result.Paths[0].ClusterDir).To(Equal("empty-base"))
}

func TestParseApplicationSets_RingTemplate_ListOverrides(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: multi-platform-controller
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/multi-platform-controller
                ring: "empty-base"
                clusterDir: "empty-base"
          - list:
              elements:
                - values.ring: ring-1
                  nameNormalized: stone-stg-rh01
                  values.clusterDir: stone-stg-rh01
                - values.ring: ring-2
                  nameNormalized: kflux-lw-p01
                  values.clusterDir: kflux-lw-p01
  template:
    metadata:
      name: multi-platform-controller-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(3))

	paths := make([]string, 0, len(result.Paths))
	for _, p := range result.Paths {
		paths = append(paths, p.Path)
	}
	g.Expect(paths).To(ConsistOf(
		"components/multi-platform-controller/rings/empty-base/empty-base",
		"components/multi-platform-controller/rings/ring-1/stone-stg-rh01",
		"components/multi-platform-controller/rings/ring-2/kflux-lw-p01",
	))

	g.Expect(result.Clusters).To(HaveKey("stone-stg-rh01"))
	g.Expect(result.Clusters["stone-stg-rh01"]).To(ConsistOf(
		"components/multi-platform-controller/rings/ring-1/stone-stg-rh01",
	))
	g.Expect(result.Clusters).To(HaveKey("kflux-lw-p01"))
	g.Expect(result.Clusters["kflux-lw-p01"]).To(ConsistOf(
		"components/multi-platform-controller/rings/ring-2/kflux-lw-p01",
	))
}

func TestParseApplicationSets_RingTemplate_DevBase(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: etcd-shield
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/etcd-shield
                ring: "ring-0"
                clusterDir: base
          - list:
              elements: []
  template:
    metadata:
      name: etcd-shield-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/etcd-shield/rings/ring-0/base"))
	g.Expect(result.Paths[0].ClusterDir).To(Equal("base"))
}

func TestParseApplicationSets_DirectClusters_RingTemplate(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: etcd-shield
spec:
  generators:
    - clusters:
        values:
          sourceRoot: components/etcd-shield
          ring: "ring-0"
          clusterDir: base
  template:
    metadata:
      name: etcd-shield-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/etcd-shield/rings/ring-0/base"))
}

// ---------------------------------------------------------------------------
// ApplicationSet shapes that were previously dropped silently.
// See TestRealRepoApplicationSets in audit_test.go for the end-to-end guard.
// ---------------------------------------------------------------------------

// Top-level list generator (no merge, no clusters generator), as used by the
// konflux-verifications-rd ApplicationSet in the rd-production overlay.
const topLevelListYAML = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: konflux-verifications-rd
spec:
  generators:
    - list:
        elements:
          - name: stone-prod-p01
            values.clusterDir: stone-prod-p01
            values.ring: ring-2
            values.sourceRoot: components/konflux-verifications-rd
  template:
    metadata:
      name: 'konflux-verifications-{{name}}'
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`

func TestParseApplicationSets_TopLevelListGenerator(t *testing.T) {
	g := NewWithT(t)

	result, err := ParseApplicationSets([]byte(topLevelListYAML))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/konflux-verifications-rd/rings/ring-2/stone-prod-p01"))
	g.Expect(result.Paths[0].ClusterDir).To(Equal("stone-prod-p01"))
	g.Expect(result.Clusters).To(HaveKey("stone-prod-p01"))
}

// A list element may override any values.* key, including ones the parser does
// not special-case. Here the element overrides useCaseDir, which must be
// interpolated like any other value.
func TestParseApplicationSets_UseCaseDirInterpolated(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: k8s-groups
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/k8s-groups
                environment: staging
                useCaseDir: rover
          - list:
              elements:
                - nameNormalized: stone-stage-p01
                  values.useCaseDir: rover/rh-sso-internal
  template:
    metadata:
      name: k8s-groups-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.useCaseDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())

	paths := make([]string, 0, len(result.Paths))
	for _, p := range result.Paths {
		paths = append(paths, p.Path)
	}
	// Both the generator default and the element override must resolve.
	g.Expect(paths).To(ConsistOf(
		"components/k8s-groups/staging/rover",
		"components/k8s-groups/staging/rover/rh-sso-internal",
	))
}

// List elements may override any values.* key, not just clusterDir/ring.
func TestParseApplicationSets_ListOverrideOfEnvironment(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: mixed
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/mixed
                environment: development
                clusterDir: ""
          - list:
              elements:
                - nameNormalized: stone-stage-p01
                  values.environment: staging
                  values.clusterDir: stone-stage-p01
  template:
    metadata:
      name: mixed-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())

	paths := make([]string, 0, len(result.Paths))
	for _, p := range result.Paths {
		paths = append(paths, p.Path)
	}
	g.Expect(paths).To(ConsistOf(
		"components/mixed/development",
		"components/mixed/staging/stone-stage-p01",
	))
}

// A {{nameNormalized}}-based template must be interpolated literally, without
// inventing a sourceRoot/environment/name layout.
func TestParseApplicationSets_NameNormalizedTemplate(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: by-name
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/by-name
                environment: staging
          - list:
              elements:
                - nameNormalized: kflux-stg-p02
  template:
    metadata:
      name: by-name-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{nameNormalized}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())

	paths := make([]string, 0, len(result.Paths))
	for _, p := range result.Paths {
		paths = append(paths, p.Path)
	}
	g.Expect(paths).To(ConsistOf(
		"components/by-name/staging",
		"components/by-name/staging/kflux-stg-p02",
	))
	// No path may end in a slash: trailing slashes break cluster matching and
	// dependency deduplication downstream.
	for _, p := range result.Paths {
		g.Expect(p.Path).NotTo(HaveSuffix("/"))
	}
}

// Multi-source templates reference repository files through "$values/" Helm
// value files without deploying a directory of their own.
func TestParseApplicationSets_MultiSourceHelmValues(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: tracing-workload-otel-collector
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                clusterDir: ""
                environment: staging
                sourceRoot: components/monitoring/tracing/otel-collector
          - list:
              elements: []
  template:
    metadata:
      name: tracing-{{nameNormalized}}
    spec:
      sources:
        - chart: opentelemetry-collector
          repoURL: https://open-telemetry.github.io/opentelemetry-helm-charts
          targetRevision: 0.101.2
          helm:
            valueFiles:
              - $values/{{values.sourceRoot}}/{{values.environment}}/otel-collector-helm-values.yaml
        - ref: values
          repoURL: https://github.com/redhat-appstudio/infra-deployments.git
          targetRevision: main
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))

	cp := result.Paths[0]
	g.Expect(cp.Path).To(BeEmpty())
	g.Expect(cp.ExtraFiles).To(ConsistOf(
		"components/monitoring/tracing/otel-collector/staging/otel-collector-helm-values.yaml",
	))
}

// A source may deploy a directory and consume value files at the same time.
func TestParseApplicationSets_PathWithHelmValues(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: chart-component
spec:
  generators:
    - clusters:
        values:
          sourceRoot: components/chart-component
          environment: production
          clusterDir: base
  template:
    metadata:
      name: chart-component
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://example.com/repo.git
        helm:
          valueFiles:
            - $values/{{values.sourceRoot}}/{{values.environment}}/values.yaml
            - values-local.yaml
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))

	cp := result.Paths[0]
	g.Expect(cp.Path).To(Equal("components/chart-component/production/base"))
	// Only "$values/" references live in this repository.
	g.Expect(cp.ExtraFiles).To(ConsistOf(
		"components/chart-component/production/values.yaml",
	))
}

// matrixAppSet returns an ApplicationSet driven by an unmodelled matrix
// generator with the given source path template.
func matrixAppSet(pathTemplate string) string {
	return `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: matrix-generated
spec:
  generators:
    - matrix:
        generators:
          - clusters:
              values:
                sourceRoot: components/matrix
          - git:
              repoURL: https://example.com/repo.git
  template:
    metadata:
      name: matrix-generated
    spec:
      source:
        path: '` + pathTemplate + `'
        repoURL: https://example.com/repo.git
`
}

// An unsupported generator must not drop the ApplicationSet: a static path is
// still tracked exactly.
func TestParseApplicationSets_UnsupportedGeneratorKeepsStaticPath(t *testing.T) {
	g := NewWithT(t)

	result, err := ParseApplicationSets([]byte(matrixAppSet("components/matrix/staging")))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/matrix/staging"))
	g.Expect(result.Paths[0].PrefixOnly).To(BeFalse())
}

// The values of an unsupported generator cannot be trusted, so a templated
// path falls back to its static prefix instead of being dropped.
func TestParseApplicationSets_UnsupportedGeneratorUsesStaticPrefix(t *testing.T) {
	g := NewWithT(t)

	result, err := ParseApplicationSets([]byte(matrixAppSet("components/matrix/{{values.environment}}")))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/matrix"))
	g.Expect(result.Paths[0].PrefixOnly).To(BeTrue())
}

// A path template with nothing static to fall back on cannot be attributed to
// anything in the repository: parsing fails instead of silently reporting no
// affected environment.
func TestParseApplicationSets_NoStaticPrefixIsAnError(t *testing.T) {
	g := NewWithT(t)

	_, err := ParseApplicationSets([]byte(matrixAppSet("{{values.sourceRoot}}/{{values.environment}}")))
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("matrix-generated"))
	g.Expect(err.Error()).To(ContainSubstring("no static prefix"))
}

// Template expressions the placeholder syntax does not cover must not be taken
// for a literal path: they fall back to the static prefix.
func TestParseApplicationSets_UnparsedTemplateExpressionUsesStaticPrefix(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: gotemplate
spec:
  generators:
    - clusters: {}
  template:
    metadata:
      name: gotemplate
    spec:
      source:
        path: 'components/gotemplate/{{ .name | lower }}'
        repoURL: https://example.com/repo.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/gotemplate"))
	g.Expect(result.Paths[0].PrefixOnly).To(BeTrue())
}

// Several list elements may resolve to the same path; every one of their
// clusters must still be attributed to it, even though the path is recorded
// once.
func TestParseApplicationSets_SharedPathAttributedToEveryCluster(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: k8s-groups
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/k8s-groups
                environment: staging
                useCaseDir: rover
          - list:
              elements:
                - nameNormalized: stone-stage-p01
                  values.useCaseDir: rover/rh-sso-internal
                - nameNormalized: kflux-stg-p02
                  values.useCaseDir: rover/rh-sso-internal
  template:
    metadata:
      name: k8s-groups-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.useCaseDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())

	shared := "components/k8s-groups/staging/rover/rh-sso-internal"
	sharedCount := 0
	for _, p := range result.Paths {
		if p.Path == shared {
			sharedCount++
		}
	}
	g.Expect(sharedCount).To(Equal(1))
	g.Expect(result.Clusters).To(HaveKeyWithValue("stone-stage-p01", []string{shared}))
	g.Expect(result.Clusters).To(HaveKeyWithValue("kflux-stg-p02", []string{shared}))
}

// Top-level generators are a union in ArgoCD: the values of one generator must
// not leak into the Applications of another, and each generator contributes
// its own paths.
func TestParseApplicationSets_TopLevelGeneratorsAreAUnion(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: union
spec:
  generators:
    - clusters:
        values:
          sourceRoot: components/union-a
          environment: staging
    - clusters:
        values:
          sourceRoot: components/union-b
          environment: production
    - list:
        elements:
          - name: stone-prod-p01
            values.sourceRoot: components/union-c
            values.environment: production
  template:
    metadata:
      name: union-{{name}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())

	paths := make([]string, 0, len(result.Paths))
	for _, p := range result.Paths {
		paths = append(paths, p.Path)
	}
	g.Expect(paths).To(ConsistOf(
		"components/union-a/staging",
		"components/union-b/production",
		"components/union-c/production",
	))
}

// A template the generator defaults cannot fully resolve (here because no list
// element supplies nameNormalized) keeps its longest static prefix and is
// flagged for prefix matching, so the shared dependencies of the environment
// still count as affected.
func TestParseApplicationSets_PartialTemplateUsesStaticPrefix(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: partial
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/partial
                environment: staging
          - list:
              elements: []
  template:
    metadata:
      name: partial-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{nameNormalized}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))

	cp := result.Paths[0]
	g.Expect(cp.Path).To(Equal("components/partial/staging"))
	g.Expect(cp.PrefixOnly).To(BeTrue())
}

// A fully resolved path must not be flagged for prefix matching, otherwise the
// dependency tree would be replaced by a much broader prefix match.
func TestParseApplicationSets_ResolvedPathIsNotPrefixOnly(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: resolved
spec:
  generators:
    - clusters:
        values:
          sourceRoot: components/resolved
          environment: staging
          clusterDir: ""
  template:
    metadata:
      name: resolved
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	result, err := ParseApplicationSets([]byte(yaml))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Paths).To(HaveLen(1))
	g.Expect(result.Paths[0].Path).To(Equal("components/resolved/staging"))
	g.Expect(result.Paths[0].PrefixOnly).To(BeFalse())
}
