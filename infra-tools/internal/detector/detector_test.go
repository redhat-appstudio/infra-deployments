package detector

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/deptree"
)

// ---------------------------------------------------------------------------
// fakeRepo implements RepoQuerier for deterministic, filesystem-free testing.
// ---------------------------------------------------------------------------

type fakeRepo struct {
	dirs    map[string][]string        // ListSubDirs results keyed by rel path
	exist   map[string]bool            // DirExists results keyed by rel path
	files   map[string]bool            // FileExists results keyed by rel path
	yamls   map[string][]byte          // BuildKustomization results keyed by rel path
	deps    map[string]map[string]bool // ResolveDeps results keyed by rel path
	depsErr map[string]error           // ResolveDeps errors keyed by rel path
}

func (f *fakeRepo) ListSubDirs(rel string) ([]string, error) {
	if d, ok := f.dirs[rel]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("no such dir: %s", rel)
}

func (f *fakeRepo) DirExists(rel string) bool {
	return f.exist[rel]
}

func (f *fakeRepo) FileExists(rel string) bool {
	return f.files[rel]
}

func (f *fakeRepo) ContainedDirExists(rel string) bool {
	return f.exist[rel]
}

func (f *fakeRepo) BuildKustomization(rel string) ([]byte, error) {
	if y, ok := f.yamls[rel]; ok {
		return y, nil
	}
	return nil, fmt.Errorf("no kustomization at %s", rel)
}

func (f *fakeRepo) ResolveDeps(rel string) (map[string]bool, error) {
	d := f.deps[rel]
	if err, ok := f.depsErr[rel]; ok {
		return d, err
	}
	if d != nil {
		return d, nil
	}
	return nil, fmt.Errorf("no deps for %s", rel)
}

// ---------------------------------------------------------------------------
// A minimal ApplicationSet YAML for testing.
// It declares a single component at components/foo with no cluster generators.
// ---------------------------------------------------------------------------

const minimalAppSetYAML = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: foo
spec:
  generators:
    - clusters: {}
  template:
    metadata:
      name: foo
    spec:
      source:
        path: components/foo
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`

// appSetWithRing returns an ApplicationSet YAML that uses the ring-based path
// template: sourceRoot/rings/ring/clusterDir.
func appSetWithRing(root, defaultRing, defaultClusterDir, listRing, listCluster string) string {
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: test-app
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: %s
                ring: %q
                clusterDir: %q
          - list:
              elements:
                - nameNormalized: %s
                  values.ring: %s
                  values.clusterDir: %s
  template:
    metadata:
      name: test-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`, root, defaultRing, defaultClusterDir, listCluster, listRing, listCluster)
}

// appSetWithCluster returns an ApplicationSet YAML that uses a merge generator
// with a cluster list, producing paths under <root>/<env> and <root>/<env>/<cluster>.
func appSetWithCluster(root, env, cluster string) string {
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: test-app
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: %s
                environment: %s
                clusterDir: ""
          - list:
              elements:
                - nameNormalized: %s
                  values.clusterDir: %s
  template:
    metadata:
      name: test-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`, root, env, cluster, cluster)
}

// ringAppSet returns a ring-based ApplicationSet whose path carries no
// environment; only the overlay that renders it identifies the environment.
func ringAppSet(root, ring, cluster, env string) string {
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: ring-app
spec:
  generators:
    - clusters:
        values:
          sourceRoot: %s
          ring: %s
          clusterDir: %s
          environment: %s
  template:
    metadata:
      name: ring-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/rings/{{values.ring}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`, root, ring, cluster, env)
}

// matchAllAppSet returns an ApplicationSet that cannot be resolved; the parser
// emits a MatchAll entry for it.
const matchAllAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: unresolved
spec:
  generators:
    - clusters: {}
  templatePatch: '{{ . }}'
  template:
    metadata:
      name: unresolved
    spec:
      source:
        path: components/unresolved
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`

// appSetWithClusterWithClusters is appSetWithCluster with several list elements.
func appSetWithClusterWithClusters(root, env string, clusters []string) string {
	var b strings.Builder
	for _, c := range clusters {
		fmt.Fprintf(&b, "                - nameNormalized: %s\n                  values.clusterDir: %s\n", c, c)
	}
	elements := b.String()
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: test-app
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: %s
                environment: %s
                clusterDir: ""
          - list:
              elements:
%s
  template:
    metadata:
      name: test-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`, root, env, elements)
}

