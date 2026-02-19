package detector

import (
	"fmt"
	"slices"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
)

// ---------------------------------------------------------------------------
// fakeRepo implements RepoQuerier for deterministic, filesystem-free testing.
// ---------------------------------------------------------------------------

type fakeRepo struct {
	dirs  map[string][]string        // ListSubDirs results keyed by rel path
	exist map[string]bool            // DirExists results keyed by rel path
	yamls map[string][]byte          // BuildKustomization results keyed by rel path
	deps  map[string]map[string]bool // ResolveDeps results keyed by rel path
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

func (f *fakeRepo) BuildKustomization(rel string) ([]byte, error) {
	if y, ok := f.yamls[rel]; ok {
		return y, nil
	}
	return nil, fmt.Errorf("no kustomization at %s", rel)
}

func (f *fakeRepo) ResolveDeps(rel string) (map[string]bool, error) {
	if d, ok := f.deps[rel]; ok {
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
        repoURL: https://example.com/repo.git
      destination:
        server: '{{server}}'
`

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
        repoURL: https://github.com/example/repo.git
      destination:
        server: '{{server}}'
`, root, env, cluster, cluster)
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

// ---------------------------------------------------------------------------
// extractPathsFromOverlays
// ---------------------------------------------------------------------------

func TestExtractPathsFromOverlays_Basic(t *testing.T) {
	g := NewWithT(t)

	yaml := appSetWithCluster("components/test", "staging", "stone-prod-p01")
	builds := []overlayBuild{
		{name: "staging-downstream", env: Staging, headYAML: []byte(yaml)},
	}

	envPaths, allClusters, err := extractPathsFromOverlays(builds)
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

func TestExtractPathsFromOverlays_MultipleEnvs(t *testing.T) {
	g := NewWithT(t)

	builds := []overlayBuild{
		{name: "staging-downstream", env: Staging, headYAML: []byte(minimalAppSetYAML)},
		{name: "production-downstream", env: Production, headYAML: []byte(minimalAppSetYAML)},
	}

	envPaths, _, err := extractPathsFromOverlays(builds)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(envPaths).To(HaveKey(Staging))
	g.Expect(envPaths).To(HaveKey(Production))
}

func TestExtractPathsFromOverlays_InvalidYAML(t *testing.T) {
	g := NewWithT(t)

	builds := []overlayBuild{
		{name: "bad", env: Development, headYAML: []byte("not: valid: yaml: [}")},
	}

	_, _, err := extractPathsFromOverlays(builds)
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
