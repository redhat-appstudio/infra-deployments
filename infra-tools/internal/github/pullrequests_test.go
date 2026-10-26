package github //nolint:testpackage // injects the fake pull-request service

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v68/github"
	. "github.com/onsi/gomega"
)

type fakePullRequestsService struct {
	pr        *gh.PullRequest
	pages     map[int][]*gh.RepositoryCommit
	getErr    error
	listErr   error
	getCalls  int
	listCalls int
}

func (f *fakePullRequestsService) Get(_ context.Context, _, _ string, _ int) (*gh.PullRequest, *gh.Response, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, nil, f.getErr
	}
	return f.pr, nil, nil
}

func (f *fakePullRequestsService) ListCommits(_ context.Context, _, _ string, _ int, opts *gh.ListOptions) ([]*gh.RepositoryCommit, *gh.Response, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, nil, f.listErr
	}
	page := 1
	if opts != nil && opts.Page > 0 {
		page = opts.Page
	}
	resp := &gh.Response{}
	if _, ok := f.pages[page+1]; ok {
		resp.NextPage = page + 1
	}
	return f.pages[page], resp, nil
}

func kargoPR() *gh.PullRequest {
	return &gh.PullRequest{
		User: &gh.User{Login: gh.Ptr(KargoBot)},
		Head: &gh.PullRequestBranch{
			Ref:  gh.Ptr("kargo-konflux-core/foo/ring-1"),
			SHA:  gh.Ptr("head-sha"),
			Repo: &gh.Repository{FullName: gh.Ptr("o/r")},
		},
		Base: &gh.PullRequestBranch{Repo: &gh.Repository{FullName: gh.Ptr("o/r")}},
	}
}

func repoCommit(author, committer string, verified bool) *gh.RepositoryCommit {
	return repoCommitSHA(author, committer, "head-sha", verified)
}

func repoCommitSHA(author, committer, sha string, verified bool) *gh.RepositoryCommit {
	return &gh.RepositoryCommit{
		SHA:       gh.Ptr(sha),
		Author:    &gh.User{Login: gh.Ptr(author)},
		Committer: &gh.User{Login: gh.Ptr(committer)},
		Commit: &gh.Commit{
			Verification: &gh.SignatureVerification{Verified: gh.Ptr(verified)},
		},
	}
}

func TestKargoSoleCommitter_VerifiedPromotionPaginates(t *testing.T) {
	g := NewWithT(t)

	fake := &fakePullRequestsService{
		pr: kargoPR(),
		pages: map[int][]*gh.RepositoryCommit{
			2: {repoCommit(KargoBot, "web-flow", true)},
		},
	}
	client := &Client{pullRequests: fake, owner: "o", repo: "r"}

	exempt, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(exempt).To(BeTrue())
	g.Expect(fake.listCalls).To(Equal(2))
}

func TestKargoSoleCommitter_FailsClosed(t *testing.T) {
	g := NewWithT(t)

	cases := []struct {
		name    string
		pr      *gh.PullRequest
		commits []*gh.RepositoryCommit
	}{
		{"human PR author", &gh.PullRequest{
			User: &gh.User{Login: gh.Ptr("someone")},
			Head: &gh.PullRequestBranch{Repo: &gh.Repository{FullName: gh.Ptr("o/r")}},
			Base: &gh.PullRequestBranch{Repo: &gh.Repository{FullName: gh.Ptr("o/r")}},
		}, []*gh.RepositoryCommit{repoCommit(KargoBot, "web-flow", true)}},
		{"fork head", &gh.PullRequest{
			User: &gh.User{Login: gh.Ptr(KargoBot)},
			Head: &gh.PullRequestBranch{Repo: &gh.Repository{FullName: gh.Ptr("fork/r")}},
			Base: &gh.PullRequestBranch{Repo: &gh.Repository{FullName: gh.Ptr("o/r")}},
		}, []*gh.RepositoryCommit{repoCommit(KargoBot, "web-flow", true)}},
		{"human commit author", kargoPR(), []*gh.RepositoryCommit{repoCommit("someone", "web-flow", true)}},
		{"untrusted committer", kargoPR(), []*gh.RepositoryCommit{repoCommit(KargoBot, "someone", true)}},
		{"unverified commit", kargoPR(), []*gh.RepositoryCommit{repoCommit(KargoBot, "web-flow", false)}},
		{"multiple commits", kargoPR(), []*gh.RepositoryCommit{
			repoCommit(KargoBot, "web-flow", true),
			repoCommit(KargoBot, "web-flow", true),
		}},
		{"no commits", kargoPR(), nil},
	}

	for _, tc := range cases {
		fake := &fakePullRequestsService{pr: tc.pr, pages: map[int][]*gh.RepositoryCommit{1: tc.commits}}
		client := &Client{pullRequests: fake, owner: "o", repo: "r"}
		exempt, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
		g.Expect(err).NotTo(HaveOccurred(), tc.name)
		g.Expect(exempt).To(BeFalse(), tc.name)
	}
}