// appSetStaticPathWithEnv returns a static-path ApplicationSet that declares
// the given environment.
func appSetStaticPathWithEnv(path, env string) string {
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: static
spec:
  generators:
    - clusters:
        values:
          environment: %s
  template:
    metadata:
      name: static
    spec:
      source:
        path: %s
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`, env, path)
}

// ---------------------------------------------------------------------------
// NewDetector
// ---------------------------------------------------------------------------

func TestNewDetector_Valid(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{dirs: map[string][]string{"overlays": {"development", "konflux-public-production"}}}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.overlayEnvs).To(HaveLen(2))
	g.Expect(d.overlayEnvs["development"]).To(Equal(Development))
	g.Expect(d.overlayEnvs["konflux-public-production"]).To(Equal(Production))
}

func TestNewDetector_Valid_Ring_Deployments(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{dirs: map[string][]string{"overlays": {"rd-dev", "rd-staging", "rd-production"}}}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"rd-dev"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.overlayEnvs).To(HaveLen(3))
	g.Expect(d.overlayEnvs["rd-dev"]).To(Equal(Development))
	g.Expect(d.overlayEnvs["rd-staging"]).To(Equal(Staging))
	g.Expect(d.overlayEnvs["rd-production"]).To(Equal(Production))
}

func TestNewDetector_UnknownOverlay(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{dirs: map[string][]string{"overlays": {"development", "unknown-overlay"}}}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	_, err := NewDetector(head, base, "overlays")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown overlay"))
	g.Expect(err.Error()).To(ContainSubstring("unknown-overlay"))
}

func TestNewDetector_BaseAddsUnknownOverlay(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development", "unknown-overlay"}}}

	_, err := NewDetector(head, base, "overlays")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown-overlay"))
}

func TestNewDetector_HeadListError(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{dirs: map[string][]string{}} // no entry for "overlays"
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	_, err := NewDetector(head, base, "overlays")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("listing overlay dirs on HEAD"))
}

func TestNewDetector_BaseMissing(t *testing.T) {
	g := NewWithT(t)

	// Base overlay dir doesn't exist (first PR that adds overlays) — should NOT error.
	head := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}
	base := &fakeRepo{dirs: map[string][]string{}} // missing

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.overlayEnvs).To(HaveKey("development"))
}

// ---------------------------------------------------------------------------
// buildAppSetOverlays
// ---------------------------------------------------------------------------

func TestBuildAppSetOverlays_Success(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("head-yaml")},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("base-yaml")},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	builds, err := d.buildAppSetOverlays()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(builds).To(HaveLen(1))
	g.Expect(builds[0].name).To(Equal("development"))
	g.Expect(builds[0].env).To(Equal(Development))
	g.Expect(string(builds[0].headYAML)).To(Equal("head-yaml"))
	g.Expect(string(builds[0].baseYAML)).To(Equal("base-yaml"))
}

func TestBuildAppSetOverlays_NewOverlayOnHead(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("head-yaml")},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		exist: map[string]bool{}, // overlay does NOT exist on base
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	builds, err := d.buildAppSetOverlays()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(builds).To(HaveLen(1))
	g.Expect(builds[0].baseYAML).To(BeNil()) // no base YAML
}

func TestBuildAppSetOverlays_HeadBuildError(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{}, // no YAML → BuildKustomization will fail
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development"}},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = d.buildAppSetOverlays()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("building overlay development on HEAD"))
}

func TestBuildAppSetOverlays_BaseBuildError(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("head-yaml")},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		exist: map[string]bool{"overlays/development": true},
		yamls: map[string][]byte{}, // BuildKustomization will fail
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = d.buildAppSetOverlays()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("building overlay development on base-ref"))
}

func TestBuildAppSetOverlays_MultipleOverlays(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development", "konflux-public-production"}},
		yamls: map[string][]byte{
			"overlays/development":               []byte("dev-yaml"),
			"overlays/konflux-public-production": []byte("prod-yaml"),
		},
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development", "konflux-public-production"}},
		yamls: map[string][]byte{
			"overlays/development":               []byte("dev-yaml"),
			"overlays/konflux-public-production": []byte("prod-yaml"),
		},
		exist: map[string]bool{
			"overlays/development":               true,
			"overlays/konflux-public-production": true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	builds, err := d.buildAppSetOverlays()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(builds).To(HaveLen(2))
}

// ---------------------------------------------------------------------------
// detectRemovedAppSetOverlays
// ---------------------------------------------------------------------------

func TestDetectRemovedAppSetOverlays_OverlayRemoved(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("yaml")},
		exist: map[string]bool{"overlays/development": true}, // development still exists, production does NOT
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development", "konflux-public-production"}},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result := &Result{AffectedEnvironments: make(map[Environment]bool), AffectedClusters: make(map[string]bool)}
	d.detectRemovedAppSetOverlays(result)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Development))
}

func TestDetectRemovedAppSetOverlays_NoneRemoved(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("yaml")},
		exist: map[string]bool{"overlays/development": true},
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development"}},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result := &Result{AffectedEnvironments: make(map[Environment]bool), AffectedClusters: make(map[string]bool)}
	d.detectRemovedAppSetOverlays(result)

	g.Expect(result.AffectedEnvironments).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// resolveComponentDeps
// ---------------------------------------------------------------------------

func TestResolveComponentDeps_WithDepTree(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {
				"components/foo/kustomization.yaml": true,
				"components/foo/deploy.yaml":        true,
			},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	envPaths := map[Environment][]appset.ComponentPath{
		Development: {{Path: "components/foo"}},
	}

	resolved := d.resolveComponentDeps(envPaths)
	g.Expect(resolved).To(HaveLen(1))
	g.Expect(resolved[0].env).To(Equal(Development))
	g.Expect(resolved[0].deps).To(HaveKey("components/foo/deploy.yaml"))
}

func TestResolveComponentDeps_NoDepsUsePrefixFallback(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true},
		deps:  map[string]map[string]bool{}, // ResolveDeps will fail
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	envPaths := map[Environment][]appset.ComponentPath{
		Development: {{Path: "components/foo"}},
	}

	resolved := d.resolveComponentDeps(envPaths)
	g.Expect(resolved).To(HaveLen(1))
	g.Expect(resolved[0].deps).To(BeNil()) // nil deps → prefix fallback
}

func TestResolveComponentDeps_SkipsMissingDir(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		exist: map[string]bool{}, // components/foo does NOT exist
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	envPaths := map[Environment][]appset.ComponentPath{
		Development: {{Path: "components/foo"}},
	}

	resolved := d.resolveComponentDeps(envPaths)
	g.Expect(resolved).To(BeEmpty())
}

func TestResolveComponentDeps_Deduplication(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		exist: map[string]bool{"components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	// Same path appears twice under the same env.
	envPaths := map[Environment][]appset.ComponentPath{
		Development: {
			{Path: "components/foo"},
			{Path: "components/foo"}, // duplicate
		},
	}

	resolved := d.resolveComponentDeps(envPaths)
	g.Expect(resolved).To(HaveLen(1)) // deduped
}

// ---------------------------------------------------------------------------
// Detect (end-to-end)
// ---------------------------------------------------------------------------

func TestDetect_DepTreeMatch(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true, "overlays/development": true},
		deps: map[string]map[string]bool{
			"components/foo": {
				"components/foo/kustomization.yaml": true,
				"components/foo/deploy.yaml":        true,
			},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)}, // same YAML → no overlay diff
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/foo/deploy.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	// The overlay YAML is the same on both refs, so the overlay diff alone
	// wouldn't trigger; the match comes from the dep tree.
}

func TestDetect_OverlayDiff(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true, "overlays/development": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("different-yaml")}, // different!
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	// The changed file doesn't match any component, but the overlay diff
	// should still mark the environment as affected.
	result, err := d.Detect([]string{"README.md"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

func TestDetect_PrefixFallback(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true},
		deps:  map[string]map[string]bool{}, // ResolveDeps fails → prefix fallback
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/foo/manifest.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

func TestDetect_NoMatch(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true, "overlays/development": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"README.md"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(BeEmpty())
	g.Expect(result.AffectedClusters).To(BeEmpty())
}

func TestDetect_StaticRuleAppOfAppSets(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true, "overlays/development": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"argo-cd-apps/app-of-app-sets/production/kustomization.yaml"})
	g.Expect(err).NotTo(HaveOccurred())

	// Static rule should mark ALL environments.
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
}

func TestDetect_WithCluster(t *testing.T) {
	g := NewWithT(t)

	yaml := appSetWithCluster("components/svc", "staging", "stone-prod-p01")

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		yamls: map[string][]byte{"overlays/staging-downstream": []byte(yaml)},
		exist: map[string]bool{
			"overlays/staging-downstream":           true,
			"components/svc/staging":                true,
			"components/svc/staging/stone-prod-p01": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/staging": {
				"components/svc/staging/kustomization.yaml": true,
			},
			"components/svc/staging/stone-prod-p01": {
				"components/svc/staging/stone-prod-p01/kustomization.yaml": true,
				"components/svc/staging/stone-prod-p01/patch.yaml":         true,
			},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		yamls: map[string][]byte{"overlays/staging-downstream": []byte(yaml)},
		exist: map[string]bool{"overlays/staging-downstream": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/svc/staging/stone-prod-p01/patch.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedClusters).To(HaveKey("stone-prod-p01"))
}

func TestDetect_RingPathsAttributedToOverlayEnv(t *testing.T) {
	g := NewWithT(t)

	stagingYAML := appSetWithRing(
		"components/multi-platform-controller",
		"empty-base", "empty-base",
		"ring-1", "stone-stg-rh01",
	)
	productionYAML := appSetWithRing(
		"components/multi-platform-controller",
		"empty-base", "empty-base",
		"ring-2", "kflux-lw-p01",
	)
	const stagingPath = "components/multi-platform-controller/rings/ring-1/stone-stg-rh01"
	const productionPath = "components/multi-platform-controller/rings/ring-2/kflux-lw-p01"

	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {"rd-staging", "rd-production"}},
		yamls: map[string][]byte{
			"overlays/rd-staging":    []byte(stagingYAML),
			"overlays/rd-production": []byte(productionYAML),
		},
		exist: map[string]bool{
			"overlays/rd-staging":    true,
			"overlays/rd-production": true,
			"components/multi-platform-controller/rings/empty-base/empty-base": true,
			stagingPath:    true,
			productionPath: true,
		},
		deps: map[string]map[string]bool{
			stagingPath: {
				stagingPath + "/host-config.yaml": true,
			},
			productionPath: {
				productionPath + "/host-config.yaml": true,
			},
		},
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"rd-staging", "rd-production"}},
		yamls: map[string][]byte{
			"overlays/rd-staging":    []byte(stagingYAML),
			"overlays/rd-production": []byte(productionYAML),
		},
		exist: map[string]bool{
			"overlays/rd-staging":    true,
			"overlays/rd-production": true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{stagingPath + "/host-config.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Production))
	g.Expect(result.AffectedClusters).To(HaveKey("stone-stg-rh01"))
	g.Expect(result.AffectedClusters).NotTo(HaveKey("kflux-lw-p01"))
	g.Expect(result.AffectedClusters).NotTo(HaveKey("empty-base"))
}

func TestDetect_RemovedOverlay(t *testing.T) {
	g := NewWithT(t)

	// HEAD has no overlays (all removed), base had "development".
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {}},
		exist: map[string]bool{}, // nothing exists on head
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development"}},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"something.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

// ---------------------------------------------------------------------------
// matchByPrefix
// ---------------------------------------------------------------------------

func TestMatchByPrefix_FileUnderDir(t *testing.T) {
	g := NewWithT(t)
	changed := map[string]bool{"components/foo/staging/deploy.yaml": true}
	g.Expect(matchByPrefix(changed, "components/foo/staging")).To(BeTrue())
}

func TestMatchByPrefix_ExactFile(t *testing.T) {
	g := NewWithT(t)
	// The changed file IS the prefix itself (e.g. a single-file component).
	changed := map[string]bool{"components/foo/staging": true}
	g.Expect(matchByPrefix(changed, "components/foo/staging")).To(BeTrue())
}

func TestMatchByPrefix_TrailingSlash(t *testing.T) {
	g := NewWithT(t)
	// pathPrefix already ends with "/".
	changed := map[string]bool{"configs/x/a.yaml": true}
	g.Expect(matchByPrefix(changed, "configs/x/")).To(BeTrue())
}

func TestMatchByPrefix_NoMatch(t *testing.T) {
	g := NewWithT(t)
	changed := map[string]bool{"components/bar/staging/deploy.yaml": true}
	g.Expect(matchByPrefix(changed, "components/foo/staging")).To(BeFalse())
}

func TestMatchByPrefix_SimilarPrefixNoMatch(t *testing.T) {
	g := NewWithT(t)
	// "components/foo-extra/..." should NOT match prefix "components/foo"
	// because matchByPrefix adds "/" before checking.
	changed := map[string]bool{"components/foo-extra/deploy.yaml": true}
	g.Expect(matchByPrefix(changed, "components/foo")).To(BeFalse())
}

func TestMatchByPrefix_EmptyChangedSet(t *testing.T) {
	g := NewWithT(t)
	g.Expect(matchByPrefix(map[string]bool{}, "components/foo")).To(BeFalse())
}

// ---------------------------------------------------------------------------
// matchDepTree
// ---------------------------------------------------------------------------

func TestMatchDepTree_Match(t *testing.T) {
	g := NewWithT(t)
	changed := map[string]bool{
		"components/foo/staging/kustomization.yaml": true,
		"README.md": true,
	}
	deps := map[string]bool{
		"components/foo/staging/kustomization.yaml": true,
		"components/foo/staging/deploy.yaml":        true,
		"components/foo/base/deploy.yaml":           true,
	}
	g.Expect(matchDepTree(changed, deps)).To(BeTrue())
}

func TestMatchDepTree_NoMatch(t *testing.T) {
	g := NewWithT(t)
	changed := map[string]bool{"README.md": true}
	deps := map[string]bool{"components/foo/staging/deploy.yaml": true}
	g.Expect(matchDepTree(changed, deps)).To(BeFalse())
}

func TestMatchDepTree_EmptySets(t *testing.T) {
	g := NewWithT(t)
	g.Expect(matchDepTree(map[string]bool{}, map[string]bool{"a": true})).To(BeFalse())
	g.Expect(matchDepTree(map[string]bool{"a": true}, map[string]bool{})).To(BeFalse())
}

// ---------------------------------------------------------------------------
// matchClusters
// ---------------------------------------------------------------------------

func TestMatchClusters_DirectClusterDir(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedClusters: make(map[string]bool)}
	cp := appset.ComponentPath{Path: "components/foo/staging/stone-prod-p01", ClusterDir: "stone-prod-p01"}

	matchClusters(cp, nil, result)

	g.Expect(result.AffectedClusters).To(HaveKey("stone-prod-p01"))
}

func TestMatchClusters_ReservedDirBase(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedClusters: make(map[string]bool)}
	cp := appset.ComponentPath{Path: "components/foo/staging/base", ClusterDir: "base"}

	// "base" is a kustomize convention — should NOT be added as a cluster.
	// Instead, it should fall through to the allClusters lookup.
	allClusters := map[string][]string{
		"stone-prod-p01": {"components/foo/staging/stone-prod-p01"},
	}
	matchClusters(cp, allClusters, result)

	g.Expect(result.AffectedClusters).NotTo(HaveKey("base"))
	// The path "components/foo/staging/base" doesn't match any cluster path,
	// so no clusters should be added.
	g.Expect(result.AffectedClusters).To(BeEmpty())
}

func TestMatchClusters_ReservedDirEmptyBase(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedClusters: make(map[string]bool)}
	cp := appset.ComponentPath{
		Path:       "components/foo/rings/empty-base/empty-base",
		ClusterDir: "empty-base",
	}

	matchClusters(cp, nil, result)

	g.Expect(result.AffectedClusters).NotTo(HaveKey("empty-base"))
}

func TestMatchClusters_ReservedDirOverlay(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedClusters: make(map[string]bool)}
	cp := appset.ComponentPath{Path: "components/foo/staging/overlay", ClusterDir: "overlay"}

	matchClusters(cp, nil, result)

	g.Expect(result.AffectedClusters).NotTo(HaveKey("overlay"))
}

func TestMatchClusters_EmptyClusterDir_LookupFromMap(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedClusters: make(map[string]bool)}
	cp := appset.ComponentPath{Path: "components/foo/staging"}

	allClusters := map[string][]string{
		"stone-prod-p01": {"components/foo/staging/stone-prod-p01"},
		"kflux-ocp-p01":  {"components/foo/staging/kflux-ocp-p01"},
		"base":           {"components/foo/staging/base"}, // reserved, should be skipped
	}
	matchClusters(cp, allClusters, result)

	g.Expect(result.AffectedClusters).To(HaveKey("stone-prod-p01"))
	g.Expect(result.AffectedClusters).To(HaveKey("kflux-ocp-p01"))
	g.Expect(result.AffectedClusters).NotTo(HaveKey("base"))
}

func TestMatchClusters_EmptyClusterDir_NoMatchingPaths(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedClusters: make(map[string]bool)}
	cp := appset.ComponentPath{Path: "components/bar/staging"}

	allClusters := map[string][]string{
		"stone-prod-p01": {"components/foo/staging/stone-prod-p01"},
	}
	matchClusters(cp, allClusters, result)

	g.Expect(result.AffectedClusters).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// detectOverlayDiffs
// ---------------------------------------------------------------------------

func TestDetectOverlayDiffs_Changed(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}

	builds := []overlayBuild{
		{name: "development", env: Development, headYAML: []byte("a"), baseYAML: []byte("b")},
		{name: "staging-downstream", env: Staging, headYAML: []byte("same"), baseYAML: []byte("same")},
	}
	detectOverlayDiffs(builds, result)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Staging))
}

func TestDetectOverlayDiffs_NewOverlay(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}

	// A new overlay has headYAML but empty baseYAML (didn't exist before).
	builds := []overlayBuild{
		{name: "konflux-public-production", env: Production, headYAML: []byte("new content"), baseYAML: nil},
	}
	detectOverlayDiffs(builds, result)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
}

// TestDetectOverlayDiffs_BaseAppSetWithExplicitEnv covers the case where a new
// ApplicationSet with environment:staging is added to argo-cd-apps/base and
// therefore appears in ALL overlays' rendered output.  Only staging should be
// marked — not development or production.
func TestDetectOverlayDiffs_BaseAppSetWithExplicitEnv(t *testing.T) {
	g := NewWithT(t)

	const stagingAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: kanary
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/monitoring/kanary
                environment: staging
                clusterDir: ""
          - list:
              elements:
                - nameNormalized: lightwell-dev
                  values.clusterDir: lightwell-dev
  template:
    metadata:
      name: kanary-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}
	// Simulate base change: the new AppSet appears in all overlays (dev, staging, prod),
	// but none of them had it before (baseYAML is empty).
	builds := []overlayBuild{
		{name: "development", env: Development, headYAML: []byte(stagingAppSet), baseYAML: []byte{}},
		{name: "staging-downstream", env: Staging, headYAML: []byte(stagingAppSet), baseYAML: []byte{}},
		{name: "production-downstream", env: Production, headYAML: []byte(stagingAppSet), baseYAML: []byte{}},
	}
	detectOverlayDiffs(builds, result)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Development))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Production))
}

func TestDetectOverlayDiffs_RemovedBaseAppSetUsesGeneratorEnv(t *testing.T) {
	g := NewWithT(t)

	const stagingAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: kanary
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/monitoring/kanary
                environment: staging
                clusterDir: ""
  template:
    metadata:
      name: kanary-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}
	// AppSet existed on base (all overlays) but was removed on HEAD.
	builds := []overlayBuild{
		{name: "development", env: Development, headYAML: []byte{}, baseYAML: []byte(stagingAppSet)},
		{name: "staging-downstream", env: Staging, headYAML: []byte{}, baseYAML: []byte(stagingAppSet)},
		{name: "production-downstream", env: Production, headYAML: []byte{}, baseYAML: []byte(stagingAppSet)},
	}
	detectOverlayDiffs(builds, result)

	// Should use generator env (staging), not the overlay's env.
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Development))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Production))
}

func TestDetectOverlayDiffs_ParseErrorFallsBackToOverlayEnv(t *testing.T) {
	g := NewWithT(t)

	result := &Result{AffectedEnvironments: make(map[Environment]bool)}
	// Invalid YAML on HEAD — parse fails, should fall back to overlay env.
	builds := []overlayBuild{
		{name: "staging-downstream", env: Staging, headYAML: []byte("not: valid: yaml: [}"), baseYAML: []byte{}},
	}
	detectOverlayDiffs(builds, result)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
}

func TestDetectOverlayDiffs_NoChanges(t *testing.T) {
	g := NewWithT(t)
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}

	builds := []overlayBuild{
		{name: "development", env: Development, headYAML: []byte("same"), baseYAML: []byte("same")},
		{name: "staging-downstream", env: Staging, headYAML: []byte("same"), baseYAML: []byte("same")},
	}
	detectOverlayDiffs(builds, result)

	g.Expect(result.AffectedEnvironments).To(BeEmpty())
}

func TestDetectOverlayDiffs_BaseAppSetWithProductionEnv(t *testing.T) {
	g := NewWithT(t)

	const prodAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: my-prod-component
spec:
  generators:
    - merge:
        mergeKeys:
          - nameNormalized
        generators:
          - clusters:
              values:
                sourceRoot: components/my-app
                environment: production
                clusterDir: ""
          - list:
              elements:
                - nameNormalized: stone-prod-p01
                  values.clusterDir: stone-prod-p01
  template:
    metadata:
      name: prod-{{nameNormalized}}
    spec:
      source:
        path: '{{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}
	builds := []overlayBuild{
		{name: "development", env: Development, headYAML: []byte(prodAppSet), baseYAML: []byte{}},
		{name: "staging-downstream", env: Staging, headYAML: []byte(prodAppSet), baseYAML: []byte{}},
		{name: "production-downstream", env: Production, headYAML: []byte(prodAppSet), baseYAML: []byte{}},
	}
	detectOverlayDiffs(builds, result)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Development))
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Staging))
}

// TestDetectOverlayDiffs_MultiEnvWarning verifies that when a base AppSet has
// no explicit environment (targets all clusters), all environments are marked —
// which is the expected "multi-env" signal.
func TestDetectOverlayDiffs_MultiEnvWarning(t *testing.T) {
	g := NewWithT(t)

	// AppSet with clusters: {} (no values.environment) targets all clusters.
	const allEnvsAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: global-component
spec:
  generators:
    - clusters: {}
  template:
    metadata:
      name: global-{{nameNormalized}}
    spec:
      source:
        path: components/global
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
      destination:
        server: '{{server}}'
`
	result := &Result{AffectedEnvironments: make(map[Environment]bool)}
	builds := []overlayBuild{
		{name: "development", env: Development, headYAML: []byte(allEnvsAppSet), baseYAML: []byte{}},
		{name: "staging-downstream", env: Staging, headYAML: []byte(allEnvsAppSet), baseYAML: []byte{}},
		{name: "production-downstream", env: Production, headYAML: []byte(allEnvsAppSet), baseYAML: []byte{}},
	}
	detectOverlayDiffs(builds, result)

	// No explicit environment → each overlay falls back to its own env → multi-env.
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
}

