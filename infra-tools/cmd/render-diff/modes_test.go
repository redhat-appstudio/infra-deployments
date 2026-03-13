package main

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

func TestParseOutputModes_Single(t *testing.T) {
	g := NewWithT(t)

	g.Expect(parseOutputModes("local")).To(Equal([]OutputMode{OutputModeLocal}))
	g.Expect(parseOutputModes("ci-summary")).To(Equal([]OutputMode{OutputModeCISummary}))
	g.Expect(parseOutputModes("ci-comment")).To(Equal([]OutputMode{OutputModeCIComment}))
	g.Expect(parseOutputModes("ci-artifact-dir")).To(Equal([]OutputMode{OutputModeCIArtifact}))
}

func TestParseOutputModes_Multiple(t *testing.T) {
	g := NewWithT(t)

	modes := parseOutputModes("ci-summary,ci-comment,ci-artifact-dir")
	g.Expect(modes).To(Equal([]OutputMode{OutputModeCISummary, OutputModeCIComment, OutputModeCIArtifact}))
}

func TestParseOutputModes_Deduplicates(t *testing.T) {
	g := NewWithT(t)

	modes := parseOutputModes("local,local,local")
	g.Expect(modes).To(Equal([]OutputMode{OutputModeLocal}))
}

func TestParseOutputModes_TrimsSpaces(t *testing.T) {
	g := NewWithT(t)

	modes := parseOutputModes(" ci-summary , ci-comment ")
	g.Expect(modes).To(Equal([]OutputMode{OutputModeCISummary, OutputModeCIComment}))
}

func TestParseOutputModes_Invalid(t *testing.T) {
	g := NewWithT(t)

	g.Expect(parseOutputModes("bogus")).To(BeNil())
	g.Expect(parseOutputModes("local,bogus")).To(BeNil())
}

func TestParseOutputModes_Empty(t *testing.T) {
	g := NewWithT(t)

	g.Expect(parseOutputModes("")).To(BeEmpty())
}

func TestRunAllOutputModes_RunsAllModes(t *testing.T) {
	g := NewWithT(t)

	// Use ci-summary mode with a temp file to verify it runs.
	f, err := os.CreateTemp(t.TempDir(), "summary-*.md")
	g.Expect(err).NotTo(HaveOccurred())
	_ = f.Close()
	t.Setenv("GITHUB_STEP_SUMMARY", f.Name())

	result := &renderdiff.DiffResult{}
	hadError := runAllOutputModes(context.Background(), []OutputMode{OutputModeCISummary}, result, "never", false, "", "", "")

	g.Expect(hadError).To(BeFalse())

	out, err := os.ReadFile(f.Name())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(out)).To(ContainSubstring("No render differences detected."))
}

func TestRunAllOutputModes_ReportsFailure(t *testing.T) {
	g := NewWithT(t)

	// ci-artifact-dir without --output-dir should fail.
	result := &renderdiff.DiffResult{}
	hadError := runAllOutputModes(context.Background(), []OutputMode{OutputModeCIArtifact}, result, "never", false, "", "", "")

	g.Expect(hadError).To(BeTrue())
}
