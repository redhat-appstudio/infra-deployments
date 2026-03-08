package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

// CommentMarker is the HTML comment used to identify render-diff PR comments
// for idempotent updates.
const CommentMarker = "<!-- render-diff-comment -->"

// IssueCommentsService is the subset of the GitHub Issues API used for comments.
type IssueCommentsService interface {
	ListComments(ctx context.Context, owner, repo string, number int, opts *gh.IssueListCommentsOptions) ([]*gh.IssueComment, *gh.Response, error)
	CreateComment(ctx context.Context, owner, repo string, number int, comment *gh.IssueComment) (*gh.IssueComment, *gh.Response, error)
	EditComment(ctx context.Context, owner, repo string, commentID int64, comment *gh.IssueComment) (*gh.IssueComment, *gh.Response, error)
}

// CommentClient wraps a GitHub Issues comment service.
type CommentClient struct {
	comments IssueCommentsService
	owner    string
	repo     string
}

// NewCommentClient creates a new comment client from a token and "owner/repo" string.
func NewCommentClient(token, repoFullName string) (*CommentClient, error) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/repo", repoFullName)
	}
	client := gh.NewClient(nil).WithAuthToken(token)
	return &CommentClient{
		comments: client.Issues,
		owner:    parts[0],
		repo:     parts[1],
	}, nil
}

// UpsertComment creates or updates a PR comment identified by CommentMarker.
// If a comment with the marker exists, it is updated; otherwise a new comment is created.
func (c *CommentClient) UpsertComment(ctx context.Context, prNumber int, body string) error {
	// Find existing comment
	existingID, err := c.findMarkedComment(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("finding existing comment: %w", err)
	}

	if existingID != 0 {
		slog.Info("Updating existing render-diff comment", "comment_id", existingID)
		_, _, err = c.comments.EditComment(ctx, c.owner, c.repo, existingID, &gh.IssueComment{
			Body: gh.Ptr(body),
		})
		if err != nil {
			return fmt.Errorf("updating comment %d: %w", existingID, err)
		}
		return nil
	}

	slog.Info("Creating new render-diff comment")
	_, _, err = c.comments.CreateComment(ctx, c.owner, c.repo, prNumber, &gh.IssueComment{
		Body: gh.Ptr(body),
	})
	if err != nil {
		return fmt.Errorf("creating comment: %w", err)
	}
	return nil
}

// findMarkedComment searches for a comment containing the CommentMarker.
func (c *CommentClient) findMarkedComment(ctx context.Context, prNumber int) (int64, error) {
	opts := &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	for {
		comments, resp, err := c.comments.ListComments(ctx, c.owner, c.repo, prNumber, opts)
		if err != nil {
			return 0, err
		}
		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, CommentMarker) {
				return comment.GetID(), nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return 0, nil
}