// ---------------------------------------------------------------------------
// extractPathsFromOverlays
// ---------------------------------------------------------------------------

func TestExtractPathsFromOverlays_Basic(t *testing.T) {
	g := NewWithT(t)

	yaml := appSetWithCluster("components/test", "staging", "stone-prod-p01")
	builds := []overlayBuild{
		{name: "staging-downstream", env: Staging, headYAML: []byte(yaml)},
	}

	envPaths, allClusters, _, err := extractPathsFromOverlays(builds)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(envPaths).To(HaveKey(Staging))
	g.Expect(envPaths[Staging]).NotTo(BeEmpty())

	// Should have extracted paths
	var paths []string
	for _, cp := range envPaths[Staging] {
		paths = append(paths, cp.Path)
	}
	g.Expect(paths).To(ContainElement("components/test/staging"))
	g.Expect(paths).To(ContainElement("components/test/staging/stone-prod-p01"))

	// Cluster should be extracted
	g.Expect(allClusters).To(HaveKey("stone-prod-p01"))
}

func TestExtractPathsFromOverlays_RingTemplateAttributedToOverlayEnv(t *testing.T) {
	g := NewWithT(t)

	stagingYAML := appSetWithRing(
		"components/multi-platform-controller",
		"empty-base", "empty-base",
		"ring-1", "stone-stg-rh01",
	)
	productionYAML := appSetWithRing(
		"components/multi-platform-controller",
		"empty-base", "empty-base",
		"ring-2", "kflux-lw-p01",
	)
	builds := []overlayBuild{
		{name: "rd-staging", env: Staging, headYAML: []byte(stagingYAML)},
		{name: "rd-production", env: Production, headYAML: []byte(productionYAML)},
	}

	envPaths, allClusters, _, err := extractPathsFromOverlays(builds)
	g.Expect(err).NotTo(HaveOccurred())

	stagingPaths := make([]string, 0, len(envPaths[Staging]))
	for _, cp := range envPaths[Staging] {
		stagingPaths = append(stagingPaths, cp.Path)
	}
	productionPaths := make([]string, 0, len(envPaths[Production]))
	for _, cp := range envPaths[Production] {
		productionPaths = append(productionPaths, cp.Path)
	}

	g.Expect(stagingPaths).To(ContainElement("components/multi-platform-controller/rings/ring-1/stone-stg-rh01"))
	g.Expect(stagingPaths).NotTo(ContainElement("components/multi-platform-controller/rings/ring-2/kflux-lw-p01"))
	g.Expect(productionPaths).To(ContainElement("components/multi-platform-controller/rings/ring-2/kflux-lw-p01"))
	g.Expect(productionPaths).NotTo(ContainElement("components/multi-platform-controller/rings/ring-1/stone-stg-rh01"))

	g.Expect(allClusters).To(HaveKey("stone-stg-rh01"))
	g.Expect(allClusters).To(HaveKey("kflux-lw-p01"))
}

