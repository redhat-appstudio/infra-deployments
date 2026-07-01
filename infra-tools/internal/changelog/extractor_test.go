package changelog_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
)

const realRef = "0bace1e20ff164a00e9d6becfce52e310a921931"

// validKustomization is a minimal kustomization.yaml matching the real file.
const validKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/konflux-ci/konflux-ci/operator/config/default?ref=0bace1e20ff164a00e9d6becfce52e310a921931
  - konflux.yaml
patches:
  - path: release-config.yaml
images:
  - name: localhost/konflux-operator
    newName: quay.io/konflux-ci/konflux-operator
    newTag: 0bace1e20ff164a00e9d6becfce52e310a921931
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "kustomization-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

func TestExtractRef_ValidFile(t *testing.T) {
	g := NewWithT(t)
	path := writeTemp(t, validKustomization)
	ref, err := changelog.ExtractRef(path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ref).To(Equal(realRef))
}

func TestExtractRef_RealFile(t *testing.T) {
	g := NewWithT(t)
	// Walk up from this test file to the repo root and find the real overlay.
	path := filepath.Join("..", "..", "..", "components",
		"konflux-operator", "rings", "ring-0", "base", "kustomization.yaml")
	ref, err := changelog.ExtractRef(path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ref).NotTo(BeEmpty())
}

func TestExtractRef_FileNotFound(t *testing.T) {
	g := NewWithT(t)
	_, err := changelog.ExtractRef("/nonexistent/path/kustomization.yaml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading"))
}

func TestExtractRef_InvalidYAML(t *testing.T) {
	g := NewWithT(t)
	path := writeTemp(t, "this: is: not: valid: yaml: ][")
	_, err := changelog.ExtractRef(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("parsing"))
}

func TestExtractRef_MissingResourceEntry(t *testing.T) {
	g := NewWithT(t)
	content := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - some-other-resource.yaml
`
	path := writeTemp(t, content)
	_, err := changelog.ExtractRef(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no operator resource URL"))
}

func TestExtractRef_ResourceURLWithoutRef(t *testing.T) {
	g := NewWithT(t)
	content := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/konflux-ci/konflux-ci/operator/config/default
`
	path := writeTemp(t, content)
	_, err := changelog.ExtractRef(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no operator resource URL"))
}

func TestExtractRefs_Unchanged(t *testing.T) {
	g := NewWithT(t)
	path := writeTemp(t, validKustomization)
	old, new, err := changelog.ExtractRefs(path, path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(old).To(Equal(realRef))
	g.Expect(new).To(Equal(realRef))
}

func TestExtractRefs_Changed(t *testing.T) {
	g := NewWithT(t)

	updatedRef := "9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f09"
	newContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/konflux-ci/konflux-ci/operator/config/default?ref=9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f09
`
	basePath := writeTemp(t, validKustomization)
	headPath := writeTemp(t, newContent)

	old, new, err := changelog.ExtractRefs(basePath, headPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(old).To(Equal(realRef))
	g.Expect(new).To(Equal(updatedRef))
}

func TestExtractRefs_BaseError(t *testing.T) {
	g := NewWithT(t)
	headPath := writeTemp(t, validKustomization)
	_, _, err := changelog.ExtractRefs("/nonexistent/kustomization.yaml", headPath)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("base ref"))
}

func TestExtractRefs_HeadError(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTemp(t, validKustomization)
	_, _, err := changelog.ExtractRefs(basePath, "/nonexistent/kustomization.yaml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("head ref"))
}
