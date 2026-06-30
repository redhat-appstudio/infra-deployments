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
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
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
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(BeEmpty())
}

func TestExtractServiceBumps_SkipsUnchangedRef(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    buildServicePatch(buildOldSHA, buildOldSHA), // same SHA on both sides
		},
	}
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(BeEmpty())
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
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(HaveLen(1))
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
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(HaveLen(2))
}

// TestExtractServiceBumps_EmptyPatch verifies that an upstream kustomization with
// an empty patch (GitHub omits patch for very large or renamed files) sets the
// hasSkipped flag so callers can degrade gracefully instead of claiming "no bumps".
func TestExtractServiceBumps_EmptyPatch(t *testing.T) {
	g := NewWithT(t)
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/kustomization.yaml",
			Patch:    "",
		},
	}
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeTrue())
	g.Expect(bumps).To(BeEmpty())
}

// TestExtractServiceBumps_NoGitHubURL verifies that a kustomization file where
// the ?ref= SHA changes but the URL is not on github.com is skipped — the bump
// cannot be attributed to a GitHub repository.
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
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(BeEmpty())
}

// TestExtractServiceBumps_MultipleBumpsInOnePatch verifies that when a single
// kustomization file references multiple upstream services and both SHAs change,
// both bumps are detected — not just the first one.
func TestExtractServiceBumps_MultipleBumpsInOnePatch(t *testing.T) {
	g := NewWithT(t)
	intOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	intNewSHA := "dddddddddddddddddddddddddddddddddddddddd"
	// Both build-service and integration-service refs change in the same file.
	patch := "-  - https://github.com/konflux-ci/build-service/config?ref=" + buildOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config?ref=" + buildNewSHA + "\n" +
		"-  - https://github.com/konflux-ci/integration-service/config?ref=" + intOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/integration-service/config?ref=" + intNewSHA + "\n"
	files := []changelog.FileChange{
		{
			Filename: "operator/upstream-kustomizations/combined/kustomization.yaml",
			Patch:    patch,
		},
	}
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(HaveLen(2))

	byRepo := make(map[string]changelog.ServiceBump)
	for _, b := range bumps {
		byRepo[b.Repo] = b
	}
	g.Expect(byRepo).To(HaveKey("build-service"))
	g.Expect(byRepo["build-service"].OldSHA).To(Equal(buildOldSHA))
	g.Expect(byRepo["build-service"].NewSHA).To(Equal(buildNewSHA))
	g.Expect(byRepo).To(HaveKey("integration-service"))
	g.Expect(byRepo["integration-service"].OldSHA).To(Equal(intOldSHA))
	g.Expect(byRepo["integration-service"].NewSHA).To(Equal(intNewSHA))
}

// TestExtractServiceBumps_MultipleURLsSameFile verifies that when a kustomization
// file has two resource URLs and only one changes, the correct SHA pair is returned
// and the owner/repo is derived from the URL that actually changed — not from any
// other github.com URL that happens to appear on a + line.
func TestExtractServiceBumps_MultipleURLsSameFile(t *testing.T) {
	g := NewWithT(t)
	// service-a's SHA did NOT change (context line, no +/- prefix).
	// build-service's SHA DID change.
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
	bumps, skipped := changelog.ExtractServiceBumps(files)
	g.Expect(skipped).To(BeFalse())
	g.Expect(bumps).To(HaveLen(1))
	g.Expect(bumps[0].Repo).To(Equal("build-service"))
	g.Expect(bumps[0].OldSHA).To(Equal(serviceBOld))
	g.Expect(bumps[0].NewSHA).To(Equal(serviceBNew))
}
