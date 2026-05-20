package changelog

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	gh "github.com/google/go-github/v68/github"
)

// fakeRepoComparer implements RepoComparer for testing without network calls.
type fakeRepoComparer struct {
	comparison *gh.CommitsComparison
	err        error
}

func (f *fakeRepoComparer) CompareCommits(_ context.Context, _, _, _, _ string, _ *gh.ListOptions) (*gh.CommitsComparison, *gh.Response, error) {
	return f.comparison, &gh.Response{}, f.err
}

// makeCommit builds a minimal *gh.RepositoryCommit for use in tests.
func makeCommit(sha, message, htmlURL string, parentCount int) *gh.RepositoryCommit {
	parents := make([]*gh.Commit, parentCount)
	for i := range parents {
		parents[i] = &gh.Commit{}
	}
	return &gh.RepositoryCommit{
		SHA:     gh.Ptr(sha),
		HTMLURL: gh.Ptr(htmlURL),
		Commit: &gh.Commit{
			Message: gh.Ptr(message),
		},
		Parents: parents,
	}
}

func makeFile(filename, patch, status string) *gh.CommitFile {
	return &gh.CommitFile{
		Filename: gh.Ptr(filename),
		Patch:    gh.Ptr(patch),
		Status:   gh.Ptr(status),
	}
}

// --- FetchOperatorCompare tests ---

func TestFetchOperatorCompare_ReturnsCommitsAndFiles(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeRepoComparer{
		comparison: &gh.CommitsComparison{
			Commits: []*gh.RepositoryCommit{
				makeCommit("abc1234500000000000000000000000000000000", "feat: add thing", "https://github.com/org/repo/commit/abc12345", 1),
				makeCommit("def5678900000000000000000000000000000000", "fix: broken stuff", "https://github.com/org/repo/commit/def56789", 1),
			},
			Files: []*gh.CommitFile{
				makeFile("operator/foo.go", "@@ -1 +1 @@\n-old\n+new", "modified"),
			},
		},
	}

	result, err := FetchOperatorCompare(context.Background(), fake, "abc1234500000000000000000000000000000000", "def5678900000000000000000000000000000000")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Commits).To(HaveLen(2))
	g.Expect(result.Files).To(HaveLen(1))
	g.Expect(result.Commits[0].Subject()).To(Equal("feat: add thing"))
	g.Expect(result.Commits[1].Subject()).To(Equal("fix: broken stuff"))
	g.Expect(result.Files[0].Filename).To(Equal("operator/foo.go"))
	g.Expect(result.Files[0].Status).To(Equal("modified"))
}

func TestFetchOperatorCompare_APIError(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeRepoComparer{err: errors.New("API unavailable")}
	_, err := FetchOperatorCompare(context.Background(), fake, "abc1234500000000000000000000000000000000", "def5678900000000000000000000000000000000")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("konflux-ci/konflux-ci"))
}

func TestFetchOperatorCompare_EmptyResponse(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeRepoComparer{
		comparison: &gh.CommitsComparison{},
	}
	result, err := FetchOperatorCompare(context.Background(), fake, "abc1234500000000000000000000000000000000", "def5678900000000000000000000000000000000")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Commits).To(BeEmpty())
	g.Expect(result.Files).To(BeEmpty())
}

// --- FetchServiceCommits tests ---

func TestFetchServiceCommits_ReturnsCommits(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeRepoComparer{
		comparison: &gh.CommitsComparison{
			Commits: []*gh.RepositoryCommit{
				makeCommit("aaa0000000000000000000000000000000000000", "fix: CVE patch", "https://github.com/org/svc/commit/aaa", 1),
				makeCommit("bbb0000000000000000000000000000000000000", "Merge pull request #42", "https://github.com/org/svc/commit/bbb", 2),
			},
		},
	}

	commits, err := FetchServiceCommits(context.Background(), fake, "konflux-ci", "build-service", "aaa0000000000000000000000000000000000000", "bbb0000000000000000000000000000000000000")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).To(HaveLen(2))
	g.Expect(commits[0].IsMerge).To(BeFalse())
	g.Expect(commits[1].IsMerge).To(BeTrue())
}

func TestFetchServiceCommits_APIError(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeRepoComparer{err: errors.New("not found")}
	_, err := FetchServiceCommits(context.Background(), fake, "konflux-ci", "build-service", "aaa0000000000000000000000000000000000000", "bbb0000000000000000000000000000000000000")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("build-service"))
}

// --- CommitInfo.Subject tests ---

func TestCommitInfo_Subject_SingleLine(t *testing.T) {
	g := NewWithT(t)
	c := CommitInfo{Message: "feat: single line message"}
	g.Expect(c.Subject()).To(Equal("feat: single line message"))
}

func TestCommitInfo_Subject_MultiLine(t *testing.T) {
	g := NewWithT(t)
	c := CommitInfo{Message: "feat: subject line\n\nBody of commit message.\nMore details here."}
	g.Expect(c.Subject()).To(Equal("feat: subject line"))
}

// --- convertCommits tests ---

func TestConvertCommits_SkipsNilEntries(t *testing.T) {
	g := NewWithT(t)
	raw := []*gh.RepositoryCommit{
		nil,
		makeCommit("abc0000000000000000000000000000000000000", "fix: real commit", "https://example.com", 1),
		nil,
	}
	result := convertCommits(raw)
	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].Subject()).To(Equal("fix: real commit"))
}
