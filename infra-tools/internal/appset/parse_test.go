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
	g.Expect(result.Paths[0].Path).To(Equal("configs/disable-self-provisioning-for-all-cluster/"))
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
