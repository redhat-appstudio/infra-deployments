package changelog_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	gh "github.com/google/go-github/v68/github"
	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
)

// makeCommit creates a RepositoryCommit with the given SHA and message.
func makeCommit(sha, message string) *gh.RepositoryCommit {
	return &gh.RepositoryCommit{
		SHA:    gh.Ptr(sha),
		Commit: &gh.Commit{Message: gh.Ptr(message)},
	}
}

// testBump is a reusable ServiceBump for commit tests.
var testBump = changelog.ServiceBump{
	Owner:  "konflux-ci",
	Repo:   "build-service",
	OldSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	NewSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
}

func TestFetchServiceCommits_ReturnsFilteredCommits(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{
		commits: []*gh.RepositoryCommit{
			makeCommit("cccccccccccc", "feat: add retry support"),
			makeCommit("dddddddddddd", "fix(pipeline): correct timeout"),
			makeCommit("eeeeeeeeeeee", "chore: update deps"),
			makeCommit("ffffffffffff", "Merge pull request #123"),
		},
	}
	commits, truncated, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(truncated).To(BeFalse())
	g.Expect(commits).To(HaveLen(2))
	g.Expect(commits[0].Type).To(Equal("feat"))
	g.Expect(commits[0].Subject).To(Equal("add retry support"))
	g.Expect(commits[0].Scope).To(BeEmpty())
	g.Expect(commits[1].Type).To(Equal("fix"))
	g.Expect(commits[1].Scope).To(Equal("pipeline"))
	g.Expect(commits[1].Subject).To(Equal("correct timeout"))
}

func TestFetchServiceCommits_EmptyWhenNoConventional(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{
		commits: []*gh.RepositoryCommit{
			makeCommit("aaaa", "chore: update deps"),
			makeCommit("bbbb", "Update readme"),
			makeCommit("cccc", "Merge pull request #123"),
		},
	}
	commits, _, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).To(BeEmpty())
}

func TestFetchServiceCommits_TruncatedWhenAheadByMax(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{aheadBy: 250}
	_, truncated, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(truncated).To(BeTrue())
}

func TestFetchServiceCommits_NotTruncatedBeforeMax(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{aheadBy: 249}
	_, truncated, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(truncated).To(BeFalse())
}

func TestFetchServiceCommits_APIError(t *testing.T) {
	g := NewWithT(t)
	// failTimes=3 — all attempts fail, error is returned with service name in message.
	fake := &fakeComparer{err: errors.New("network error"), failTimes: 3}
	_, _, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("build-service"))
	g.Expect(fake.calls).To(Equal(3))
}

// TestFetchServiceCommits_NonRetryableErrorFastFails verifies that a 404 does
// not trigger retries — only one call should be made.
func TestFetchServiceCommits_NonRetryableErrorFastFails(t *testing.T) {
	g := NewWithT(t)
	notFound := &gh.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
	fake := &fakeComparer{err: notFound, failTimes: 3}
	_, _, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).To(HaveOccurred())
	g.Expect(fake.calls).To(Equal(1))
}

// TestFetchServiceCommits_TakesFirstLineOnly verifies that only the subject
// line of a multi-paragraph commit message is parsed.
func TestFetchServiceCommits_TakesFirstLineOnly(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{
		commits: []*gh.RepositoryCommit{
			makeCommit("aaaa", "feat: add feature\n\nThis is the body.\nWith multiple lines."),
		},
	}
	commits, _, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).To(HaveLen(1))
	g.Expect(commits[0].Subject).To(Equal("add feature"))
}

// TestFetchServiceCommits_BreakingChangeStripsExclamation verifies that the
// breaking change marker (!) is stripped and the commit is still captured.
func TestFetchServiceCommits_BreakingChangeStripsExclamation(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{
		commits: []*gh.RepositoryCommit{
			makeCommit("aaaa", "feat!: breaking API change"),
		},
	}
	commits, _, err := changelog.FetchServiceCommits(context.Background(), fake, testBump)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).To(HaveLen(1))
	g.Expect(commits[0].Type).To(Equal("feat"))
	g.Expect(commits[0].Subject).To(Equal("breaking API change"))
}
