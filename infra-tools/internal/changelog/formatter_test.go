package changelog

import (
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

const (
	testOldSHA = "0bace1e20ff164a00e9d6becfce52e310a921931"
	testNewSHA = "9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f09"
)

func makeServiceChange(owner, repo, oldSHA, newSHA string, commits []CommitInfo) ServiceChange {
	return ServiceChange{
		Bump: ServiceBump{
			Owner:  owner,
			Repo:   repo,
			OldSHA: oldSHA,
			NewSHA: newSHA,
		},
		Commits: commits,
	}
}

// --- Format ---

func TestFormat_ContainsMarker(t *testing.T) {
	g := NewWithT(t)
	data := ChangelogData{OldSHA: testOldSHA, NewSHA: testNewSHA}
	result := Format(data)
	g.Expect(result).To(HavePrefix(CommentMarker))
}

func TestFormat_ContainsCompareLink(t *testing.T) {
	g := NewWithT(t)
	data := ChangelogData{OldSHA: testOldSHA, NewSHA: testNewSHA}
	result := Format(data)
	g.Expect(result).To(ContainSubstring("0bace1e20ff1"))
	g.Expect(result).To(ContainSubstring("9f8e7d6c5b4a"))
}

func TestFormat_WithServiceChanges(t *testing.T) {
	g := NewWithT(t)

	commits := []CommitInfo{
		{SHA: "abc1234500000000000000000000000000000000", Message: "fix: CVE patch", HTMLURL: "https://github.com/org/svc/commit/abc12345"},
	}
	data := ChangelogData{
		OldSHA: testOldSHA,
		NewSHA: testNewSHA,
		ServiceChanges: []ServiceChange{
			makeServiceChange("konflux-ci", "build-service",
				"211140a26c96e8f028a0595fb779ea13210ed5c8",
				"04a4744321a7fb747f796da783d51fc322aef598",
				commits),
		},
	}

	result := Format(data)
	g.Expect(result).To(ContainSubstring("build-service"))
	g.Expect(result).To(ContainSubstring("fix: CVE patch"))
	g.Expect(result).To(ContainSubstring("211140a26c96"))
	g.Expect(result).To(ContainSubstring("04a4744321a7"))
	g.Expect(result).To(ContainSubstring("compare"))
}

func TestFormat_ServiceChangeNoCommits(t *testing.T) {
	g := NewWithT(t)

	data := ChangelogData{
		OldSHA: testOldSHA,
		NewSHA: testNewSHA,
		ServiceChanges: []ServiceChange{
			makeServiceChange("konflux-ci", "namespace-lister",
				"bb70d9ec086170b825194574208c579263452743",
				"19c8344f1f897ba84a4182c087e84de9e8cd603b",
				nil),
		},
	}

	result := Format(data)
	g.Expect(result).To(ContainSubstring("No notable commits found"))
}

func TestFormat_WithOperatorNotableCommits(t *testing.T) {
	g := NewWithT(t)

	data := ChangelogData{
		OldSHA: testOldSHA,
		NewSHA: testNewSHA,
		OperatorResult: FilterResult{
			Notable: []CommitInfo{
				{SHA: "feat123400000000000000000000000000000000", Message: "feat: new CRD field", HTMLURL: "https://github.com/org/repo/commit/feat1234"},
				{SHA: "fix567800000000000000000000000000000000", Message: "fix: nil pointer", HTMLURL: "https://github.com/org/repo/commit/fix5678"},
			},
			Remaining: []CommitInfo{
				{SHA: "chore12300000000000000000000000000000000", Message: "chore: update deps", HTMLURL: "https://github.com/org/repo/commit/chore123"},
			},
		},
	}

	result := Format(data)
	g.Expect(result).To(ContainSubstring("feat: new CRD field"))
	g.Expect(result).To(ContainSubstring("fix: nil pointer"))
	g.Expect(result).To(ContainSubstring("chore: update deps"))
	g.Expect(result).To(ContainSubstring("<details>"))
	g.Expect(result).To(ContainSubstring("Other commits (1)"))
	g.Expect(result).To(ContainSubstring("</details>"))
}

func TestFormat_NoOperatorCommits(t *testing.T) {
	g := NewWithT(t)

	data := ChangelogData{OldSHA: testOldSHA, NewSHA: testNewSHA}
	result := Format(data)
	g.Expect(result).To(ContainSubstring("No operator commits found"))
	g.Expect(result).To(ContainSubstring("View full comparison"))
}

// --- FormatNoChange ---

func TestFormatNoChange_ContainsMarkerAndMessage(t *testing.T) {
	g := NewWithT(t)
	result := FormatNoChange()
	g.Expect(result).To(HavePrefix(CommentMarker))
	g.Expect(result).To(ContainSubstring("No operator SHA change detected"))
}

// --- FormatError ---

func TestFormatError_ContainsMarkerAndError(t *testing.T) {
	g := NewWithT(t)
	result := FormatError(errors.New("API rate limit exceeded"), "")
	g.Expect(result).To(HavePrefix(CommentMarker))
	g.Expect(result).To(ContainSubstring("API rate limit exceeded"))
	g.Expect(result).To(ContainSubstring("failed"))
}

func TestFormatError_IncludesRunURLWhenProvided(t *testing.T) {
	g := NewWithT(t)
	result := FormatError(errors.New("timeout"), "https://github.com/org/repo/actions/runs/123")
	g.Expect(result).To(ContainSubstring("https://github.com/org/repo/actions/runs/123"))
}

func TestFormatError_NoRunURLOmitsLink(t *testing.T) {
	g := NewWithT(t)
	result := FormatError(errors.New("timeout"), "")
	g.Expect(result).NotTo(ContainSubstring("View workflow run"))
}

// --- Structure sanity check ---

func TestFormat_MarkerIsFirstLine(t *testing.T) {
	g := NewWithT(t)
	data := ChangelogData{OldSHA: testOldSHA, NewSHA: testNewSHA}
	result := Format(data)
	firstLine := strings.SplitN(result, "\n", 2)[0]
	g.Expect(firstLine).To(Equal(CommentMarker))
}
