package changelog_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
)

// --- ExtractImageDigestChanges ---

// TestExtractImageDigestChanges_SingleImage verifies that a digest change for a
// single image (namespace-lister style: digest before name) is detected.
func TestExtractImageDigestChanges_SingleImage(t *testing.T) {
	g := NewWithT(t)

	oldDigest := "sha256:" + str64('a')
	newDigest := "sha256:" + str64('b')

	patch := " images:\n" +
		"-- digest: " + oldDigest + "\n" +
		"+- digest: " + newDigest + "\n" +
		"   name: quay.io/konflux-ci/namespace-lister\n" +
		"   newName: quay.io/konflux-ci/namespace-lister\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/namespace-lister/kustomization.yaml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(HaveLen(1))
	g.Expect(changes[0].ImageName).To(Equal("quay.io/konflux-ci/namespace-lister"))
	g.Expect(changes[0].OldDigest).To(Equal(oldDigest))
	g.Expect(changes[0].NewDigest).To(Equal(newDigest))
}

// TestExtractImageDigestChanges_MultipleImagesAllChange verifies that three
// digest changes within a single kustomization file (UI proxy style) are all
// detected independently.
func TestExtractImageDigestChanges_MultipleImagesAllChange(t *testing.T) {
	g := NewWithT(t)

	oldA := "sha256:" + str64('a')
	newA := "sha256:" + str64('b')
	oldB := "sha256:" + str64('c')
	newB := "sha256:" + str64('d')
	oldC := "sha256:" + str64('e')
	newC := "sha256:" + str64('f')

	patch := " images:\n" +
		"-- digest: " + oldA + "\n" +
		"+- digest: " + newA + "\n" +
		"   name: quay.io/konflux-ci/konflux-ui\n" +
		"   newName: quay.io/konflux-ci/konflux-ui\n" +
		"-- digest: " + oldB + "\n" +
		"+- digest: " + newB + "\n" +
		"   name: quay.io/konflux-ci/oauth2-proxy\n" +
		"   newName: quay.io/konflux-ci/oauth2-proxy\n" +
		"-- digest: " + oldC + "\n" +
		"+- digest: " + newC + "\n" +
		"   name: quay.io/konflux-ci/reverse-proxy\n" +
		"   newName: quay.io/konflux-ci/reverse-proxy\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/ui/core/proxy/kustomization.yaml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(HaveLen(3))
	g.Expect(changes[0].ImageName).To(Equal("quay.io/konflux-ci/konflux-ui"))
	g.Expect(changes[1].ImageName).To(Equal("quay.io/konflux-ci/oauth2-proxy"))
	g.Expect(changes[2].ImageName).To(Equal("quay.io/konflux-ci/reverse-proxy"))
}

// TestExtractImageDigestChanges_OneOfThreeChanges verifies that when only one
// of three images in a file changes, only that one is reported.
func TestExtractImageDigestChanges_OneOfThreeChanges(t *testing.T) {
	g := NewWithT(t)

	oldB := "sha256:" + str64('c')
	newB := "sha256:" + str64('d')
	unchangedA := "sha256:" + str64('a')
	unchangedC := "sha256:" + str64('e')

	// Image A unchanged (context lines), image B changed, image C unchanged.
	patch := " images:\n" +
		" - digest: " + unchangedA + "\n" +
		"   name: quay.io/konflux-ci/konflux-ui\n" +
		"   newName: quay.io/konflux-ci/konflux-ui\n" +
		"-- digest: " + oldB + "\n" +
		"+- digest: " + newB + "\n" +
		"   name: quay.io/konflux-ci/oauth2-proxy\n" +
		"   newName: quay.io/konflux-ci/oauth2-proxy\n" +
		" - digest: " + unchangedC + "\n" +
		"   name: quay.io/konflux-ci/reverse-proxy\n" +
		"   newName: quay.io/konflux-ci/reverse-proxy\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/ui/core/proxy/kustomization.yaml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(HaveLen(1))
	g.Expect(changes[0].ImageName).To(Equal("quay.io/konflux-ci/oauth2-proxy"))
	g.Expect(changes[0].OldDigest).To(Equal(oldB))
	g.Expect(changes[0].NewDigest).To(Equal(newB))
}