func TestExtractPathsFromOverlays_MultipleEnvs(t *testing.T) {
	g := NewWithT(t)

	builds := []overlayBuild{
		{name: "staging-downstream", env: Staging, headYAML: []byte(minimalAppSetYAML)},
		{name: "production-downstream", env: Production, headYAML: []byte(minimalAppSetYAML)},
	}

	envPaths, _, _, err := extractPathsFromOverlays(builds)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(envPaths).To(HaveKey(Staging))
	g.Expect(envPaths).To(HaveKey(Production))
}

func TestExtractPathsFromOverlays_InvalidYAML(t *testing.T) {
	g := NewWithT(t)

	builds := []overlayBuild{
		{name: "bad", env: Development, headYAML: []byte("not: valid: yaml: [}")},
	}

	_, _, _, err := extractPathsFromOverlays(builds)
	g.Expect(err).To(HaveOccurred())
}

// ---------------------------------------------------------------------------
// matchChangedFiles (integration: dep tree + prefix + clusters)
// ---------------------------------------------------------------------------

func TestMatchChangedFiles_DepTreeMatch(t *testing.T) {
	g := NewWithT(t)
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}

	resolved := []componentDeps{
		{
			cp:  appset.ComponentPath{Path: "components/foo/staging"},
			env: Staging,
			deps: map[string]bool{
				"components/foo/staging/kustomization.yaml": true,
				"components/foo/base/deploy.yaml":           true,
			},
		},
	}

	matchChangedFiles(
		[]string{"components/foo/base/deploy.yaml"},
		resolved, nil, result,
	)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
}

func TestMatchChangedFiles_PrefixFallback(t *testing.T) {
	g := NewWithT(t)
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}

	// deps is nil → prefix matching
	resolved := []componentDeps{
		{
			cp:   appset.ComponentPath{Path: "configs/plain-dir"},
			env:  Production,
			deps: nil,
		},
	}

	matchChangedFiles(
		[]string{"configs/plain-dir/manifest.yaml"},
		resolved, nil, result,
	)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
}

func TestMatchChangedFiles_NoMatch(t *testing.T) {
	g := NewWithT(t)
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}

	resolved := []componentDeps{
		{
			cp:  appset.ComponentPath{Path: "components/foo/staging"},
			env: Staging,
			deps: map[string]bool{
				"components/foo/staging/kustomization.yaml": true,
			},
		},
	}

	matchChangedFiles(
		[]string{"README.md"},
		resolved, nil, result,
	)

	g.Expect(result.AffectedEnvironments).To(BeEmpty())
	g.Expect(result.AffectedClusters).To(BeEmpty())
}

func TestMatchChangedFiles_WithClusters(t *testing.T) {
	g := NewWithT(t)
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}

	resolved := []componentDeps{
		{
			cp:  appset.ComponentPath{Path: "components/foo/staging/stone-prod-p01", ClusterDir: "stone-prod-p01"},
			env: Staging,
			deps: map[string]bool{
				"components/foo/staging/stone-prod-p01/kustomization.yaml": true,
			},
		},
	}

	matchChangedFiles(
		[]string{"components/foo/staging/stone-prod-p01/kustomization.yaml"},
		resolved, nil, result,
	)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedClusters).To(HaveKey("stone-prod-p01"))
}

// ---------------------------------------------------------------------------
// applyStaticRules
// ---------------------------------------------------------------------------

func TestStaticRule_AppOfAppSets(t *testing.T) {
	g := NewWithT(t)

	// Any change under app-of-app-sets should affect all environments.
	changedFiles := []string{"argo-cd-apps/app-of-app-sets/production/change-source-path.yaml"}

	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
		ChangedFiles:         changedFiles,
	}

	applyStaticRules(changedFiles, result)

	for _, env := range []Environment{Development, Staging, Production} {
		g.Expect(result.AffectedEnvironments).To(HaveKey(env), "expected environment %q to be affected", env)
	}
}

func TestStaticRule_NonManagedFile(t *testing.T) {
	g := NewWithT(t)

	// Files outside components/, configs/, argo-cd-apps/ should NOT affect any
	// environment. Only the overlay diff and dep tree matching decide that.
	changedFiles := []string{"README.md"}

	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
		ChangedFiles:         changedFiles,
	}

	applyStaticRules(changedFiles, result)

	g.Expect(result.AffectedEnvironments).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

func TestResultLabels(t *testing.T) {
	g := NewWithT(t)

	result := &Result{
		AffectedEnvironments: map[Environment]bool{
			Production:  true,
			Development: true,
		},
		AffectedClusters: map[string]bool{
			"stone-prod-p01": true,
		},
	}

	ls := result.Labels()

	g.Expect(ls.Environments).To(Equal([]string{
		"environment/development",
		"environment/production",
	}))
	g.Expect(ls.Clusters).To(Equal([]string{
		"cluster/stone-prod-p01",
	}))

	// All() should yield everything sorted together
	g.Expect(slices.Collect(ls.All())).To(Equal([]string{
		"cluster/stone-prod-p01",
		"environment/development",
		"environment/production",
	}))
}

// ---------------------------------------------------------------------------
// ApplicationSet shapes that were previously dropped silently.
//
// See the parser-level tests in internal/appset/parse_test.go and the
// end-to-end guard in internal/appset/audit_test.go.
// ---------------------------------------------------------------------------

// topLevelListAppSet is the shape used by konflux-verifications-rd in the
// rd-production overlay: a single list generator with per-element values.
const topLevelListAppSet = `apiVersion: argoproj.io/v1alpha1
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

// A production component reached through a top-level list generator must be
// detected as a production change (which also drives the hold labels).
func TestDetect_TopLevelListGenerator(t *testing.T) {
	g := NewWithT(t)

	const componentPath = "components/konflux-verifications-rd/rings/ring-2/stone-prod-p01"

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"rd-production"}},
		yamls: map[string][]byte{"overlays/rd-production": []byte(topLevelListAppSet)},
		exist: map[string]bool{
			"overlays/rd-production": true,
			componentPath:            true,
		},
		deps: map[string]map[string]bool{
			componentPath: {componentPath + "/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"rd-production"}},
		yamls: map[string][]byte{"overlays/rd-production": []byte(topLevelListAppSet)},
		exist: map[string]bool{"overlays/rd-production": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{componentPath + "/kustomization.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
	g.Expect(result.AffectedClusters).To(HaveKey("stone-prod-p01"))
}

// A prefix-only component path (unresolved template) is matched by prefix: the
// directory it was derived from may not even exist.
func TestResolveComponentDeps_PrefixOnlyMatchesByPrefix(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		exist: map[string]bool{"overlays/staging-downstream": true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		exist: map[string]bool{"overlays/staging-downstream": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Staging: {{Path: "components/partial/staging", PrefixOnly: true}},
	})
	g.Expect(resolved).To(HaveLen(1))

	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
		ChangedFiles:         []string{"components/partial/staging/base/deploy.yaml"},
	}
	matchChangedFiles(
		[]string{"components/partial/staging/base/deploy.yaml"},
		resolved, nil, result,
	)

	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
}

// A prefix-only component path also follows the dependency trees of the
// kustomizations under the prefix: a cluster overlay referencing a shared base
// outside the prefix (the backup ApplicationSet's ../../base/member) must make
// a base-only change affect the environment.
func TestResolveComponentDeps_PrefixOnlyFollowsDepsUnderPrefix(t *testing.T) {
	g := NewWithT(t)

	prefix := "components/backup/production"
	clusterDir := prefix + "/stone-prod-p01"
	nestedDir := clusterDir + "/extra"
	head := &fakeRepo{
		dirs: map[string][]string{
			"overlays": {"production-downstream"},
			prefix:     {"stone-prod-p01"},
			clusterDir: {"extra"},
			nestedDir:  {},
		},
		exist: map[string]bool{prefix: true},
		deps: map[string]map[string]bool{
			clusterDir: {
				clusterDir + "/kustomization.yaml":                 true,
				"components/backup/base/member/kustomization.yaml": true,
				"components/backup/base/member/schedule.yaml":      true,
			},
			nestedDir: {
				nestedDir + "/kustomization.yaml":         true,
				"components/backup/base/extra/patch.yaml": true,
			},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"production-downstream"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Production: {{Path: prefix, PrefixOnly: true}},
	})
	g.Expect(resolved).To(HaveLen(1))

	for _, changed := range []string{
		"components/backup/base/member/schedule.yaml", // first-level kustomization
		"components/backup/base/extra/patch.yaml",     // nested kustomization
		prefix + "/stone-prod-p01/new-file.yaml",      // under the prefix
	} {
		result := &Result{
			AffectedEnvironments: make(map[Environment]bool),
			AffectedClusters:     make(map[string]bool),
		}
		matchChangedFiles([]string{changed}, resolved, nil, result)
		g.Expect(result.AffectedEnvironments).To(HaveKey(Production), changed)
	}

	// Files outside the prefix and its dependency trees still do not match.
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}
	matchChangedFiles([]string{"components/backup/base/unused/x.yaml"}, resolved, nil, result)
	g.Expect(result.AffectedEnvironments).To(BeEmpty())
}

// If HEAD turned a kustomization into plain manifests, a new plain manifest
// must still mark the environment affected even though the base ref supplied a
// dependency tree.
func TestDetect_PlainManifestOnHeadUsesPrefixFallback(t *testing.T) {
	g := NewWithT(t)

	const changed = "components/foo/new-plain.yaml"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/kustomization.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{changed})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

// An incomplete dependency tree (unreadable generator or chart subtree) must
// still use prefix matching as well as the dependencies found so far.
func TestDetect_PartialDepTreeUsesPrefixFallback(t *testing.T) {
	g := NewWithT(t)

	const changed = "components/foo/new-chart-value.yaml"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/kustomization.yaml": true},
		},
		depsErr: map[string]error{"components/foo": deptree.ErrIncomplete},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/kustomization.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{changed})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

// A prefix-only path may exist only on base. Its dependency trees must still be
// walked there, so a dropped shared base is visible.
func TestResolveComponentDeps_PrefixOnlyUsesBaseDeps(t *testing.T) {
	g := NewWithT(t)

	const prefix = "components/backup/production"
	const clusterDir = prefix + "/stone-prod-p01"
	const shared = "components/backup/base/member/schedule.yaml"

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		exist: map[string]bool{prefix: true},
	}
	base := &fakeRepo{
		dirs: map[string][]string{
			"overlays": {"production-downstream"},
			prefix:     {"stone-prod-p01"},
			clusterDir: {},
		},
		exist: map[string]bool{prefix: true},
		deps: map[string]map[string]bool{
			clusterDir: {
				clusterDir + "/kustomization.yaml": true,
				shared:                             true,
			},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Production: {{Path: prefix, PrefixOnly: true}},
	})
	g.Expect(resolved).To(HaveLen(1))

	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}
	matchChangedFiles([]string{shared}, resolved, nil, result)
	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
}

// A MatchAll entry (unresolvable template without a static prefix) fails
// closed: any change affects its environment, without attributing clusters,
// and detection itself does not fail.
func TestDetect_MatchAllMarksEnvironmentOnAnyChange(t *testing.T) {
	g := NewWithT(t)

	appSet := []byte(`apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: unresolvable
