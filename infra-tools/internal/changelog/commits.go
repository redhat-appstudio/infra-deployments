package changelog

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

// conventionalPattern matches conventional commit subject lines.
// Captures: type (feat|fix), optional (scope), optional !, and the message.
var conventionalPattern = regexp.MustCompile(`^(feat|fix)(\([^)]+\))?!?: (.+)`)

// CommitMaxFromCompare is the GitHub compare API hard limit on returned commits.
// When AheadBy hits this value the result may be incomplete; callers should
// note truncation in the output rather than silently omitting commits.
const CommitMaxFromCompare = 250

// ConventionalCommit holds a parsed conventional commit entry.
type ConventionalCommit struct {
	Type    string // "feat" or "fix"
	Scope   string // optional scope, e.g. "pipeline"; empty if not present
	Subject string // message after the type/scope prefix
	SHA     string // 12-char short SHA for traceability
}

// FetchServiceCommits calls the GitHub compare API for bump.Owner/bump.Repo
// between bump.OldSHA and bump.NewSHA and returns filtered conventional commits.
//
// Reuses the RepoComparer interface so the same retry and auth logic from
// FetchOperatorCompare applies here — no separate client or interface needed.
// The second return value is true when AheadBy hit the 250-commit API limit,
// indicating results may be incomplete.
func FetchServiceCommits(ctx context.Context, c RepoComparer, bump ServiceBump) ([]ConventionalCommit, bool, error) {
	var comparison *gh.CommitsComparison
	err := retryDo(ctx, 3, func() error {
		var e error
		comparison, _, e = c.CompareCommits(ctx, bump.Owner, bump.Repo, bump.OldSHA, bump.NewSHA, nil)
		return e
	})
	if err != nil {
		return nil, false, fmt.Errorf("comparing %s/%s %s...%s: %w",
			bump.Owner, bump.Repo, refShort(bump.OldSHA), refShort(bump.NewSHA), err)
	}
	commits := filterConventional(comparison.Commits)
	truncated := comparison.GetAheadBy() >= CommitMaxFromCompare
	return commits, truncated, nil
}

// filterConventional returns only feat/fix conventional commits, taking only
// the first line of each commit message to avoid multi-paragraph bodies.
func filterConventional(raw []*gh.RepositoryCommit) []ConventionalCommit {
	var out []ConventionalCommit
	for _, c := range raw {
		if c == nil || c.Commit == nil || c.Commit.Message == nil {
			continue
		}
		subject := firstLine(*c.Commit.Message)
		typ, scope, msg, ok := parseConventionalCommit(subject)
		if !ok {
			continue
		}
		out = append(out, ConventionalCommit{
			Type:    typ,
			Scope:   scope,
			Subject: msg,
			SHA:     refShort(c.GetSHA()),
		})
	}
	return out
}

// parseConventionalCommit parses a conventional commit subject line of the form
// "feat: msg", "fix(scope): msg", "feat!: msg", or "fix(scope)!: msg".
// Returns (type, scope, message, true) on a match, or ("","","",false) otherwise.
func parseConventionalCommit(subject string) (typ, scope, msg string, ok bool) {
	m := conventionalPattern.FindStringSubmatch(subject)
	if m == nil {
		return "", "", "", false
	}
	// m[1]=type, m[2]="(scope)" with parens (empty string if absent), m[3]=message
	return m[1], strings.Trim(m[2], "()"), m[3], true
}

// firstLine returns the first line of a potentially multi-line string.
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