// TestExtractImageDigestChanges_NameBeforeDigest verifies dex-style kustomizations
// where name/newName appear before digest in the images: block.
func TestExtractImageDigestChanges_NameBeforeDigest(t *testing.T) {
	g := NewWithT(t)

	oldDigest := "sha256:" + str64('a')
	newDigest := "sha256:" + str64('b')

	patch := " images:\n" +
		" - name: quay.io/konflux-ci/dex\n" +
		"   newName: quay.io/konflux-ci/dex\n" +
		"-   digest: " + oldDigest + "\n" +
		"+   digest: " + newDigest + "\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/ui/dex/kustomization.yml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(HaveLen(1))
	g.Expect(changes[0].ImageName).To(Equal("quay.io/konflux-ci/dex"))
	g.Expect(changes[0].OldDigest).To(Equal(oldDigest))
	g.Expect(changes[0].NewDigest).To(Equal(newDigest))
}

// TestExtractImageDigestChanges_PrefersNewName verifies that newName is used
// when name is a short local alias.
func TestExtractImageDigestChanges_PrefersNewName(t *testing.T) {
	g := NewWithT(t)

	oldDigest := "sha256:" + str64('a')
	newDigest := "sha256:" + str64('b')

	patch := " images:\n" +
		" - name: dex\n" +
		"   newName: quay.io/konflux-ci/dex\n" +
		"-   digest: " + oldDigest + "\n" +
		"+   digest: " + newDigest + "\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/ui/dex/kustomization.yml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(HaveLen(1))
	g.Expect(changes[0].ImageName).To(Equal("quay.io/konflux-ci/dex"))
}

// TestExtractImageDigestChanges_HunkBoundary verifies pending digest state does
// not leak across diff hunk headers.
func TestExtractImageDigestChanges_HunkBoundary(t *testing.T) {
	g := NewWithT(t)

	oldDigest := "sha256:" + str64('a')
	newDigest := "sha256:" + str64('b')

	patch := "@@ -1,5 +1,5 @@\n" +
		"-- digest: " + oldDigest + "\n" +
		"+- digest: " + newDigest + "\n" +
		"@@ -10,5 +10,5 @@\n" +
		"   name: quay.io/konflux-ci/unrelated\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/namespace-lister/kustomization.yaml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(BeEmpty())
}

// digests are the same, no change is emitted.
func TestExtractImageDigestChanges_DigestUnchanged(t *testing.T) {
	g := NewWithT(t)

	sameDigest := "sha256:" + str64('a')
	patch := "-- digest: " + sameDigest + "\n" +
		"+- digest: " + sameDigest + "\n" +
		"   name: quay.io/konflux-ci/namespace-lister\n"

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/namespace-lister/kustomization.yaml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(BeEmpty())
}

// TestExtractImageDigestChanges_EmptyPatch verifies that an upstream
// kustomization file with no patch data sets hasSkipped=true.
func TestExtractImageDigestChanges_EmptyPatch(t *testing.T) {
	g := NewWithT(t)

	files := []changelog.FileChange{
		{Filename: "operator/upstream-kustomizations/namespace-lister/kustomization.yaml", Patch: ""},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeTrue())
	g.Expect(changes).To(BeEmpty())
}

// TestExtractImageDigestChanges_IgnoresNonKustomizationFiles verifies that
// files not under operator/upstream-kustomizations/ are ignored.
func TestExtractImageDigestChanges_IgnoresNonKustomizationFiles(t *testing.T) {
	g := NewWithT(t)

	oldDigest := "sha256:" + str64('a')
	newDigest := "sha256:" + str64('b')
	patch := "-- digest: " + oldDigest + "\n" +
		"+- digest: " + newDigest + "\n" +
		"   name: quay.io/other/image\n"

	files := []changelog.FileChange{
		{Filename: "some/other/path/kustomization.yaml", Patch: patch},
	}
	changes, hasSkipped := changelog.ExtractImageDigestChanges(files)
	g.Expect(hasSkipped).To(BeFalse())
	g.Expect(changes).To(BeEmpty())
}

// --- RegistryInspector ---

func TestNewRegistryInspector(t *testing.T) {
	g := NewWithT(t)
	g.Expect(changelog.NewRegistryInspector()).NotTo(BeNil())
}

func TestInspectLabels_InvalidReference(t *testing.T) {
	g := NewWithT(t)
	r := changelog.NewRegistryInspector()
	_, err := r.InspectLabels(context.Background(), "not-a-valid-image-ref")
	g.Expect(err).To(HaveOccurred())
}

// str64 returns a 64-character hex string filled with the given byte,
// used to construct syntactically valid sha256: test digests.
func str64(b byte) string {
	out := make([]byte, 64)
	for i := range out {
		out[i] = b
	}
	return string(out)
}
