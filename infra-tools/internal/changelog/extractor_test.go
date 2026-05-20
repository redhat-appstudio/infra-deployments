package changelog

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

const realSHA = "0bace1e20ff164a00e9d6becfce52e310a921931"

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

func TestExtractSHA_ValidFile(t *testing.T) {
	g := NewWithT(t)
	path := writeTemp(t, validKustomization)
	sha, err := ExtractSHA(path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sha).To(Equal(realSHA))
}

func TestExtractSHA_RealFile(t *testing.T) {
	g := NewWithT(t)
	// Walk up from this test file to the repo root and find the real overlay.
	path := filepath.Join("..", "..", "..", "components",
		"konflux-operator", "development", "invariant", "kustomization.yaml")
	sha, err := ExtractSHA(path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sha).To(MatchRegexp(`^[0-9a-f]{40}$`))
}

func TestExtractSHA_FileNotFound(t *testing.T) {
	g := NewWithT(t)
	_, err := ExtractSHA("/nonexistent/path/kustomization.yaml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reading"))
}

func TestExtractSHA_InvalidYAML(t *testing.T) {
	g := NewWithT(t)
	path := writeTemp(t, "this: is: not: valid: yaml: ][")
	_, err := ExtractSHA(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("parsing"))
}

func TestExtractSHA_MissingImageEntry(t *testing.T) {
	g := NewWithT(t)
	content := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: some-other-image
    newTag: 0bace1e20ff164a00e9d6becfce52e310a921931
`
	path := writeTemp(t, content)
	_, err := ExtractSHA(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no image entry found"))
}

func TestExtractSHA_InvalidSHAFormat(t *testing.T) {
	g := NewWithT(t)
	content := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: localhost/konflux-operator
    newName: quay.io/konflux-ci/konflux-operator
    newTag: not-a-sha
`
	path := writeTemp(t, content)
	_, err := ExtractSHA(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not a 40-character hex SHA"))
}

func TestExtractSHAs_Unchanged(t *testing.T) {
	g := NewWithT(t)
	path := writeTemp(t, validKustomization)
	old, new, err := ExtractSHAs(path, path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(old).To(Equal(realSHA))
	g.Expect(new).To(Equal(realSHA))
}

func TestExtractSHAs_Changed(t *testing.T) {
	g := NewWithT(t)

	updatedSHA := "9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f09"
	newContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: localhost/konflux-operator
    newName: quay.io/konflux-ci/konflux-operator
    newTag: 9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f09
`
	basePath := writeTemp(t, validKustomization)
	headPath := writeTemp(t, newContent)

	old, new, err := ExtractSHAs(basePath, headPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(old).To(Equal(realSHA))
	g.Expect(new).To(Equal(updatedSHA))
}

func TestExtractSHAs_BaseError(t *testing.T) {
	g := NewWithT(t)
	headPath := writeTemp(t, validKustomization)
	_, _, err := ExtractSHAs("/nonexistent/kustomization.yaml", headPath)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("base SHA"))
}

func TestExtractSHAs_HeadError(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTemp(t, validKustomization)
	_, _, err := ExtractSHAs(basePath, "/nonexistent/kustomization.yaml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("head SHA"))
}
