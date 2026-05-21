package changelog

import (
	"testing"

	. "github.com/onsi/gomega"
)

func makeCI(subject string, isMerge bool) CommitInfo {
	return CommitInfo{
		SHA:     "abc0000000000000000000000000000000000000",
		Message: subject,
		HTMLURL: "https://example.com",
		IsMerge: isMerge,
	}
}

// --- ConventionalCommitsFilter ---

func TestConventionalCommitsFilter_KeepsFeatAndFix(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("feat: add new thing", false),
		makeCI("fix: broken stuff", false),
		makeCI("feat(scope): scoped feature", false),
		makeCI("fix(#123): issue fix", false),
	}
	result := ConventionalCommitsFilter(commits)
	g.Expect(result).To(HaveLen(4))
}

func TestConventionalCommitsFilter_ExcludesChoreAndDocs(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("chore: update deps", false),
		makeCI("chore(deps): renovate update", false),
		makeCI("docs: update readme", false),
		makeCI("ci: add workflow", false),
		makeCI("refactor: clean up", false),
		makeCI("test: add tests", false),
	}
	result := ConventionalCommitsFilter(commits)
	g.Expect(result).To(BeEmpty())
}

func TestConventionalCommitsFilter_ExcludesMergeCommits(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("feat: real feature", false),
		makeCI("Merge pull request #42 from foo/bar", true),
		makeCI("feat: another feature", true), // merge commit even with feat prefix
	}
	result := ConventionalCommitsFilter(commits)
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].Subject()).To(Equal("feat: real feature"))
}

func TestConventionalCommitsFilter_EmptyInput(t *testing.T) {
	g := NewWithT(t)
	result := ConventionalCommitsFilter(nil)
	g.Expect(result).To(BeEmpty())
}

func TestConventionalCommitsFilter_CaseInsensitive(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("Feat: uppercase feature", false),
		makeCI("FIX: uppercase fix", false),
	}
	result := ConventionalCommitsFilter(commits)
	g.Expect(result).To(HaveLen(2))
}

// --- FilterOperatorCommits ---

func TestFilterOperatorCommits_SplitsIntoTiers(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("feat: new feature", false),
		makeCI("fix: bug fix", false),
		makeCI("chore: deps update", false),
		makeCI("chore: sync manifests", false),
		makeCI("Merge pull request #1", true),
	}

	result := FilterOperatorCommits(commits)

	// Notable: only feat and fix
	g.Expect(result.Notable).To(HaveLen(2))
	g.Expect(result.Notable[0].Subject()).To(Equal("feat: new feature"))
	g.Expect(result.Notable[1].Subject()).To(Equal("fix: bug fix"))

	// Remaining: non-merge, non-notable only (the two chore commits)
	g.Expect(result.Remaining).To(HaveLen(2))
	g.Expect(result.Remaining[0].Subject()).To(Equal("chore: deps update"))
	g.Expect(result.Remaining[1].Subject()).To(Equal("chore: sync manifests"))
}

func TestFilterOperatorCommits_AllNotable(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("feat: feature", false),
		makeCI("fix: fix", false),
	}
	result := FilterOperatorCommits(commits)
	// When all commits are feat/fix, Remaining is empty
	g.Expect(result.Notable).To(HaveLen(2))
	g.Expect(result.Remaining).To(BeEmpty())
}

func TestFilterOperatorCommits_EmptyInput(t *testing.T) {
	g := NewWithT(t)
	result := FilterOperatorCommits(nil)
	g.Expect(result.Notable).To(BeEmpty())
	g.Expect(result.Remaining).To(BeEmpty())
}

func TestFilterOperatorCommits_OnlyMergeCommits(t *testing.T) {
	g := NewWithT(t)
	commits := []CommitInfo{
		makeCI("Merge pull request #1", true),
		makeCI("Merge pull request #2", true),
	}
	result := FilterOperatorCommits(commits)
	g.Expect(result.Notable).To(BeEmpty())
	g.Expect(result.Remaining).To(BeEmpty())
}