spec:
  generators:
    - git:
        repoURL: https://example.com/repo.git
  template:
    metadata:
      name: unresolvable
    spec:
      source:
        path: '{{values.sourceRoot}}/production'
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`)
	overlay := "overlays/production-downstream"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		exist: map[string]bool{overlay: true},
		yamls: map[string][]byte{overlay: appSet},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		exist: map[string]bool{overlay: true},
		yamls: map[string][]byte{overlay: appSet},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/unrelated/staging/kustomization.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
	g.Expect(result.AffectedClusters).To(BeEmpty())
}

// A source whose directory is not in this repository (e.g. an external-repo
// path) may still consume "$values/" files from it. Those files must keep
// matching exactly even though the directory is missing.
func TestResolveComponentDeps_MissingDirKeepsExtraFiles(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		exist: map[string]bool{}, // the source path does NOT exist here
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"staging-downstream"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	valuesFile := "components/chart/staging/values.yaml"
	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Staging: {{Path: "charts/external", ExtraFiles: []string{valuesFile}}},
	})
	g.Expect(resolved).To(HaveLen(1))
	g.Expect(resolved[0].cp.Path).To(BeEmpty())
	g.Expect(resolved[0].cp.ExtraFiles).To(ConsistOf(valuesFile))

	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}
	matchChangedFiles([]string{valuesFile}, resolved, nil, result)
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
}

// A resolved entry and a prefix-only entry sharing the same Path must both be
// kept, otherwise the resolved one could narrow prefix matching to its
// dependency tree.
func TestResolveComponentDeps_PrefixOnlyNotShadowedByResolved(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		exist: map[string]bool{"components/foo/staging": true},
		deps: map[string]map[string]bool{
			"components/foo/staging": {"components/foo/staging/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"staging-downstream"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Staging: {
			{Path: "components/foo/staging"},
			{Path: "components/foo/staging", PrefixOnly: true},
		},
	})
	g.Expect(resolved).To(HaveLen(2))

	// A file outside the dependency tree but under the prefix still matches.
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}
	matchChangedFiles([]string{"components/foo/staging/stone-stage-p01/patch.yaml"}, resolved, nil, result)
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
}

// A prefix-only entry is shared by several clusters: it must not label every
// cluster whose paths live under the prefix.
func TestMatchClusters_PrefixOnlyDoesNotLabelClustersUnderPrefix(t *testing.T) {
	g := NewWithT(t)
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
	}
	allClusters := map[string][]string{
		"stone-stage-p01": {"components/foo/staging/stone-stage-p01"},
		"kflux-stg-p02":   {"components/foo/staging/kflux-stg-p02"},
	}

	matchClusters(appset.ComponentPath{Path: "components/foo/staging", PrefixOnly: true}, allClusters, result)
	g.Expect(result.AffectedClusters).To(BeEmpty())

	// A resolved entry with the same path keeps the cluster fallback.
	matchClusters(appset.ComponentPath{Path: "components/foo/staging"}, allClusters, result)
	g.Expect(result.AffectedClusters).To(HaveLen(2))
}

// multiSourceAppSetYAML is a chart-only multi-source ApplicationSet whose only
// file in this repository is a "$values/" Helm value file.
const multiSourceAppSetYAML = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: chart
spec:
  generators:
    - clusters: {}
  template:
    spec:
      sources:
        - chart: some-chart
          repoURL: https://charts.example.com
          targetRevision: 1.0.0
          helm:
            valueFiles:
              - $values/components/chart/staging/values.yaml
        - ref: values
          repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`

// A change to a "$values/" Helm value file of a multi-source ApplicationSet,
// with nothing else changed, marks the overlay's environment through Detect;
// AffectedComponents has no deployed directory to return for render-diff.
func TestDetect_MultiSourceValueFileChange(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		yamls: map[string][]byte{"overlays/staging-downstream": []byte(multiSourceAppSetYAML)},
		exist: map[string]bool{"overlays/staging-downstream": true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		yamls: map[string][]byte{"overlays/staging-downstream": []byte(multiSourceAppSetYAML)}, // no overlay diff
		exist: map[string]bool{"overlays/staging-downstream": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	changed := []string{"components/chart/staging/values.yaml"}
	result, err := d.Detect(changed)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))

	affected, err := d.AffectedComponents(changed)
	g.Expect(err).NotTo(HaveOccurred())
	// A value-file-only match has no directory for render-diff to build, so it
	// is used for environment labels by Detect but not returned here.
	g.Expect(affected[Staging]).To(BeEmpty())

	// An unrelated file does not match the value file.
	result, err = d.Detect([]string{"components/chart/staging/other.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).NotTo(HaveKey(Staging))
}

// Entries sharing a Path but carrying different ExtraFiles must not be
// collapsed by the dedup key: each keeps its own value files and matches
// independently.
func TestResolveComponentDeps_SamePathDifferentExtraFilesKept(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		exist: map[string]bool{"components/chart/staging": true},
		deps: map[string]map[string]bool{
			"components/chart/staging": {"components/chart/staging/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"staging-downstream"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	valuesA := "components/chart/values-a.yaml"
	valuesB := "components/chart/values-b.yaml"
	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Staging: {
			{Path: "components/chart/staging", ExtraFiles: []string{valuesA}},
			{Path: "components/chart/staging", ExtraFiles: []string{valuesB}},
		},
	})
	g.Expect(resolved).To(HaveLen(2))

	for _, changed := range []string{valuesA, valuesB} {
		changedSet := map[string]bool{changed: true}
		matching := 0
		for _, cd := range resolved {
			if cd.matchesAny(changedSet) {
				matching++
				g.Expect(cd.cp.ExtraFiles).To(ConsistOf(changed))
			}
		}
		g.Expect(matching).To(Equal(1), "exactly one entry should match %s", changed)
	}
}

// Entries that share a Path but target different clusters must both survive
// the detector dedup key so matchClusters can label each one.
func TestResolveComponentDeps_KeepsSamePathDifferentClusterDir(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		exist: map[string]bool{"components/chart/staging": true},
		deps: map[string]map[string]bool{
			"components/chart/staging": {"components/chart/staging/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"staging-downstream"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	resolved := d.resolveComponentDeps(map[Environment][]appset.ComponentPath{
		Staging: {
			{Path: "components/chart/staging", ClusterDir: "stone-stage-p01"},
			{Path: "components/chart/staging", ClusterDir: "kflux-stg-p02"},
		},
	})
	g.Expect(resolved).To(HaveLen(2))
	clusters := []string{resolved[0].cp.ClusterDir, resolved[1].cp.ClusterDir}
	g.Expect(clusters).To(ConsistOf("stone-stage-p01", "kflux-stg-p02"))
}

// AffectedComponents feeds render-diff, which can only build a deployed
// directory. MatchAll, PrefixOnly and value-file-only matches are filtered, and
// duplicate paths are collapsed.
func TestMatchAffectedComponents_FiltersNonRenderableAndDedupes(t *testing.T) {
	g := NewWithT(t)

	resolved := []componentDeps{
		{env: Staging, cp: appset.ComponentPath{Path: "components/foo/staging", ClusterDir: "cluster-a"}},
		{env: Staging, cp: appset.ComponentPath{
			Path: "components/foo/staging", ClusterDir: "cluster-a",
			ExtraFiles: []string{"components/foo/values.yaml"},
		}},
		{env: Staging, cp: appset.ComponentPath{Path: "components/foo/staging", PrefixOnly: true}},
		{env: Staging, cp: appset.ComponentPath{MatchAll: true}},
		{env: Staging, cp: appset.ComponentPath{ExtraFiles: []string{"components/foo/values.yaml"}}},
	}

	affected := matchAffectedComponents(
		[]string{"components/foo/staging/kustomization.yaml", "components/foo/values.yaml"},
		resolved, nil,
	)

	g.Expect(affected[Staging]).To(HaveLen(1))
	g.Expect(affected[Staging][0].Path).To(Equal("components/foo/staging"))
}

// prefixDeps never walks the repository root, an absolute path or a path
// escaping the checkout, even if such a prefix reaches the detector.
func TestPrefixDeps_RejectsPathsOutsideRepo(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}, ".": {"components"}, "..": {"x"}},
		exist: map[string]bool{".": true, "..": true, "/etc": true, "../../..": true},
		deps: map[string]map[string]bool{
			".":        {"kustomization.yaml": true},
			"..":       {"../kustomization.yaml": true},
			"/etc":     {"/etc/kustomization.yaml": true},
			"../../..": {"../../../kustomization.yaml": true},
		},
	}
	base := &fakeRepo{dirs: map[string][]string{"overlays": {"development"}}}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	for _, prefix := range []string{".", "..", "/etc", "../../.."} {
		g.Expect(d.prefixDeps(prefix)).To(BeNil(), "prefix %q", prefix)
	}
}

