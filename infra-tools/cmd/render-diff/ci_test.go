package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

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
