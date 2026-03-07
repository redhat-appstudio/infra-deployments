package main

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

func TestBuildCommentBody_WithBuildError(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/broken/staging",
				Env:   "staging",
				Error: "kustomize build /tmp/broken: accumulating resources: resource not found",
			},
		},
	}

	body := buildCommentBody(result, "abc123", "def456", "")

	g.Expect(body).To(ContainSubstring("build error"))
	g.Expect(body).To(ContainSubstring("`components/broken/staging`"))
	g.Expect(body).To(ContainSubstring("staging"))
}

func TestBuildCommentBody_SkipsNonKustomizationError(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:       "components/plain/staging",
				Env:        "staging",
				Error:      "unable to find one of 'kustomization.yaml' in directory '/tmp/plain'",
				SkipOutput: true,
			},
			{
				Path:  "components/foo/staging",
				Env:   "staging",
				Added: 5, Removed: 2,
				Diff: "some diff",
			},
		},
		TotalAdded:   5,
		TotalRemoved: 2,
	}

	body := buildCommentBody(result, "abc123", "def456", "")

	g.Expect(body).NotTo(ContainSubstring("components/plain/staging"))
	g.Expect(body).To(ContainSubstring("`components/foo/staging`"))
}

func TestBuildCommentBody_NoDiffs(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{}

	body := buildCommentBody(result, "abc123", "def456", "")

	g.Expect(body).To(ContainSubstring("No render differences detected."))
}

func TestBuildCommentBody_MixedDiffsAndErrors(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/broken/staging",
				Env:   "staging",
				Error: "accumulating resources: resource not found",
			},
			{
				Path:  "components/foo/staging",
				Env:   "staging",
				Added: 3, Removed: 1,
				Diff: "some diff",
			},
		},
		TotalAdded:   3,
		TotalRemoved: 1,
	}

	body := buildCommentBody(result, "abc123", "def456", "")

	g.Expect(body).To(ContainSubstring("| `components/broken/staging` | staging | build error |"))
	g.Expect(body).To(ContainSubstring("| `components/foo/staging` | staging | +3 -1 |"))
}

func TestBuildCommentBody_WithRunURL(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{Path: "components/foo/staging", Env: "staging", Added: 3, Removed: 1, Diff: "some diff"},
		},
		TotalAdded:   3,
		TotalRemoved: 1,
	}

	body := buildCommentBody(result, "abc123", "def456", "https://github.com/owner/repo/actions/runs/99999")

	g.Expect(body).To(ContainSubstring("[workflow summary](https://github.com/owner/repo/actions/runs/99999)"))
	g.Expect(body).NotTo(ContainSubstring("../actions"))
}

func TestBuildCommentBody_WithoutRunURL(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{Path: "components/foo/staging", Env: "staging", Added: 3, Removed: 1, Diff: "some diff"},
		},
		TotalAdded:   3,
		TotalRemoved: 1,
	}

	body := buildCommentBody(result, "abc123", "def456", "")

	g.Expect(body).To(ContainSubstring("[workflow summary](../actions)"))
}

func TestBuildRunURL(t *testing.T) {
	g := NewWithT(t)

	g.Expect(buildRunURL("https://github.com", "owner/repo", "12345")).
		To(Equal("https://github.com/owner/repo/actions/runs/12345"))

	g.Expect(buildRunURL("", "owner/repo", "12345")).To(BeEmpty())
	g.Expect(buildRunURL("https://github.com", "", "12345")).To(BeEmpty())
	g.Expect(buildRunURL("https://github.com", "owner/repo", "")).To(BeEmpty())
}

// writeCISummaryToString is a helper that captures writeCISummary output to a string
// by pointing GITHUB_STEP_SUMMARY at a temp file.
func writeCISummaryToString(t *testing.T, result *renderdiff.DiffResult) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "summary-*.md")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	t.Setenv("GITHUB_STEP_SUMMARY", f.Name())

	if err := writeCISummary(result); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestWriteCISummary_NoDiffs(t *testing.T) {
	g := NewWithT(t)

	body := writeCISummaryToString(t, &renderdiff.DiffResult{})

	g.Expect(body).To(ContainSubstring("No render differences detected."))
}

func TestWriteCISummary_WithDiff(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/foo/staging",
				Env:   "staging",
				Added: 3, Removed: 1,
				Diff: "+added line\n-removed line\n",
			},
		},
		TotalAdded:   3,
		TotalRemoved: 1,
	}

	body := writeCISummaryToString(t, result)

	g.Expect(body).To(ContainSubstring("# Kustomize Render Diff"))
	g.Expect(body).To(ContainSubstring("**1 components** with differences"))
	g.Expect(body).To(ContainSubstring("components/foo/staging (staging) — +3 -1"))
	g.Expect(body).To(ContainSubstring("```diff"))
}

func TestWriteCISummary_WithBuildError(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/broken/staging",
				Env:   "staging",
				Error: "accumulating resources: resource not found",
			},
		},
	}

	body := writeCISummaryToString(t, result)

	g.Expect(body).To(ContainSubstring("components/broken/staging (staging) — build error"))
	g.Expect(body).To(ContainSubstring("accumulating resources: resource not found"))
}

func TestWriteCISummary_SkipsNonKustomizationError(t *testing.T) {
	g := NewWithT(t)

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:       "components/plain/staging",
				Env:        "staging",
				Error:      "unable to find one of 'kustomization.yaml'",
				SkipOutput: true,
			},
			{
				Path:  "components/foo/staging",
				Env:   "staging",
				Added: 2, Removed: 0,
				Diff: "+new line\n",
			},
		},
		TotalAdded: 2,
	}

	body := writeCISummaryToString(t, result)

	g.Expect(body).NotTo(ContainSubstring("components/plain/staging"))
	g.Expect(body).To(ContainSubstring("components/foo/staging"))
}

func TestWriteCISummary_TruncatesLargeDiff(t *testing.T) {
	g := NewWithT(t)

	largeDiff := strings.Repeat("x", 60*1024) // 60KB, above 50KB threshold

	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/big/staging",
				Env:   "staging",
				Added: 1000, Removed: 0,
				Diff: largeDiff,
			},
		},
		TotalAdded: 1000,
	}

	body := writeCISummaryToString(t, result)

	g.Expect(body).To(ContainSubstring("Diff truncated"))
	g.Expect(len(body)).To(BeNumerically("<", len(largeDiff)))
}
