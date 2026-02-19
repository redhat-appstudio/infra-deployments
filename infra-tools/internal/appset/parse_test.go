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
