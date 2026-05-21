package changelog

import "strings"

// CommitFilter is a pluggable function that decides which commits from a
// repository are worth including in the changelog. It receives the full list
// of commits between two SHAs and returns the subset to display.
//
// The interface is intentionally simple so it can be swapped per-service
// without changing any other code. The conventional commits implementation
// below is the default.
type CommitFilter func(commits []CommitInfo) []CommitInfo

// ConventionalCommitsFilter is the default CommitFilter. It returns commits
// whose subject line starts with "feat" or "fix", excluding merge commits.
// This works well for repositories that follow the conventional commits spec.
//
// Sub-service repositories that do not follow conventional commits should use
// AllNonMergeCommitsFilter instead.
func ConventionalCommitsFilter(commits []CommitInfo) []CommitInfo {
	var out []CommitInfo
	for _, c := range commits {
		if c.IsMerge {
			continue
		}
		subject := strings.ToLower(c.Subject())
		if strings.HasPrefix(subject, "feat") || strings.HasPrefix(subject, "fix") {
			out = append(out, c)
		}
	}
	return out
}

// FilterResult holds the output of filtering commits for the operator repo.
// Operator commits are split into two tiers for display.
type FilterResult struct {
	// Notable contains feat: and fix: commits — shown prominently at the top.
	Notable []CommitInfo
	// Remaining contains every non-merge commit that is NOT already in Notable —
	// shown in a collapsible block so nothing is hidden but the view stays clean.
	Remaining []CommitInfo
}

// FilterOperatorCommits applies two-tier filtering to the operator repo commits.
// Notable (feat/fix) are surfaced prominently; every other non-merge commit goes
// into Remaining so reviewers can expand the block to see the full picture.
func FilterOperatorCommits(commits []CommitInfo) FilterResult {
	var notable, remaining []CommitInfo
	for _, c := range commits {
		if c.IsMerge {
			continue
		}
		subject := strings.ToLower(c.Subject())
		if strings.HasPrefix(subject, "feat") || strings.HasPrefix(subject, "fix") {
			notable = append(notable, c)
		} else {
			remaining = append(remaining, c)
		}
	}
	return FilterResult{Notable: notable, Remaining: remaining}
}
