// Package changelog provides logic for generating a human-readable changelog
// when the Konflux operator ref is bumped in infra-deployments.
package changelog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	gh "github.com/google/go-github/v68/github"
)

// RepoComparer is the GitHub compare API surface used to fetch file diffs
// between two refs in a repository. Defined as an interface so tests can
// inject a fake without making real network calls.
type RepoComparer interface {
	CompareCommits(ctx context.Context, owner, repo, base, head string, opts *gh.ListOptions) (*gh.CommitsComparison, *gh.Response, error)
}

// FileChange holds the fields we need from a file changed in a comparison.
type FileChange struct {
	Filename string
	Patch    string // unified diff; may be empty for binary or very large files
}

// OperatorCompare holds the changed files and conventional commits from comparing
// two operator refs. Commits are filtered to feat/fix only (same rules as
// FetchServiceCommits) so callers can render an operator changelog without a
// second Compare API call.
type OperatorCompare struct {
	Files            []FileChange
	Truncated        bool // true when the API returned the 300-file maximum, indicating possible truncation
	Commits          []ConventionalCommit
	CommitsTruncated bool // true when AheadBy hit the 250-commit API limit
}

// maxCompareFiles is the GitHub compare API hard limit on returned files.
// When the response hits this limit the result may be incomplete; callers
// should treat Truncated == true the same way they treat an API failure.
const maxCompareFiles = 300

// NewRepoComparer creates a RepoComparer backed by the real GitHub REST API.
// Pass an empty token to use unauthenticated access (60 req/hour limit).
// In CI, always pass the GITHUB_TOKEN.
func NewRepoComparer(token string) RepoComparer {
	client := gh.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	return client.Repositories
}

// FetchOperatorCompare calls the GitHub compare API for konflux-ci/konflux-ci
// between oldRef and newRef and returns the changed files plus filtered
// conventional commits. The call is retried up to three times with exponential
// backoff to survive transient failures.
//
// When the API returns exactly maxCompareFiles files the result is marked
// Truncated; callers should degrade bump detection in the same way they handle
// API errors. Commits are still returned when Truncated is true — file truncation
// is independent of the commit list.
func FetchOperatorCompare(ctx context.Context, c RepoComparer, oldRef, newRef string) (*OperatorCompare, error) {
	var comparison *gh.CommitsComparison
	err := retryDo(ctx, 3, func() error {
		var e error
		comparison, _, e = c.CompareCommits(ctx, "konflux-ci", "konflux-ci", oldRef, newRef, nil)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("comparing %s...%s in konflux-ci/konflux-ci: %w",
			refShort(oldRef), refShort(newRef), err)
	}
	files := convertFiles(comparison.Files)
	return &OperatorCompare{
		Files:            files,
		Truncated:        len(files) >= maxCompareFiles,
		Commits:          filterConventional(comparison.Commits),
		CommitsTruncated: comparison.GetAheadBy() >= CommitMaxFromCompare,
	}, nil
}

// retryDo calls fn up to maxAttempts times. Between attempts it sleeps for
// 2^attempt seconds (1 s, 2 s, …), honouring context cancellation.
// Only retryable errors (network failures, 429, 5xx) are retried — permanent
// errors like 401 or 404 fail immediately without waiting.
func retryDo(ctx context.Context, maxAttempts int, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == maxAttempts-1 || !isRetryable(err) {
			break
		}
		wait := time.Duration(1<<attempt) * time.Second // 1 s, 2 s, …
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}

// isRetryable returns true for errors worth retrying: network-level errors and
// GitHub HTTP errors with status 429 (rate limited) or 5xx (server error).
// Permanent errors such as 401 Unauthorized or 404 Not Found are not retried.
func isRetryable(err error) bool {
	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) {
		code := ghErr.Response.StatusCode
		return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
	}
	return true // network / timeout errors are always retryable
}

func convertFiles(raw []*gh.CommitFile) []FileChange {
	out := make([]FileChange, 0, len(raw))
	for _, f := range raw {
		if f == nil {
			continue
		}
		out = append(out, FileChange{
			Filename: f.GetFilename(),
			Patch:    f.GetPatch(),
		})
	}
	return out
}

// refShort returns the first 12 characters of ref, or the full string if shorter.
func refShort(ref string) string {
	if len(ref) > 12 {
		return ref[:12]
	}
	return ref
}