// ---------------------------------------------------------------------------
// Unreferenced changes
// ---------------------------------------------------------------------------

// unreferencedDetector builds a detector whose only ApplicationSet deploys
// components/foo, with foo/deploy.yaml as the sole dependency. files are the
// changed paths that exist on HEAD.
func unreferencedDetector(t *testing.T, files ...string) *Detector {
	t.Helper()

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"components/foo": true, "overlays/development": true},
		files: fileSet,
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	return d
}

// A change in a component directory that no ApplicationSet references must be
// reported as unreferenced, not silently treated as "no impact".
func TestDetect_UnreferencedComponentChange(t *testing.T) {
	g := NewWithT(t)

	const orphan = "components/deleted-component/staging/kflux-stg-p02/deploy.yaml"
	d := unreferencedDetector(t, orphan)

	result, err := d.Detect([]string{orphan})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(BeEmpty())
	g.Expect(result.UnreferencedFiles).To(ConsistOf(orphan))
}

// A matched change is referenced, so it must not be reported.
func TestDetect_ReferencedChangeIsNotUnreferenced(t *testing.T) {
	g := NewWithT(t)

	d := unreferencedDetector(t, "components/foo/deploy.yaml")

	result, err := d.Detect([]string{"components/foo/deploy.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	g.Expect(result.UnreferencedFiles).To(BeEmpty())
}

// Files outside components/ and configs/, and documentation/ownership files,
// are not reported: they are not deployable manifests by design.
func TestDetect_UnreferencedIgnoresNonDeployableFiles(t *testing.T) {
	g := NewWithT(t)

	files := []string{
		"README.md",
		"hack/foo.sh",
		"components/foo/README.md",
		"components/foo/OWNERS",
		"components/foo/diagram.png",
		"components/policies/base/foo/.chainsaw-test/chainsaw-test.yaml",
		"components/konflux-operator/ci/openshift-overlay-e2e/install.sh",
		"components/konflux-info/banner-schema.json",
	}
	d := unreferencedDetector(t, files...)

	result, err := d.Detect(files)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.UnreferencedFiles).To(BeEmpty())
}

func TestIsDeployableFile(t *testing.T) {
	g := NewWithT(t)

	cases := map[string]bool{
		"components/foo/deploy.yaml":                                      true,
		"configs/foo/kustomization.yaml":                                  true,
		"components/foo/README.md":                                        false,
		"components/foo/OWNERS":                                           false,
		"components/foo/diagram.png":                                      false,
		"components/policies/base/foo/.chainsaw-test/chainsaw-test.yaml":  false,
		"components/konflux-operator/ci/openshift-overlay-e2e/install.sh": false,
		"components/konflux-info/banner-schema.json":                      false,
		"components/kubearchive/upgrade.sh":                               false,
		"components/openshift-gitops/base/deploy.yaml":                    false,
	}
	for file, want := range cases {
		g.Expect(isDeployableFile(file)).To(Equal(want), "file %q", file)
	}
}

// Tier 1 base files are copied into each ring's base-snapshot by Kargo, so
// they are deployed even when no kustomization references them directly.
func TestDetect_RingBaseChangeIsReferenced(t *testing.T) {
	g := NewWithT(t)

	const file = "components/foo/base/deploy.yaml"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{
			"components/foo":       true,
			"components/foo/rings": true,
			"overlays/development": true,
		},
		files: map[string]bool{file: true},
		deps:  map[string]map[string]bool{"components/foo": {"components/foo/deploy.yaml": true}},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{file})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.UnreferencedFiles).To(BeEmpty())
}

func TestMatchChangedFiles_RecordsAllMatchKinds(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		name    string
		cp      appset.ComponentPath
		changed string
	}{
		{"match all", appset.ComponentPath{MatchAll: true}, "components/x/anything.yaml"},
		{"extra file", appset.ComponentPath{ExtraFiles: []string{"components/x/values.yaml"}}, "components/x/values.yaml"},
		{"prefix only", appset.ComponentPath{Path: "components/x", PrefixOnly: true}, "components/x/sub/file.yaml"},
		{"nil deps prefix", appset.ComponentPath{Path: "components/x"}, "components/x/sub/file.yaml"},
	}

	for _, tc := range cases {
		result := &Result{
			AffectedEnvironments: make(map[Environment]bool),
			AffectedClusters:     make(map[string]bool),
		}
		matched := matchChangedFiles(
			[]string{tc.changed},
			[]componentDeps{{cp: tc.cp, env: Development}},
			nil, result,
		)
		g.Expect(matched).To(HaveKey(tc.changed), tc.name)
		g.Expect(result.AffectedEnvironments).To(HaveKey(Development), tc.name)
	}
}

// A PR can contain both referenced and orphaned files.
func TestDetect_ReferencedAndUnreferencedTogether(t *testing.T) {
	g := NewWithT(t)

	const referenced = "components/foo/deploy.yaml"
	const orphan = "components/orphan/deploy.yaml"
	d := unreferencedDetector(t, referenced, orphan)

	result, err := d.Detect([]string{referenced, orphan})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	g.Expect(result.UnreferencedFiles).To(ConsistOf(orphan))
}

// Config files are deployable too: an orphaned config change is reported.
func TestDetect_UnreferencedConfigChange(t *testing.T) {
	g := NewWithT(t)

	const orphan = "configs/orphaned/staging/kustomization.yaml"
	d := unreferencedDetector(t, orphan)

	result, err := d.Detect([]string{orphan})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.UnreferencedFiles).To(ConsistOf(orphan))
}

// A deleted file was referenced on the base ref; it must not be reported as
// unreferenced merely because it no longer exists on HEAD.
func TestDetect_DeletedFileIsNotUnreferenced(t *testing.T) {
	g := NewWithT(t)

	d := unreferencedDetector(t)
	const deleted = "components/foo/deleted.yaml"

	result, err := d.Detect([]string{deleted})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.UnreferencedFiles).To(BeEmpty())
}

// ---------------------------------------------------------------------------
// Components that only exist on one ref
// ---------------------------------------------------------------------------

// A PR that removes a component no longer has its ApplicationSet on HEAD. The
// change must still be detected through the base ref, otherwise a removal looks
// like "no impact".
func TestDetect_ComponentRemovedInHead(t *testing.T) {
	g := NewWithT(t)

	const appSetYAML = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: foo
spec:
  generators:
    - clusters: {}
  template:
    metadata:
      name: foo
    spec:
      source:
        path: components/foo
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	// HEAD no longer renders any ApplicationSet.
	const emptyHead = `apiVersion: v1
kind: Namespace
metadata:
  name: placeholder
`

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(emptyHead)},
		exist: map[string]bool{"overlays/development": true},
		// The file still exists on HEAD, so it is a candidate for the
		// unreferenced warning: only the removed (base-only) ApplicationSet
		// references it.
		files: map[string]bool{"components/foo/deploy.yaml": true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(appSetYAML)},
		exist: map[string]bool{
			"overlays/development": true,
			"components/foo":       true,
		},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	const changed = "components/foo/deploy.yaml"

	result, err := d.Detect([]string{changed})
	g.Expect(err).NotTo(HaveOccurred())
	// Detect already labels Development through its removed-overlay diff, so
	// the environment label does not prove base-ref path extraction. The
	// meaningful removal checks are the unreferenced warning below and
	// AffectedComponents afterwards.
	// The file exists on HEAD but only the removed (base-only) ApplicationSet
	// references it, so it is dead after merge and must be reported as
	// unreferenced even though the removal diff is still rendered.
	g.Expect(result.UnreferencedFiles).To(ConsistOf(changed))

	// render-diff uses AffectedComponents, so the base-only component must be
	// returned there: the overlay diff alone does not name the component.
	affected, err := d.AffectedComponents([]string{changed})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Development]).To(HaveLen(1))
	g.Expect(affected[Development][0].Path).To(Equal("components/foo"))
}

// A file that HEAD still contains but no longer references is dead, even if the
// base ref still lists it. Base-only dependencies must not suppress the
// unreferenced warning for a component that exists on HEAD.
func TestDetect_HeadDroppedResourceIsUnreferenced(t *testing.T) {
	g := NewWithT(t)

	const changed = "components/foo/a.yaml"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		files: map[string]bool{changed: true},
		deps: map[string]map[string]bool{
			// HEAD dropped a.yaml from resources.
			"components/foo": {"components/foo/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {
				"components/foo/kustomization.yaml": true,
				changed:                             true,
			},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{changed})
	g.Expect(err).NotTo(HaveOccurred())
	// The union still attributes the change to the component's environment.
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
	// But HEAD no longer references it, so it is dead after merge.
	g.Expect(result.UnreferencedFiles).To(ConsistOf(changed))
}

