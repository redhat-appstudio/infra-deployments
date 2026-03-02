package renderdiff

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
)

// fakeBuilder implements RepoBuilder for testing.
type fakeBuilder struct {
	exist map[string]bool
	yamls map[string][]byte
}

func (f *fakeBuilder) DirExists(rel string) bool {
	return f.exist[rel]
}

func (f *fakeBuilder) BuildKustomization(rel string) ([]byte, error) {
	if y, ok := f.yamls[rel]; ok {
		return y, nil
	}
	return nil, fmt.Errorf("no kustomization at %s", rel)
}

func TestEngine_NormalDiff(t *testing.T) {
	g := NewWithT(t)

	head := &fakeBuilder{
		exist: map[string]bool{"components/foo/staging": true},
		yamls: map[string][]byte{"components/foo/staging": []byte("apiVersion: v1\nkind: ConfigMap\ndata:\n  key: new-value\n")},
	}
	base := &fakeBuilder{
		exist: map[string]bool{"components/foo/staging": true},
		yamls: map[string][]byte{"components/foo/staging": []byte("apiVersion: v1\nkind: ConfigMap\ndata:\n  key: old-value\n")},
	}

	engine := NewEngine(head, base, 2)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Staging: {{Path: "components/foo/staging"}},
	}

	result, err := engine.Run(context.Background(), affected)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(HaveLen(1))
	g.Expect(result.Diffs[0].Path).To(Equal("components/foo/staging"))
	g.Expect(result.Diffs[0].Env).To(Equal(detector.Staging))
	g.Expect(result.Diffs[0].Added).To(Equal(1))
	g.Expect(result.Diffs[0].Removed).To(Equal(1))
	g.Expect(result.Diffs[0].Diff).To(ContainSubstring("-  key: old-value"))
	g.Expect(result.Diffs[0].Diff).To(ContainSubstring("+  key: new-value"))
	g.Expect(result.TotalAdded).To(Equal(1))
	g.Expect(result.TotalRemoved).To(Equal(1))
}

func TestEngine_NewComponent(t *testing.T) {
	g := NewWithT(t)

	head := &fakeBuilder{
		exist: map[string]bool{"components/new/dev": true},
		yamls: map[string][]byte{"components/new/dev": []byte("apiVersion: v1\nkind: Service\n")},
	}
	base := &fakeBuilder{
		exist: map[string]bool{}, // does not exist on base
	}

	engine := NewEngine(head, base, 2)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Development: {{Path: "components/new/dev"}},
	}

	result, err := engine.Run(context.Background(), affected)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(HaveLen(1))
	g.Expect(result.Diffs[0].Added).To(BeNumerically(">", 0))
	g.Expect(result.Diffs[0].Removed).To(Equal(0))
	g.Expect(result.Diffs[0].BaseYAML).To(BeNil())
}

func TestEngine_RemovedComponent(t *testing.T) {
	g := NewWithT(t)

	head := &fakeBuilder{
		exist: map[string]bool{}, // does not exist on head
	}
	base := &fakeBuilder{
		exist: map[string]bool{"components/old/prod": true},
		yamls: map[string][]byte{"components/old/prod": []byte("apiVersion: v1\nkind: Deployment\n")},
	}

	engine := NewEngine(head, base, 2)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Production: {{Path: "components/old/prod"}},
	}

	result, err := engine.Run(context.Background(), affected)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(HaveLen(1))
	g.Expect(result.Diffs[0].Added).To(Equal(0))
	g.Expect(result.Diffs[0].Removed).To(BeNumerically(">", 0))
	g.Expect(result.Diffs[0].HeadYAML).To(BeNil())
}

func TestEngine_BuildFailure_ReportsError(t *testing.T) {
	g := NewWithT(t)

	head := &fakeBuilder{
		exist: map[string]bool{"components/broken/staging": true},
		yamls: map[string][]byte{}, // BuildKustomization will fail
	}
	base := &fakeBuilder{
		exist: map[string]bool{"components/broken/staging": true},
		yamls: map[string][]byte{"components/broken/staging": []byte("valid: yaml\n")},
	}

	engine := NewEngine(head, base, 2)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Staging: {{Path: "components/broken/staging"}},
	}

	result, err := engine.Run(context.Background(), affected)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(HaveLen(1))
	g.Expect(result.Diffs[0].Error).NotTo(BeEmpty())
	g.Expect(result.Diffs[0].Path).To(Equal("components/broken/staging"))
}

func TestEngine_NoEffectiveChange(t *testing.T) {
	g := NewWithT(t)

	sameYAML := []byte("apiVersion: v1\nkind: ConfigMap\ndata:\n  key: same\n")
	head := &fakeBuilder{
		exist: map[string]bool{"components/foo/staging": true},
		yamls: map[string][]byte{"components/foo/staging": sameYAML},
	}
	base := &fakeBuilder{
		exist: map[string]bool{"components/foo/staging": true},
		yamls: map[string][]byte{"components/foo/staging": sameYAML},
	}

	engine := NewEngine(head, base, 2)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Staging: {{Path: "components/foo/staging"}},
	}

	result, err := engine.Run(context.Background(), affected)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(BeEmpty()) // identical YAML → omitted
}

func TestEngine_EmptyInput(t *testing.T) {
	g := NewWithT(t)

	engine := NewEngine(&fakeBuilder{}, &fakeBuilder{}, 2)
	result, err := engine.Run(context.Background(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(BeEmpty())
}

func TestEngine_Progressive(t *testing.T) {
	g := NewWithT(t)

	head := &fakeBuilder{
		exist: map[string]bool{"components/a/dev": true, "components/b/dev": true},
		yamls: map[string][]byte{
			"components/a/dev": []byte("new-a\n"),
			"components/b/dev": []byte("new-b\n"),
		},
	}
	base := &fakeBuilder{
		exist: map[string]bool{"components/a/dev": true, "components/b/dev": true},
		yamls: map[string][]byte{
			"components/a/dev": []byte("old-a\n"),
			"components/b/dev": []byte("old-b\n"),
		},
	}

	engine := NewEngine(head, base, 2)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Development: {
			{Path: "components/a/dev"},
			{Path: "components/b/dev"},
		},
	}

	ch := make(chan ComponentDiff, 10)
	result, err := engine.RunProgressive(context.Background(), affected, ch)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Diffs).To(HaveLen(2))

	// Channel is already closed and drained by RunProgressive.
	// Verify via the result.
	g.Expect(result.TotalAdded).To(Equal(2))
	g.Expect(result.TotalRemoved).To(Equal(2))
}

func TestCountStats(t *testing.T) {
	g := NewWithT(t)

	diff := `--- a/file
+++ b/file
@@ -1,3 +1,3 @@
 unchanged
-removed line
+added line
 unchanged
+another added
`
	added, removed := countStats(diff)
	g.Expect(added).To(Equal(2))
	g.Expect(removed).To(Equal(1))
}
