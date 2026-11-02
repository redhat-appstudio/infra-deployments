package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

// PullRequestsService is the subset of the GitHub Pull Requests API used here.
type PullRequestsService interface {
	Get(ctx context.Context, owner, repo string, number int) (*gh.PullRequest, *gh.Response, error)
	ListCommits(ctx context.Context, owner, repo string, number int, opts *gh.ListOptions) ([]*gh.RepositoryCommit, *gh.Response, error)
}

// PRHeadSHA returns the current head SHA of the PR.
func (c *Client) PRHeadSHA(ctx context.Context, prNumber int) (string, error) {
	pr, _, err := c.pullRequests.Get(ctx, c.owner, c.repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("getting PR #%d: %w", prNumber, err)
	}
	return pr.GetHead().GetSHA(), nil
}

// KargoSoleCommitter reports whether the PR is a same-repository promotion
// authored and pushed by Kargo, with a single verified commit on a Kargo
// promotion branch, committed by GitHub's web-flow signer or by Kargo itself,
// and a head matching expectedHeadSHA. Promotion branches are assumed to be
// writable only by the Kargo automation; a repository writer could otherwise
// replace the branch and bypass this check.
func (c *Client) KargoSoleCommitter(ctx context.Context, prNumber int, expectedHeadSHA, sender string) (bool, error) {
	if !IsKargoBot(sender) {
		return false, nil
	}
	pr, _, err := c.pullRequests.Get(ctx, c.owner, c.repo, prNumber)
	if err != nil {
		return false, fmt.Errorf("getting PR #%d: %w", prNumber, err)
	}
	if !IsKargoBot(pr.GetUser().GetLogin()) {
		return false, nil
	}
	if pr.GetHead().GetRepo().GetFullName() != pr.GetBase().GetRepo().GetFullName() {
		return false, nil
	}
	if expectedHeadSHA == "" || pr.GetHead().GetSHA() != expectedHeadSHA {
		return false, nil
	}
	headRef := pr.GetHead().GetRef()
	// Promotion branches follow <project>/<component>/<ring>.
	parts := strings.Split(headRef, "/")
	if len(parts) != 3 || !strings.HasPrefix(parts[0], "kargo-") ||
		!strings.HasPrefix(parts[2], "ring-") {
		return false, nil
	}

	commits := 0
	opts := &gh.ListOptions{PerPage: 100}
	for {
		page, resp, err := c.pullRequests.ListCommits(ctx, c.owner, c.repo, prNumber, opts)
		if err != nil {
			return false, fmt.Errorf("listing commits for PR #%d: %w", prNumber, err)
		}
		for _, commit := range page {
			commits++
			if commit.GetSHA() != expectedHeadSHA {
				return false, nil
			}
			if !IsKargoBot(commit.GetAuthor().GetLogin()) {
				return false, nil
			}
			committer := commit.GetCommitter().GetLogin()
			if committer != "web-flow" && !IsKargoBot(committer) {
				return false, nil
			}
			if !commit.GetCommit().GetVerification().GetVerified() {
				return false, nil
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return commits == 1, nil
		}
		opts.Page = resp.NextPage
	}
}