func TestKargoSoleCommitter_CommitHeadMismatch(t *testing.T) {
	g := NewWithT(t)

	fake := &fakePullRequestsService{
		pr:    kargoPR(),
		pages: map[int][]*gh.RepositoryCommit{1: {repoCommitSHA(KargoBot, "web-flow", "other", true)}},
	}
	client := &Client{pullRequests: fake, owner: "o", repo: "r"}

	exempt, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(exempt).To(BeFalse())
}

func TestKargoSoleCommitter_AcceptsKargoCommitter(t *testing.T) {
	g := NewWithT(t)

	fake := &fakePullRequestsService{
		pr:    kargoPR(),
		pages: map[int][]*gh.RepositoryCommit{1: {repoCommit(KargoBot, KargoBot, true)}},
	}
	client := &Client{pullRequests: fake, owner: "o", repo: "r"}

	exempt, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(exempt).To(BeTrue())
}

func TestPRHeadSHA(t *testing.T) {
	g := NewWithT(t)

	client := &Client{pullRequests: &fakePullRequestsService{pr: kargoPR()}, owner: "o", repo: "r"}
	sha, err := client.PRHeadSHA(context.Background(), 1)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sha).To(Equal("head-sha"))
}

func TestKargoSoleCommitter_RejectsNonKargoPRAuthor(t *testing.T) {
	g := NewWithT(t)

	for _, author := range []string{"someone", "konflux-kargo-bot"} {
		pr := kargoPR()
		pr.User.Login = gh.Ptr(author)
		fake := &fakePullRequestsService{
			pr:    pr,
			pages: map[int][]*gh.RepositoryCommit{1: {repoCommit(KargoBot, "web-flow", true)}},
		}
		client := &Client{pullRequests: fake, owner: "o", repo: "r"}

		exempt, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
		g.Expect(err).NotTo(HaveOccurred(), author)
		g.Expect(exempt).To(BeFalse(), author)
	}
}

func TestKargoSoleCommitter_RejectsNonKargoSender(t *testing.T) {
	g := NewWithT(t)

	fake := &fakePullRequestsService{
		pr:    kargoPR(),
		pages: map[int][]*gh.RepositoryCommit{1: {repoCommit(KargoBot, "web-flow", true)}},
	}
	client := &Client{pullRequests: fake, owner: "o", repo: "r"}

	exempt, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", "someone")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(exempt).To(BeFalse())
}

func TestKargoSoleCommitter_HeadBinding(t *testing.T) {
	g := NewWithT(t)

	commits := map[int][]*gh.RepositoryCommit{1: {repoCommit(KargoBot, "web-flow", true)}}

	for _, tc := range []struct {
		name    string
		mutate  func(*gh.PullRequest)
		headSHA string
	}{
		{"wrong head sha", func(*gh.PullRequest) {}, "other"},
		{"empty head sha", func(*gh.PullRequest) {}, ""},
		{"non-promotion branch", func(pr *gh.PullRequest) { pr.Head.Ref = gh.Ptr("feature/foo") }, "head-sha"},
		{"fork", func(pr *gh.PullRequest) { pr.Head.Repo.FullName = gh.Ptr("fork/r") }, "head-sha"},
	} {
		pr := kargoPR()
		tc.mutate(pr)
		client := &Client{
			pullRequests: &fakePullRequestsService{pr: pr, pages: commits},
			owner:        "o",
			repo:         "r",
		}
		exempt, err := client.KargoSoleCommitter(context.Background(), 1, tc.headSHA, KargoBot)
		g.Expect(err).NotTo(HaveOccurred(), tc.name)
		g.Expect(exempt).To(BeFalse(), tc.name)
	}
}

func TestKargoSoleCommitter_Errors(t *testing.T) {
	g := NewWithT(t)

	client := &Client{
		pullRequests: &fakePullRequestsService{getErr: errors.New("get")},
		owner:        "o",
		repo:         "r",
	}
	_, err := client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
	g.Expect(err).To(MatchError("getting PR #1: get"))

	client = &Client{
		pullRequests: &fakePullRequestsService{pr: kargoPR(), listErr: errors.New("list")},
		owner:        "o",
		repo:         "r",
	}
	_, err = client.KargoSoleCommitter(context.Background(), 1, "head-sha", KargoBot)
	g.Expect(err).To(MatchError("listing commits for PR #1: list"))
}

func TestIsKargoBot(t *testing.T) {
	g := NewWithT(t)

	g.Expect(IsKargoBot(KargoBot)).To(BeTrue())
	g.Expect(IsKargoBot("konflux-kargo-bot")).To(BeFalse())
	g.Expect(IsKargoBot("someone")).To(BeFalse())
}
