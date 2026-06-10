package changelog_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
)

const (
	buildOldSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	buildNewSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// buildServicePatch returns a realistic unified diff snippet for the build-service
// kustomization file, showing a ref bump from oldSHA to newSHA.
// The URL base is identical on both lines — only ?ref= changes, matching real
// kustomization diffs.
func buildServicePatch(oldSHA, newSHA string) string {
	return "-  - https://github.com/konflux-ci/build-service/config/default?ref=" + oldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config/default?ref=" + newSHA + "\n"
}

func TestExtractServiceBumps_SingleBump(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildNewSHA),
		},
	}
	bumps := changelog.ExtractServiceBumps(files)
	g.Expect(bumps).To(HaveLen(1))
	g.Expect(bumps[0].Owner).To(Equal("konflux-ci"))
	g.Expect(bumps[0].Repo).To(Equal("build-service"))
	g.Expect(bumps[0].OldSHA).To(Equal(buildOldSHA))
	g.Expect(bumps[0].NewSHA).To(Equal(buildNewSHA))
}

func TestExtractServiceBumps_SkipsNonUpstreamFiles(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/config/default/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildNewSHA),
		},
		{
			Filename: "operator/upstream-kustomizations/build-service/other.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildNewSHA),
		},
	}
	g.Expect(changelog.ExtractServiceBumps(files)).To(BeEmpty())
}

func TestExtractServiceBumps_SkipsUnchangedRef(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildOldSHA), // same SHA on both sides
		},
	}
	g.Expect(changelog.ExtractServiceBumps(files)).To(BeEmpty())
}

func TestExtractServiceBumps_DeduplicatesSameRepo(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildNewSHA),
		},
		{
			Filename: "operator/upstream-kustomizations/build-service/extra/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildNewSHA),
		},
	}
	g.Expect(changelog.ExtractServiceBumps(files)).To(HaveLen(1))
}

func TestExtractServiceBumps_MultipleBumps(t *testing.T) {
	g := NewWithT(t)
	intOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	intNewSHA := "dddddddddddddddddddddddddddddddddddddddd"
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildNewSHA),
		},
		{
			Filename: "operator/upstream-kustomizations/integration-service/kustomization.yaml",
			Patch: "-  - https://github.com/konflux-ci/integration-service/...?ref=" + intOldSHA + "\n" +
				"+  - https://github.com/konflux-ci/integration-service/...?ref=" + intNewSHA + "\n",
		},
	}
	bumps := changelog.ExtractServiceBumps(files)
	g.Expect(bumps).To(HaveLen(2))
}

func TestExtractServiceBumps_EmptyPatch(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    "",
		},
	}
	g.Expect(changelog.ExtractServiceBumps(files)).To(BeEmpty())
}

// TestExtractServiceBumps_NoGitHubURL verifies that a kustomization file where
// the ?ref= SHA changes but no github.com URL appears in the added lines is
// skipped — extractGitHubRepo returns ("","") and the bump is not recorded.
func TestExtractServiceBumps_NoGitHubURL(t *testing.T) {
	g := NewWithT(t)
	// The resource URL is not on github.com, so we cannot build a ServiceBump.
	patch := "-  - https://quay.io/some-org/some-image?ref=" + buildOldSHA + "\n" +
		"+  - https://quay.io/some-org/some-image?ref=" + buildNewSHA + "\n"
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/some-service/kustomization.yaml",
			Patch:    patch,
		},
	}
	g.Expect(changelog.ExtractServiceBumps(files)).To(BeEmpty())
}

// TestExtractServiceBumps_MultipleURLsSameFile verifies that when a kustomization
// file has two resource URLs and only one changes, the correct SHA pair is returned.
// This guards against the bug where the first - SHA and first + SHA are paired
// regardless of whether they refer to the same URL.
func TestExtractServiceBumps_MultipleURLsSameFile(t *testing.T) {
	g := NewWithT(t)
	// service-a's SHA did NOT change (context line, no +/- prefix).
	// service-b's SHA DID change.
	serviceARef := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	serviceBOld := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	serviceBNew := "cccccccccccccccccccccccccccccccccccccccc"
	patch := "   - https://github.com/konflux-ci/service-a/config?ref=" + serviceARef + "\n" +
		"-  - https://github.com/konflux-ci/build-service/config?ref=" + serviceBOld + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config?ref=" + serviceBNew + "\n"
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    patch,
		},
	}
	bumps := changelog.ExtractServiceBumps(files)
	g.Expect(bumps).To(HaveLen(1))
	g.Expect(bumps[0].OldSHA).To(Equal(serviceBOld))
	g.Expect(bumps[0].NewSHA).To(Equal(serviceBNew))
}
