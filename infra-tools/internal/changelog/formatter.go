package changelog

import (
	"fmt"
	"strings"
)

// CommentMarker is the HTML comment used to identify changelog PR comments
// for idempotent updates. When the tool runs again on a force-push it finds
// this marker and updates the existing comment rather than creating a new one.
const CommentMarker = "<!-- changelog-generator-comment -->"

// ChangelogData holds all the processed data needed to render a changelog.
type ChangelogData struct {
	// OldSHA and NewSHA are the operator commit SHAs being compared.
	OldSHA string
	NewSHA string
	// OperatorResult is the filtered set of operator repo commits.
	OperatorResult FilterResult
	// ServiceChanges maps each bumped service to its filtered commits.
	ServiceChanges []ServiceChange
}

// ServiceChange pairs a ServiceBump with the filtered commits for that service.
type ServiceChange struct {
	Bump    ServiceBump
	Commits []CommitInfo
}

// Format renders the changelog as a markdown string suitable for a GitHub PR
// comment. The CommentMarker is always the first line so UpsertComment can
// find and update it.
func Format(data ChangelogData) string {
	var b strings.Builder

	fmt.Fprintln(&b, CommentMarker)
	fmt.Fprintln(&b, "### Operator Changelog")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Comparing [`%s`](https://github.com/konflux-ci/konflux-ci/commit/%s) → [`%s`](https://github.com/konflux-ci/konflux-ci/commit/%s)\n",
		data.OldSHA[:12], data.OldSHA, data.NewSHA[:12], data.NewSHA)
	fmt.Fprintln(&b)

	writeServiceChanges(&b, data.ServiceChanges)
	writeOperatorChanges(&b, data.OperatorResult, data.OldSHA, data.NewSHA)

	return b.String()
}

// FormatNoChange renders a minimal comment when the operator SHA did not
// change in the PR — so reviewers can see the tool ran and found nothing.
func FormatNoChange() string {
	var b strings.Builder
	fmt.Fprintln(&b, CommentMarker)
	fmt.Fprintln(&b, "### Operator Changelog")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "No operator SHA change detected in this PR.")
	return b.String()
}

// FormatError renders a degraded comment when the changelog could not be
// fully generated due to an API error. runURL is a link to the workflow run
// for debugging; pass empty string if unavailable.
func FormatError(err error, runURL string) string {
	var b strings.Builder
	fmt.Fprintln(&b, CommentMarker)
	fmt.Fprintln(&b, "### Operator Changelog")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "⚠️ Changelog generation failed. Please check the changes manually.")
	if runURL != "" {
		fmt.Fprintf(&b, "\n[View workflow run](%s) for details.\n", runURL)
	}
	fmt.Fprintf(&b, "\n```\n%v\n```\n", err)
	return b.String()
}

// writeServiceChanges renders the "Underlying Service Changes" section.
func writeServiceChanges(b *strings.Builder, changes []ServiceChange) {
	if len(changes) == 0 {
		return
	}

	fmt.Fprintln(b, "#### Underlying Service Changes")
	fmt.Fprintln(b)

	for _, sc := range changes {
		compareURL := fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s",
			sc.Bump.Owner, sc.Bump.Repo, sc.Bump.OldSHA, sc.Bump.NewSHA)

		fmt.Fprintf(b, "**%s** — [`%s`](%s) → [`%s`](%s) ([compare](%s))\n",
			sc.Bump.Repo,
			sc.Bump.OldSHA[:12],
			fmt.Sprintf("https://github.com/%s/%s/commit/%s", sc.Bump.Owner, sc.Bump.Repo, sc.Bump.OldSHA),
			sc.Bump.NewSHA[:12],
			fmt.Sprintf("https://github.com/%s/%s/commit/%s", sc.Bump.Owner, sc.Bump.Repo, sc.Bump.NewSHA),
			compareURL,
		)

		if len(sc.Commits) == 0 {
			fmt.Fprintln(b, "_No notable commits found._")
		} else {
			for _, c := range sc.Commits {
				fmt.Fprintf(b, "- [`%s`](%s) %s\n", c.SHA[:8], c.HTMLURL, c.Subject())
			}
		}
		fmt.Fprintln(b)
	}
}

// writeOperatorChanges renders the "Operator Changes" section with two tiers:
// notable (feat/fix) shown directly, remaining commits in a collapsible block.
func writeOperatorChanges(b *strings.Builder, result FilterResult, oldSHA, newSHA string) {
	compareURL := fmt.Sprintf("https://github.com/konflux-ci/konflux-ci/compare/%s...%s", oldSHA, newSHA)

	if len(result.Notable) == 0 && len(result.Remaining) == 0 {
		fmt.Fprintln(b, "#### Operator Changes")
		fmt.Fprintln(b)
		fmt.Fprintf(b, "_No operator commits found. [View full comparison](%s)_\n", compareURL)
		return
	}

	fmt.Fprintln(b, "#### Operator Changes")
	fmt.Fprintln(b)

	if len(result.Notable) > 0 {
		for _, c := range result.Notable {
			fmt.Fprintf(b, "- [`%s`](%s) %s\n", c.SHA[:8], c.HTMLURL, c.Subject())
		}
		fmt.Fprintln(b)
	}

	if len(result.Remaining) > 0 {
		summary := fmt.Sprintf("Other commits (%d)", len(result.Remaining))
		fmt.Fprintf(b, "<details>\n<summary>%s</summary>\n\n", summary)
		for _, c := range result.Remaining {
			fmt.Fprintf(b, "- [`%s`](%s) %s\n", c.SHA[:8], c.HTMLURL, c.Subject())
		}
		fmt.Fprintf(b, "\n[View full comparison](%s)\n", compareURL)
		fmt.Fprintln(b, "</details>")
	} else {
		fmt.Fprintf(b, "[View full comparison](%s)\n", compareURL)
	}
}