// A PR can delete a whole overlay directory. Its ApplicationSets then exist
// only on the base ref; render-diff still needs their component paths so the
// removal is visible instead of silently producing no diff.
func TestAffectedComponents_OverlayRemovedInHead(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {}},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(minimalAppSetYAML)},
		exist: map[string]bool{
			"overlays/production-downstream": true,
			"components/foo":                 true,
		},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents([]string{"components/foo/deploy.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Production]).To(HaveLen(1))
	g.Expect(affected[Production][0].Path).To(Equal("components/foo"))
}

// An ApplicationSet can declare an environment that differs from the overlay
// rendering it. The overlay environment is the deployment target and must
// never be narrowed; the declared one is recorded in addition.
func TestAffectedComponents_DeclaredEnvironmentDoesNotNarrowOverlay(t *testing.T) {
	g := NewWithT(t)

	const stagingAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: staged
spec:
  generators:
    - clusters:
        values:
          environment: staging
  template:
    metadata:
      name: staged
    spec:
      source:
        path: components/foo
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {}},
	}
	base := &fakeRepo{
		dirs: map[string][]string{
			"overlays": {"development", "staging-downstream", "production-downstream"},
		},
		yamls: map[string][]byte{
			"overlays/development":           []byte(stagingAppSet),
			"overlays/staging-downstream":    []byte(stagingAppSet),
			"overlays/production-downstream": []byte(stagingAppSet),
		},
		exist: map[string]bool{
			"overlays/development":           true,
			"overlays/staging-downstream":    true,
			"overlays/production-downstream": true,
			"components/foo":                 true,
		},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents([]string{"components/foo/deploy.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	for _, env := range []Environment{Development, Staging, Production} {
		g.Expect(affected[env]).To(HaveLen(1), "environment %s", env)
		g.Expect(affected[env][0].Path).To(Equal("components/foo"))
	}
}

// Regression: a production overlay can include an unpatched base AppSet that
// still declares values.environment: staging. That value is a path-template
// variable, not the deployment target, so the path must keep Production.
func TestAffectedComponents_ProductionOverlayKeepsProduction(t *testing.T) {
	g := NewWithT(t)

	const stagingAppSet = `apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: staged
spec:
  generators:
    - clusters:
        values:
          environment: staging
  template:
    metadata:
      name: staged
    spec:
      source:
        path: components/foo
        repoURL: https://github.com/redhat-appstudio/infra-deployments.git
`
	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {}},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(stagingAppSet)},
		exist: map[string]bool{
			"overlays/production-downstream": true,
			"components/foo":                 true,
		},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/deploy.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents([]string{"components/foo/deploy.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Production]).To(HaveLen(1))
	g.Expect(affected[Production][0].Path).To(Equal("components/foo"))
}

// Dependencies are resolved on both refs: a file that a component depended on
// only on the base ref is still part of the change.
func TestDetect_DepsResolvedOnBothRefs(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {
				"components/foo/kustomization.yaml":    true,
				"components/foo/removed-resource.yaml": true,
			},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/foo/removed-resource.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

// An empty dependency tree on one ref (a kustomization with no local
// resources) must merge cleanly with the other ref's tree.
func TestDetect_EmptyDepTreeOnOneRef(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps:  map[string]map[string]bool{"components/foo": {}},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/base-only.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/foo/base-only.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

// A component that exists on both refs and gains a dependency in HEAD pulls in
// the new dependency even though the base tree does not know it.
func TestDetect_DepsResolvedOnHead(t *testing.T) {
	g := NewWithT(t)

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {
				"components/foo/kustomization.yaml": true,
				"components/foo/new-resource.yaml":  true,
			},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(minimalAppSetYAML)},
		exist: map[string]bool{"overlays/development": true, "components/foo": true},
		deps: map[string]map[string]bool{
			"components/foo": {"components/foo/kustomization.yaml": true},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	result, err := d.Detect([]string{"components/foo/new-resource.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.AffectedEnvironments).To(HaveKey(Development))
}

// ---------------------------------------------------------------------------
// FileEnvs (per-file environment attribution)
// ---------------------------------------------------------------------------

// A file specific to one environment is attributed to that environment only.
func TestDetect_FileEnvsSingleEnvironment(t *testing.T) {
	g := NewWithT(t)

	yaml := appSetWithCluster("components/svc", "staging", "kflux-stg-p02")
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		yamls: map[string][]byte{"overlays/staging-downstream": []byte(yaml)},
		exist: map[string]bool{
			"overlays/staging-downstream":          true,
			"components/svc/staging":               true,
			"components/svc/staging/kflux-stg-p02": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/staging": {
				"components/svc/staging/kustomization.yaml": true,
			},
			"components/svc/staging/kflux-stg-p02": {
				"components/svc/staging/kflux-stg-p02/kustomization.yaml": true,
			},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"staging-downstream"}},
		yamls: map[string][]byte{"overlays/staging-downstream": []byte(yaml)},
		exist: map[string]bool{"overlays/staging-downstream": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	const file = "components/svc/staging/kflux-stg-p02/kustomization.yaml"
	result, err := d.Detect([]string{file})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.FileEnvs).To(HaveKey(file))
	g.Expect(result.FileEnvs[file]).To(Equal(map[Environment]bool{Staging: true}))
}

// A file that both a staging and a production component depend on is attributed
// to both, which is what the ring check uses to spot shared base files.
func TestDetect_FileEnvsSharedBaseFile(t *testing.T) {
	g := NewWithT(t)

	stagingYAML := appSetWithCluster("components/svc", "staging", "kflux-stg-p02")
	productionYAML := appSetWithCluster("components/svc", "production", "stone-prod-p01")

	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {"staging-downstream", "production-downstream"}},
		yamls: map[string][]byte{
			"overlays/staging-downstream":    []byte(stagingYAML),
			"overlays/production-downstream": []byte(productionYAML),
		},
		exist: map[string]bool{
			"overlays/staging-downstream":              true,
			"overlays/production-downstream":           true,
			"components/svc/staging/kflux-stg-p02":     true,
			"components/svc/production/stone-prod-p01": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/staging/kflux-stg-p02": {
				"components/svc/base/shared.yaml": true,
			},
			"components/svc/production/stone-prod-p01": {
				"components/svc/base/shared.yaml": true,
			},
		},
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"staging-downstream", "production-downstream"}},
		yamls: map[string][]byte{
			"overlays/staging-downstream":    []byte(stagingYAML),
			"overlays/production-downstream": []byte(productionYAML),
		},
		exist: map[string]bool{
			"overlays/staging-downstream":    true,
			"overlays/production-downstream": true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	const shared = "components/svc/base/shared.yaml"
	result, err := d.Detect([]string{shared})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.FileEnvs[shared]).To(Equal(map[Environment]bool{Staging: true, Production: true}))
	g.Expect(result.AffectedEnvironments).To(HaveKey(Staging))
	g.Expect(result.AffectedEnvironments).To(HaveKey(Production))
}

// Ring-style paths carry no environment, so FileEnvs drives ring enforcement.
// A MatchAll entry must not add a second environment to an otherwise
// path-specific ring file and hide the direct conflict.
func TestDetect_RingPathsDirectConflictWithMatchAll(t *testing.T) {
	g := NewWithT(t)

	stagingYAML := ringAppSet("components/svc", "ring-1", "kflux-stg-p02", "staging")
	productionYAML := ringAppSet("components/svc", "ring-2", "stone-prod-p01", "production") +
		"\n---\n" + matchAllAppSet

	const stagingFile = "components/svc/rings/ring-1/kflux-stg-p02/kustomization.yaml"
	const productionFile = "components/svc/rings/ring-2/stone-prod-p01/kustomization.yaml"

	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {"staging-downstream", "production-downstream"}},
		yamls: map[string][]byte{
			"overlays/staging-downstream":    []byte(stagingYAML),
			"overlays/production-downstream": []byte(productionYAML),
		},
		exist: map[string]bool{
			"overlays/staging-downstream":                true,
			"overlays/production-downstream":             true,
			"components/svc/rings/ring-1/kflux-stg-p02":  true,
			"components/svc/rings/ring-2/stone-prod-p01": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/rings/ring-1/kflux-stg-p02":  {stagingFile: true},
			"components/svc/rings/ring-2/stone-prod-p01": {productionFile: true},
		},
	}
	base := &fakeRepo{
		dirs:  head.dirs,
		yamls: head.yamls,
		exist: map[string]bool{
			"overlays/staging-downstream":    true,
			"overlays/production-downstream": true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	changed := []string{stagingFile, productionFile}
	result, err := d.Detect(changed)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.FileEnvs[stagingFile]).To(Equal(map[Environment]bool{Staging: true}))
	g.Expect(result.FileEnvs[productionFile]).To(Equal(map[Environment]bool{Production: true}))

	ring := CheckRingDeployment(changed, result.AffectedEnvironments, result.FileEnvs)
	g.Expect(ring.DirectConflict).To(BeTrue())
	g.Expect(ring.StagingFiles).To(ConsistOf(stagingFile))
	g.Expect(ring.ProductionFiles).To(ConsistOf(productionFile))
}

// ---------------------------------------------------------------------------
// AffectedComponents: ApplicationSet-only changes
// ---------------------------------------------------------------------------

// Adding a cluster to an ApplicationSet list changes what is rendered without
// changing any file the list points at. render-diff must still render it.
func TestAffectedComponents_NewClusterInAppSet(t *testing.T) {
	g := NewWithT(t)

	baseYAML := appSetWithCluster("components/svc", "production", "stone-prod-p01")
	headYAML := appSetWithClusterWithClusters("components/svc", "production",
		[]string{"stone-prod-p01", "stone-prod-p02"})

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(headYAML)},
		exist: map[string]bool{
			"overlays/production-downstream":           true,
			"components/svc/production":                true,
			"components/svc/production/stone-prod-p01": true,
			"components/svc/production/stone-prod-p02": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/production": {
				"components/svc/production/kustomization.yaml": true,
			},
			"components/svc/production/stone-prod-p01": {
				"components/svc/production/stone-prod-p01/kustomization.yaml": true,
			},
			"components/svc/production/stone-prod-p02": {
				"components/svc/production/stone-prod-p02/kustomization.yaml": true,
			},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(baseYAML)},
		exist: map[string]bool{
			"overlays/production-downstream":           true,
			"components/svc/production":                true,
			"components/svc/production/stone-prod-p01": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/production": {
				"components/svc/production/kustomization.yaml": true,
			},
			"components/svc/production/stone-prod-p01": {
				"components/svc/production/stone-prod-p01/kustomization.yaml": true,
			},
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	// No changed files at all: the change is purely in the ApplicationSet.
	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())

	paths := make([]string, 0, len(affected[Production]))
	for _, cp := range affected[Production] {
		paths = append(paths, cp.Path)
	}
	g.Expect(paths).To(ContainElement("components/svc/production/stone-prod-p02"))
}

// An unchanged overlay adds nothing, even when unchanged files are present.
func TestAffectedComponents_UnchangedOverlayAddsNothing(t *testing.T) {
	g := NewWithT(t)

	yaml := appSetWithCluster("components/svc", "production", "stone-prod-p01")
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(yaml)},
		exist: map[string]bool{
			"overlays/production-downstream":           true,
			"components/svc/production":                true,
			"components/svc/production/stone-prod-p01": true,
		},
		deps: map[string]map[string]bool{
			"components/svc/production": {
				"components/svc/production/kustomization.yaml": true,
			},
			"components/svc/production/stone-prod-p01": {
				"components/svc/production/stone-prod-p01/kustomization.yaml": true,
			},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(yaml)},
		exist: map[string]bool{"overlays/production-downstream": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected).To(BeEmpty())
}

