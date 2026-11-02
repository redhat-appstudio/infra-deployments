// Package github provides helpers for managing PR labels via the GitHub API.
package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

// LabelPrefixes are the prefixes managed by this tool. Labels with these
// prefixes will be added/removed as needed.
var LabelPrefixes = []string{
	"environment/",
	"cluster/",
	"infra/",
	"prod/",
}

// HoldProductionLabel is applied when the PR affects the production
// environment.  Prow Tide can be configured to refuse merging PRs that
// carry this label, ensuring a human explicitly removes it after review.
const HoldProductionLabel = "infra/hold-production"

// NeedsApprovalProductionLabel is applied when the PR affects the production
// environment, signalling that explicit approval is required before merging.
const NeedsApprovalProductionLabel = "prod/needs-approval"

// UnreferencedChangesLabel is applied when the PR changes files under
// components/ or configs/ that no ApplicationSet references. Such changes are
// not deployed by anything and are usually a sign of editing dead code.
const UnreferencedChangesLabel = "infra/unreferenced-changes"

// ApprovedProductionLabel records that a human approved a production change.
// This tool never applies it: "prod/" is a managed prefix, so the label is
// removed on every label sync and a new push requires a fresh approval.
const ApprovedProductionLabel = "prod/approved"

// KargoBot is the GitHub login of the Kargo automation bot (App ID 286661375).
// Renaming the App requires updating this constant.
const KargoBot = "konflux-kargo-bot[bot]"

// IsKargoBot reports whether login is the Kargo GitHub App bot.
func IsKargoBot(login string) bool {
	return login == KargoBot
}

// IssuesService is the subset of the GitHub Issues API used by this package.
type IssuesService interface {
	ListLabelsByIssue(ctx context.Context, owner, repo string, number int, opts *gh.ListOptions) ([]*gh.Label, *gh.Response, error)
	RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*gh.Response, error)
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*gh.Label, *gh.Response, error)
	GetLabel(ctx context.Context, owner, repo, name string) (*gh.Label, *gh.Response, error)
	CreateLabel(ctx context.Context, owner, repo string, label *gh.Label) (*gh.Label, *gh.Response, error)
}

// Client wraps the GitHub APIs used by this tool.
type Client struct {
	issues       IssuesService
	pullRequests PullRequestsService
	owner        string
	repo         string
}

// NewClient creates a new GitHub client from a token and "owner/repo" string.
func NewClient(token, repoFullName string) (*Client, error) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/repo", repoFullName)
	}
	client := gh.NewClient(nil).WithAuthToken(token)
	return &Client{
		issues:       client.Issues,
		pullRequests: client.PullRequests,
		owner:        parts[0],
		repo:         parts[1],
	}, nil
}

// SyncLabels ensures the PR has exactly the given labels (for managed prefixes)
// and removes any stale managed labels.
func (c *Client) SyncLabels(ctx context.Context, prNumber int, desiredLabels []string) error {
	// Get current labels on the PR. The list endpoint is paginated: a PR with
	// more than one page of labels must not lose managed labels beyond page 1,
	// otherwise they are never removed.
	currentLabels, err := c.listAllLabels(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("listing labels for PR #%d: %w", prNumber, err)
	}

	// Build sets of current managed labels and desired labels
	currentManaged := make(map[string]bool)
	for _, label := range currentLabels {
		name := label.GetName()
		if isManagedLabel(name) {
			currentManaged[name] = true
		}
	}

	desiredSet := make(map[string]bool, len(desiredLabels))
	for _, l := range desiredLabels {
		desiredSet[l] = true
	}

	// Remove labels that are no longer needed
	for label := range currentManaged {
		if !desiredSet[label] {
			slog.Info("Removing label", "label", label)
			_, err := c.issues.RemoveLabelForIssue(ctx, c.owner, c.repo, prNumber, label)
			if err != nil {
				return fmt.Errorf("removing label %q from PR #%d: %w", label, prNumber, err)
			}
		}
	}

	// Add labels that are missing
	var toAdd []string
	for label := range desiredSet {
		if !currentManaged[label] {
			toAdd = append(toAdd, label)
		}
	}
	if len(toAdd) > 0 {
		slog.Info("Adding labels", "labels", toAdd)

		// Ensure labels exist in the repo
		for _, label := range toAdd {
			if err := c.ensureLabelExists(ctx, label); err != nil {
				return err
			}
		}

		_, _, err := c.issues.AddLabelsToIssue(ctx, c.owner, c.repo, prNumber, toAdd)
		if err != nil {
			return fmt.Errorf("adding labels %v to PR #%d: %w", toAdd, prNumber, err)
		}
	}

	return nil
}

// listAllLabels returns every label on the PR, following pagination.
func (c *Client) listAllLabels(ctx context.Context, prNumber int) ([]*gh.Label, error) {
	var all []*gh.Label
	opts := &gh.ListOptions{PerPage: 100}
	for {
		page, resp, err := c.issues.ListLabelsByIssue(ctx, c.owner, c.repo, prNumber, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if resp == nil || resp.NextPage == 0 {
			return all, nil
		}
		opts.Page = resp.NextPage
	}
}

// ensureLabelExists creates the label in the repo if it doesn't exist.
func (c *Client) ensureLabelExists(ctx context.Context, label string) error {
	_, resp, err := c.issues.GetLabel(ctx, c.owner, c.repo, label)
	if err == nil {
		return nil
	}
	if resp != nil && resp.StatusCode == 404 {
		color := labelColor(label)
		_, _, err := c.issues.CreateLabel(ctx, c.owner, c.repo, &gh.Label{
			Name:  gh.Ptr(label),
			Color: gh.Ptr(color),
		})
		if err != nil {
			return fmt.Errorf("creating label %q: %w", label, err)
		}
		return nil
	}
	return fmt.Errorf("checking label %q: %w", label, err)
}

// labelColor returns a hex color for the label based on its prefix.
func labelColor(label string) string {
	switch {
	case strings.HasPrefix(label, "environment/production"):
		return "d73a4a" // red
	case strings.HasPrefix(label, "environment/staging"):
		return "fbca04" // yellow
	case strings.HasPrefix(label, "environment/development"):
		return "0e8a16" // green
	case strings.HasPrefix(label, "environment/none"):
		return "c5def5" // light blue
	case label == HoldProductionLabel:
		return "e11d48" // bright red — blocks merge
	case label == UnreferencedChangesLabel:
		return "f9a825" // orange — changes nothing deploys
	case strings.HasPrefix(label, "cluster/"):
		return "1d76db" // blue
	default:
		return "ededed" // grey
	}
}

// isManagedLabel checks if a label is managed by this tool.
func isManagedLabel(name string) bool {
	for _, prefix := range LabelPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
