package changelog

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

// RepoComparer is the subset of the GitHub Repositories API used by this
// package. Defined as an interface so tests can inject a fake implementation
// without making real network calls.
type RepoComparer interface {
	CompareCommits(ctx context.Context, owner, repo, base, head string, opts *gh.ListOptions) (*gh.CommitsComparison, *gh.Response, error)
}

// CommitInfo holds the fields we need from a single commit in a comparison.
type CommitInfo struct {
	SHA     string
	Message string // full commit message
	HTMLURL string
	IsMerge bool // true when the commit has two or more parents (merge commit)
}

// Subject returns the first line of the commit message.
func (c CommitInfo) Subject() string {
	return strings.SplitN(c.Message, "\n", 2)[0]
}

// FileChange holds the fields we need from a file changed in a comparison.
type FileChange struct {
	Filename string
	Patch    string // unified diff patch; may be empty for binary files
	Status   string // added, modified, removed, renamed
}

// OperatorCompare holds the raw data returned by the operator repo compare call.
// Both Commits and Files cover the full range between the two SHAs.
type OperatorCompare struct {
	Commits []CommitInfo
	Files   []FileChange
}

// NewRepoComparer creates a RepoComparer backed by the real GitHub API.
// Pass an empty token to use unauthenticated access (rate-limited to 60/hour).
// In CI, always pass the GITHUB_TOKEN.
func NewRepoComparer(token string) RepoComparer {
	client := gh.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	return client.Repositories
}

// FetchOperatorCompare calls the GitHub compare API for the operator repo
// (konflux-ci/konflux-ci) between oldSHA and newSHA, and returns the full
// commit list and changed file list.
func FetchOperatorCompare(ctx context.Context, c RepoComparer, oldSHA, newSHA string) (*OperatorCompare, error) {
	comparison, _, err := c.CompareCommits(ctx, "konflux-ci", "konflux-ci", oldSHA, newSHA, nil)
	if err != nil {
		return nil, fmt.Errorf("comparing %s...%s in konflux-ci/konflux-ci: %w", oldSHA[:12], newSHA[:12], err)
	}

	result := &OperatorCompare{
		Commits: convertCommits(comparison.Commits),
		Files:   convertFiles(comparison.Files),
	}
	return result, nil
}

// FetchServiceCommits calls the GitHub compare API for a sub-service repo
// (e.g. konflux-ci/build-service) and returns the commit list. Merge commits
// are included; callers should use filter.go to remove them if needed.
func FetchServiceCommits(ctx context.Context, c RepoComparer, owner, repo, oldSHA, newSHA string) ([]CommitInfo, error) {
	comparison, _, err := c.CompareCommits(ctx, owner, repo, oldSHA, newSHA, nil)
	if err != nil {
		return nil, fmt.Errorf("comparing %s...%s in %s/%s: %w", oldSHA[:12], newSHA[:12], owner, repo, err)
	}
	return convertCommits(comparison.Commits), nil
}

// convertCommits maps the go-github RepositoryCommit slice into our leaner type.
func convertCommits(raw []*gh.RepositoryCommit) []CommitInfo {
	out := make([]CommitInfo, 0, len(raw))
	for _, c := range raw {
		if c == nil || c.Commit == nil {
			continue
		}
		info := CommitInfo{
			SHA:     c.GetSHA(),
			Message: c.Commit.GetMessage(),
			HTMLURL: c.GetHTMLURL(),
			IsMerge: len(c.Parents) >= 2,
		}
		out = append(out, info)
	}
	return out
}

// convertFiles maps the go-github CommitFile slice into our leaner type.
func convertFiles(raw []*gh.CommitFile) []FileChange {
	out := make([]FileChange, 0, len(raw))
	for _, f := range raw {
		if f == nil {
			continue
		}
		out = append(out, FileChange{
			Filename: f.GetFilename(),
			Patch:    f.GetPatch(),
			Status:   f.GetStatus(),
		})
	}
	return out
}