// A new ApplicationSet reference must render even when the component directory
// already existed, unchanged, on the base ref.
func TestAffectedComponents_NewReferenceToExistingDirectory(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/production/stone-prod-p02"
	baseYAML := appSetWithCluster("components/svc", "production", "stone-prod-p01")
	headYAML := appSetWithCluster("components/svc", "production", "stone-prod-p02")

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(headYAML)},
		exist: map[string]bool{
			"overlays/production-downstream": true,
			"components/svc/production":      true,
			path:                             true,
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(baseYAML)},
		exist: map[string]bool{
			"overlays/production-downstream":           true,
			"components/svc/production":                true,
			"components/svc/production/stone-prod-p01": true,
			path: true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())

	var found *appset.ComponentPath
	for i := range affected[Production] {
		if affected[Production][i].Path == path {
			found = &affected[Production][i]
			break
		}
	}
	if found == nil {
		t.Fatalf("path %s not found in %v", path, affected[Production])
	}
	g.Expect(found.NewDeployment).To(BeTrue())
}

// Moving a deployment between declared environments without changing its path
// is still a new deployment for the new environment.
func TestAffectedComponents_EnvironmentMoveIsNewDeployment(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/shared"
	headYAML := appSetStaticPathWithEnv(path, "production")
	baseYAML := appSetStaticPathWithEnv(path, "staging")

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(headYAML)},
		exist: map[string]bool{"overlays/development": true, path: true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(baseYAML)},
		exist: map[string]bool{"overlays/development": true, path: true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Production]).To(HaveLen(1))
	g.Expect(affected[Production][0].Path).To(Equal(path))
	g.Expect(affected[Production][0].NewDeployment).To(BeTrue())
}

// An added path is attributed to the overlay environment and to a different
// declared environment, matching extractPathsFromOverlays.
func TestAffectedComponents_AddedPathUsesBothEnvironments(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/new"
	headYAML := appSetStaticPathWithEnv(path, "staging")
	baseYAML := `apiVersion: v1
kind: Namespace
metadata:
  name: placeholder
`

	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(headYAML)},
		exist: map[string]bool{"overlays/development": true, path: true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(baseYAML)},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	for _, env := range []Environment{Development, Staging} {
		g.Expect(affected[env]).To(HaveLen(1), "environment %s", env)
		g.Expect(affected[env][0].Path).To(Equal(path))
		g.Expect(affected[env][0].NewDeployment).To(BeTrue())
	}
}

// Different value files for the same directory and cluster are one render job.
func TestAppendNewComponentPaths_DedupesRenderTarget(t *testing.T) {
	g := NewWithT(t)

	paths := appendNewComponentPaths(nil, []appset.ComponentPath{
		{Path: "components/svc/new", ExtraFiles: []string{"a.yaml"}, NewDeployment: true},
		{Path: "components/svc/new", ExtraFiles: []string{"b.yaml"}, NewDeployment: true},
	})
	g.Expect(paths).To(HaveLen(1))
}

// A base-ref parse failure must not turn every HEAD path into a new deployment.
func TestAffectedComponents_BaseParseFailureDoesNotMarkNew(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/dev"
	headYAML := appSetStaticPathWithEnv(path, "development")
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(headYAML)},
		exist: map[string]bool{"overlays/development": true, path: true},
		deps: map[string]map[string]bool{
			path: {path + "/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("not: valid: yaml: [}")},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents([]string{path + "/kustomization.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Development]).To(HaveLen(1))
	g.Expect(affected[Development][0].NewDeployment).To(BeFalse())
}

// A target already deployed by one overlay in an environment is not new just
// because another overlay in the same environment starts rendering it.
func TestAffectedComponents_SharedTargetAcrossOverlaysIsNotNew(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/production/shared"
	appSet := appSetStaticPathWithEnv(path, "production")
	empty := `apiVersion: v1
kind: Namespace
metadata:
  name: placeholder
`
	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {"production-downstream", "konflux-public-production"}},
		yamls: map[string][]byte{
			"overlays/production-downstream":     []byte(appSet),
			"overlays/konflux-public-production": []byte(appSet),
		},
		exist: map[string]bool{
			"overlays/production-downstream":     true,
			"overlays/konflux-public-production": true,
			path:                                 true,
		},
	}
	base := &fakeRepo{
		dirs: head.dirs,
		yamls: map[string][]byte{
			"overlays/production-downstream":     []byte(appSet),
			"overlays/konflux-public-production": []byte(empty),
		},
		exist: map[string]bool{
			"overlays/production-downstream":     true,
			"overlays/konflux-public-production": true,
			path:                                 true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected).To(BeEmpty())
}

// A new reference to a directory that exists only on the base ref stays a
// removal: it must not be marked NewDeployment, which would suppress both
// builds.
func TestAffectedComponents_NewReferenceToBaseOnlyDirectory(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/old"
	headYAML := appSetStaticPathWithEnv(path, "development")
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(headYAML)},
		exist: map[string]bool{"overlays/development": true},
	}
	base := &fakeRepo{
		dirs: map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(`apiVersion: v1
kind: Namespace
metadata:
  name: placeholder
`)},
		exist: map[string]bool{"overlays/development": true, path: true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Development]).To(HaveLen(1))
	g.Expect(affected[Development][0].Path).To(Equal(path))
	g.Expect(affected[Development][0].NewDeployment).To(BeFalse())
}

// A path that changed files already matched and the ApplicationSet also starts
// deploying must keep a single job and be marked as a new deployment.
func TestAffectedComponents_ChangedFileInNewDeployment(t *testing.T) {
	g := NewWithT(t)

	const basePath = "components/svc/production/stone-prod-p01"
	const newPath = "components/svc/production/stone-prod-p02"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(appSetStaticPathWithEnv(newPath, "production"))},
		exist: map[string]bool{
			"overlays/production-downstream": true,
			newPath:                          true,
		},
		deps: map[string]map[string]bool{
			newPath: {newPath + "/kustomization.yaml": true},
		},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"production-downstream"}},
		yamls: map[string][]byte{"overlays/production-downstream": []byte(appSetStaticPathWithEnv(basePath, "production"))},
		exist: map[string]bool{
			"overlays/production-downstream": true,
			basePath:                         true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents([]string{newPath + "/kustomization.yaml"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Production]).To(HaveLen(1))
	g.Expect(affected[Production][0].Path).To(Equal(newPath))
	g.Expect(affected[Production][0].NewDeployment).To(BeTrue())
}

func TestAppendNewComponentPaths_PromotesMatchedPath(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/new"
	paths := appendNewComponentPaths(
		[]appset.ComponentPath{{Path: path}},
		[]appset.ComponentPath{{Path: path, NewDeployment: true}},
	)
	g.Expect(paths).To(HaveLen(1))
	g.Expect(paths[0].NewDeployment).To(BeTrue())
}

// A failed base parse must not stop HEAD paths from being rendered.
func TestAffectedComponents_BaseParseFailureStillAddsHeadPaths(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/dev"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(appSetStaticPathWithEnv(path, "development"))},
		exist: map[string]bool{"overlays/development": true, path: true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte("not: valid: yaml: [}")},
		exist: map[string]bool{"overlays/development": true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Development]).To(HaveLen(1))
	g.Expect(affected[Development][0].Path).To(Equal(path))
	g.Expect(affected[Development][0].NewDeployment).To(BeFalse())
}

// An unparseable base overlay makes every path in its environment untrusted:
// another overlay in the same environment must not mark paths as new.
func TestAffectedComponents_BaseParseFailureBlocksNewDeploymentInEnv(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/production/shared"
	head := &fakeRepo{
		dirs: map[string][]string{"overlays": {"production-downstream", "konflux-public-production"}},
		yamls: map[string][]byte{
			"overlays/production-downstream":     []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: placeholder\n"),
			"overlays/konflux-public-production": []byte(appSetStaticPathWithEnv(path, "production")),
		},
		exist: map[string]bool{
			"overlays/production-downstream":     true,
			"overlays/konflux-public-production": true,
			path:                                 true,
		},
	}
	base := &fakeRepo{
		dirs: head.dirs,
		yamls: map[string][]byte{
			"overlays/production-downstream":     []byte("not: valid: yaml: [}"),
			"overlays/konflux-public-production": []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: placeholder\n"),
		},
		exist: map[string]bool{
			"overlays/production-downstream":     true,
			"overlays/konflux-public-production": true,
		},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Production]).To(HaveLen(1))
	g.Expect(affected[Production][0].Path).To(Equal(path))
	g.Expect(affected[Production][0].NewDeployment).To(BeFalse())
}

// A resolved HEAD path covered by a base PrefixOnly entry was already deployed;
// it must render against base, not as a full addition.
func TestAffectedComponents_BasePrefixOnlyCoverDoesNotMarkNew(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/dev/base"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(appSetStaticPathWithEnv(path, "development"))},
		exist: map[string]bool{"overlays/development": true, path: true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(appSetStaticPathWithEnv("components/svc/{{name}}/base", "development"))},
		exist: map[string]bool{"overlays/development": true, path: true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Development]).To(HaveLen(1))
	g.Expect(affected[Development][0].Path).To(Equal(path))
	g.Expect(affected[Development][0].NewDeployment).To(BeFalse())
}

// A base MatchAll entry covers every resolved HEAD path in its environment.
func TestAffectedComponents_BaseMatchAllCoverDoesNotMarkNew(t *testing.T) {
	g := NewWithT(t)

	const path = "components/svc/dev/base"
	head := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(appSetStaticPathWithEnv(path, "development"))},
		exist: map[string]bool{"overlays/development": true, path: true},
	}
	base := &fakeRepo{
		dirs:  map[string][]string{"overlays": {"development"}},
		yamls: map[string][]byte{"overlays/development": []byte(matchAllAppSet)},
		exist: map[string]bool{"overlays/development": true, path: true},
	}

	d, err := NewDetector(head, base, "overlays")
	g.Expect(err).NotTo(HaveOccurred())

	affected, err := d.AffectedComponents(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(affected[Development]).To(HaveLen(1))
	g.Expect(affected[Development][0].Path).To(Equal(path))
	g.Expect(affected[Development][0].NewDeployment).To(BeFalse())
}
