package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

func TestDiffFileName_WithClusterDir(t *testing.T) {
	g := NewWithT(t)

	cd := renderdiff.ComponentDiff{
		Path:       "components/foo/production",
		ClusterDir: "stone-prod-p02",
		Env:        "production",
	}

	g.Expect(diffFileName(cd)).To(Equal("stone-prod-p02--production.diff"))
}

func TestDiffFileName_WithoutClusterDir(t *testing.T) {
	g := NewWithT(t)

	cd := renderdiff.ComponentDiff{
		Path: "components/foo/staging",
		Env:  "staging",
	}

	g.Expect(diffFileName(cd)).To(Equal("components__foo__staging__staging.diff"))
}

func TestDedupeFileName_Unique(t *testing.T) {
	g := NewWithT(t)

	seen := make(map[string]int)

	g.Expect(dedupeFileName("foo.diff", seen)).To(Equal("foo.diff"))
	g.Expect(dedupeFileName("bar.diff", seen)).To(Equal("bar.diff"))
}

func TestDedupeFileName_Collision(t *testing.T) {
	g := NewWithT(t)

	seen := make(map[string]int)

	g.Expect(dedupeFileName("foo.diff", seen)).To(Equal("foo.diff"))
	g.Expect(dedupeFileName("foo.diff", seen)).To(Equal("foo-2.diff"))
	g.Expect(dedupeFileName("foo.diff", seen)).To(Equal("foo-3.diff"))
}

func TestWriteDiffFiles_WritesFiles(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/foo/staging",
				Env:   "staging",
				Added: 1, Removed: 0,
				Diff: "+new line\n",
			},
			{
				Path:       "components/bar/production",
				ClusterDir: "stone-prod-p01",
				Env:        "production",
				Added:      2, Removed: 1,
				Diff: "+added\n-removed\n",
			},
		},
		TotalAdded:   3,
		TotalRemoved: 1,
	}

	err := writeDiffFiles(result, dir)
	g.Expect(err).NotTo(HaveOccurred())

	content1, err := os.ReadFile(filepath.Join(dir, "components__foo__staging__staging.diff"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(content1)).To(Equal("+new line\n"))

	content2, err := os.ReadFile(filepath.Join(dir, "stone-prod-p01--production.diff"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(content2)).To(Equal("+added\n-removed\n"))
}

func TestWriteDiffFiles_SkipsErrorsAndEmptyDiffs(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	result := &renderdiff.DiffResult{
		Diffs: []renderdiff.ComponentDiff{
			{
				Path:  "components/broken/staging",
				Env:   "staging",
				Error: "build failed",
			},
			{
				Path: "components/same/staging",
				Env:  "staging",
				Diff: "",
			},
			{
				Path:  "components/good/staging",
				Env:   "staging",
				Added: 1,
				Diff:  "+line\n",
			},
		},
	}

	err := writeDiffFiles(result, dir)
	g.Expect(err).NotTo(HaveOccurred())

	entries, err := os.ReadDir(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Name()).To(Equal("components__good__staging__staging.diff"))
}

func TestSortDiffs(t *testing.T) {
	g := NewWithT(t)

	diffs := []renderdiff.ComponentDiff{
		{Path: "components/b/staging", Env: "staging"},
		{Path: "components/a/production", Env: "production"},
		{Path: "components/a/staging", Env: "staging"},
		{Path: "components/c/production", Env: "production"},
	}

	sortDiffs(diffs)

	g.Expect(diffs[0].Path).To(Equal("components/a/production"))
	g.Expect(diffs[1].Path).To(Equal("components/c/production"))
	g.Expect(diffs[2].Path).To(Equal("components/a/staging"))
	g.Expect(diffs[3].Path).To(Equal("components/b/staging"))
}
