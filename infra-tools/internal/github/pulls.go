package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v68/github"
)

// PullsService is the subset of the GitHub Pull Requests API used for bot
// verification.
type PullsService interface {
	Get(ctx context.Context, owner, repo string, number int) (*gh.PullRequest, *gh.Response, error)
	ListCommits(ctx context.Context, owner, repo string, number int, opts *gh.ListOptions) ([]*gh.RepositoryCommit, *gh.Response, error)
}

// KargoCheckResult holds the outcome and a human-readable reason from
// IsKargoAutomation.
type KargoCheckResult struct {
	IsBot  bool
	Reason string
}

// maxInspectableCommits is the GitHub API hard cap on commits returned by the
// list-commits endpoint. PRs exceeding this cannot be fully verified.
const maxInspectableCommits = 250

// IsKargoAutomation returns true only when ALL of the following hold:
//
//  1. KargoBotAccountID is non-zero (the constant has been populated).
//  2. The PR opener's login, immutable account ID, and account type match the
//     known Kargo bot constants.
//  3. The PR has at least one commit and no more than maxInspectableCommits.
//  4. The number of commits returned by the API matches pr.Commits (no hidden
//     commits beyond the API cap).
//  5. Every commit was authored by the Kargo bot account ID and carries a
//     verified signature with reason "valid".
//
// Any error from the GitHub API is returned to the caller; the caller MUST
// treat errors as fail-closed (apply the hold labels).
func (c *Client) IsKargoAutomation(ctx context.Context, prNumber int) (KargoCheckResult, error) {

	// Guard: the account-ID constant must be populated before this is useful.
	// If it is still 0 (the default placeholder), skip the exemption entirely
	// so a mis-configured deployment doesn't accidentally skip the hold labels.
	if KargoBotAccountID == 0 {
		return KargoCheckResult{
			IsBot:  false,
			Reason: "KargoBotAccountID constant is 0 (not yet configured); treating PR as non-bot",
		}, nil
	}

	// Fetch the PR once; both checks below consume it.
	pr, _, err := c.pulls.Get(ctx, c.owner, c.repo, prNumber)
	if err != nil {
		return KargoCheckResult{}, fmt.Errorf("fetching PR #%d: %w", prNumber, err)
	}

	if !isKargoBotUser(pr.GetUser()) {
		return KargoCheckResult{
			IsBot:  false,
			Reason: "PR opener is not the Kargo bot",
		}, nil
	}

	reason, err := c.verifyAllCommits(ctx, pr)
	if err != nil {
		return KargoCheckResult{}, fmt.Errorf("verifying PR commits: %w", err)
	}
	if reason != "" {
		return KargoCheckResult{IsBot: false, Reason: reason}, nil
	}

	return KargoCheckResult{
		IsBot:  true,
		Reason: fmt.Sprintf("PR opener is the Kargo bot and all %d commits are authored by the Kargo bot with a valid signature", pr.GetCommits()),
	}, nil
}

// isKargoBotUser returns true when u matches all three Kargo bot identity
// fields: login, immutable numeric account ID, and account type.
func isKargoBotUser(u *gh.User) bool {
	return u.GetLogin() == KargoBotLogin &&
		u.GetID() == KargoBotAccountID &&
		u.GetType() == "Bot"
}

// verifyAllCommits paginates through every commit on pr and returns a
// non-empty reason string if any commit fails the Kargo bot authorship or
// signature checks. It fails when:
//   - the PR has zero commits
//   - the PR has more than maxInspectableCommits commits (API hard cap)
//   - the number of commits seen across all pages differs from pr.GetCommits()
//   - at least one commit is not authored by the Kargo bot
//   - at least one commit does not have a cryptographically valid signature
func (c *Client) verifyAllCommits(ctx context.Context, pr *gh.PullRequest) (string, error) {
	prNumber := pr.GetNumber()
	declaredCount := pr.GetCommits()

	if declaredCount == 0 {
		return "PR has no commits; cannot verify bot authorship", nil
	}
	if declaredCount > maxInspectableCommits {
		return fmt.Sprintf("PR has %d commits, which exceeds the API cap of %d; cannot fully verify bot authorship", declaredCount, maxInspectableCommits), nil
	}

	opts := &gh.ListOptions{PerPage: 100}
	seen := 0
	for {
		commits, resp, err := c.pulls.ListCommits(ctx, c.owner, c.repo, prNumber, opts)
		if err != nil {
			return "", fmt.Errorf("listing commits (page %d): %w", opts.Page, err)
		}

		for _, rc := range commits {
			seen++
			sha := rc.GetSHA()
			if len(sha) > 7 {
				sha = sha[:7]
			}

			// Author must be the Kargo bot account.
			// rc.GetAuthor() is the GitHub user linked to the git author e-mail;
			// it is nil when the e-mail is not registered to any account.
			if rc.GetAuthor().GetID() != KargoBotAccountID {
				return fmt.Sprintf("commit %s: author account ID %d does not match Kargo bot ID %d", sha, rc.GetAuthor().GetID(), KargoBotAccountID), nil
			}

			// Require a cryptographically valid signature.
			v := rc.GetCommit().GetVerification()
			if !v.GetVerified() || v.GetReason() != "valid" {
				return fmt.Sprintf("commit %s: signature not valid (verified=%v reason=%q)", sha, v.GetVerified(), v.GetReason()), nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Guard against commits that fell outside the API window.
	if seen != declaredCount {
		return fmt.Sprintf("inspected %d commits but PR declares %d; cannot fully verify bot authorship", seen, declaredCount), nil
	}

	return "", nil
}
