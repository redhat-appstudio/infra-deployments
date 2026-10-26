package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// runOutputMode executes a single output mode against a pre-computed result.
// Returns an error instead of calling Fatal, so the caller can continue with
// remaining modes.
func runOutputMode(ctx context.Context, mode OutputMode, result *renderdiff.DiffResult, colorMode string, openDiff bool, outputDir, headSHA, baseSHA string) error {
	switch mode {
	case OutputModeLocal:
		useColor := shouldUseColor(colorMode)
		if outputDir != "" {
			if err := writeDiffFiles(result, outputDir); err != nil {
				return fmt.Errorf("writing diff files: %w", err)
			}
		}
		if openDiff {
			if err := openInDiffTool(result); err != nil {
				return fmt.Errorf("opening diff tool: %w", err)
			}
			return nil
		}
		for _, cd := range result.Diffs {
			printComponentDiff(cd, useColor)
		}
		printSummary(result)
	case OutputModeCISummary:
		if err := writeCISummary(result); err != nil {
			return err
		}
	case OutputModeCIComment:
		if err := postCIComment(ctx, result, headSHA, baseSHA); err != nil {
			return err
		}
	case OutputModeCIArtifact:
		if outputDir == "" {
			return fmt.Errorf("--output-dir is required for ci-artifact-dir mode")
		}
		if err := writeDiffFiles(result, outputDir); err != nil {
			return fmt.Errorf("writing artifact diff files: %w", err)
		}
		fmt.Printf("Wrote %d diff files to %s\n", len(result.Diffs), outputDir)
	}
	return nil
}

// writeCISummary generates markdown for $GITHUB_STEP_SUMMARY.
// When the GITHUB_STEP_SUMMARY environment variable is set, output is written
// directly to that file so it doesn't mix with other modes' stdout output.
// Falls back to stdout when the variable is unset.
func writeCISummary(result *renderdiff.DiffResult) error {
	var dest io.Writer = os.Stdout
	if summaryPath := os.Getenv("GITHUB_STEP_SUMMARY"); summaryPath != "" {
		f, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("opening $GITHUB_STEP_SUMMARY: %w", err)
		}
		defer func() { _ = f.Close() }()
		dest = f
	}

	w := bufio.NewWriter(dest)
	if _, err := io.WriteString(w, renderCISummary(result)); err != nil {
		return err
	}
	return w.Flush()
}

// GitHub truncates or rejects oversized output: the step summary is capped at
// 1 MiB and a comment body at 65,536 characters. Stay comfortably below both.
const (
	summarySizeBudget = 900 * 1024
	commentSizeBudget = 60_000
)

// perDiffSizeBudget is how much of a single diff is inlined in the summary.
const perDiffSizeBudget = 50 * 1024

// renderCISummary renders the markdown for $GITHUB_STEP_SUMMARY, dropping
// component details once the size budget is reached so GitHub never truncates
// the output mid-document.
func renderCISummary(result *renderdiff.DiffResult) string {
	if len(result.Diffs) == 0 {
		return "No render differences detected.\n"
	}

	sortDiffs(result.Diffs)

	shown := 0
	for _, d := range result.Diffs {
		if !d.SkipOutput {
			shown++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Kustomize Render Diff\n\n")
	fmt.Fprintf(&b, "**%d components** with differences (+%d -%d lines)\n\n", shown, result.TotalAdded, result.TotalRemoved)

	omitted := 0
	for _, d := range result.Diffs {
		if d.SkipOutput {
			continue
		}
		block := renderSummaryBlock(d)
		if b.Len()+len(block) > summarySizeBudget {
			omitted++
			continue
		}
		b.WriteString(block)
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "⚠️ %d components omitted: the summary would exceed the GitHub size limit. Download the artifact for the complete diff.\n", omitted)
	}
	return b.String()
}

// renderSummaryBlock renders one collapsible component section.
func renderSummaryBlock(d renderdiff.ComponentDiff) string {
	var b strings.Builder

	if d.Error != "" {
		fmt.Fprintf(&b, "<details>\n<summary>%s (%s) — build error</summary>\n\n", d.Path, d.Env)
		fmt.Fprintf(&b, "```\n%s\n```\n\n", d.Error)
		fmt.Fprintln(&b, "</details>")
		fmt.Fprintln(&b)
		return b.String()
	}

	fmt.Fprintf(&b, "<details>\n<summary>%s (%s) — +%d -%d</summary>\n\n", d.Path, d.Env, d.Added, d.Removed)
	if len(d.Diff) > perDiffSizeBudget {
		b.WriteString("```diff\n")
		b.WriteString(truncateAtLine(d.Diff, perDiffSizeBudget))
		b.WriteString("\n```\n\n")
		fmt.Fprintln(&b, "⚠️ Diff truncated. Download the full artifact for the complete diff.")
	} else {
		fmt.Fprintf(&b, "```diff\n%s\n```\n\n", d.Diff)
	}
	fmt.Fprintln(&b, "</details>")
	fmt.Fprintln(&b)
	return b.String()
}

// truncateAtLine cuts text at the last newline before limit so the result never
// ends mid-line or mid-rune (diffs are line-oriented, and a half UTF-8 sequence
// would corrupt the markdown).
func truncateAtLine(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := text[:limit]
	if i := strings.LastIndexByte(cut, '\n'); i > 0 {
		return cut[:i]
	}
	// No newline in range: fall back to a rune-safe cut.
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}

// postCIComment generates the PR comment markdown and posts it to GitHub.
// CI-specific configuration is read from environment variables:
//   - GITHUB_TOKEN: API token for authentication
//   - GITHUB_REPOSITORY: repository in "owner/repo" format
//   - PR_NUMBER: pull request number to comment on
//   - GITHUB_SERVER_URL: GitHub server URL (e.g., https://github.com)
//   - GITHUB_RUN_ID: current workflow run ID
//
// If any of the required vars (token, repo, PR) are missing, the comment body
// is printed to stdout instead.
func postCIComment(ctx context.Context, result *renderdiff.DiffResult, headSHA, baseSHA string) error {
	runURL := buildRunURL(
		os.Getenv("GITHUB_SERVER_URL"),
		os.Getenv("GITHUB_REPOSITORY"),
		os.Getenv("GITHUB_RUN_ID"),
	)
	body := buildCommentBody(result, headSHA, baseSHA, runURL)

	token := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")
	prStr := os.Getenv("PR_NUMBER")

	if token == "" || repo == "" || prStr == "" {
		// Missing CI env vars — print to stdout as fallback.
		fmt.Print(body)
		return nil
	}

	prNumber := 0
	if _, err := fmt.Sscanf(prStr, "%d", &prNumber); err != nil || prNumber == 0 {
		return fmt.Errorf("invalid PR_NUMBER %q", prStr)
	}

	client, err := ghclient.NewCommentClient(token, repo)
	if err != nil {
		return fmt.Errorf("creating GitHub client: %w", err)
	}
	if err := client.UpsertComment(ctx, prNumber, body); err != nil {
		return fmt.Errorf("posting PR comment: %w", err)
	}
	slog.Info("PR comment posted", "pr", prNumber)
	return nil
}

// buildRunURL constructs a direct link to the GitHub Actions workflow run.
// Returns empty string if any component is missing.
func buildRunURL(serverURL, repo, runID string) string {
	if serverURL == "" || repo == "" || runID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
}

// buildCommentBody generates the markdown for a PR comment.
// When runURL is non-empty, the workflow summary link points directly to the
// specific run; otherwise it falls back to the relative ../actions link.
func buildCommentBody(result *renderdiff.DiffResult, headSHA, baseSHA, runURL string) string {
	var b strings.Builder

	fmt.Fprintln(&b, "<!-- render-diff-comment -->")
	fmt.Fprintln(&b, "### Kustomize Render Diff")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Comparing `%s` → `%s`\n\n", baseSHA, headSHA)

	if len(result.Diffs) == 0 {
		fmt.Fprintln(&b, "No render differences detected.")
		return b.String()
	}

	sortDiffs(result.Diffs)

	fmt.Fprintln(&b, "| Component | Environment | Changes |")
	fmt.Fprintln(&b, "|-----------|-------------|---------|")
	omitted := 0
	for _, d := range result.Diffs {
		if d.SkipOutput {
			continue
		}
		var row string
		if d.Error != "" {
			row = fmt.Sprintf("| `%s` | %s | build error |\n", d.Path, d.Env)
		} else {
			row = fmt.Sprintf("| `%s` | %s | +%d -%d |\n", d.Path, d.Env, d.Added, d.Removed)
		}
		// Keep the body under GitHub's comment limit: a rejected comment would
		// leave the PR with no render diff at all.
		if b.Len()+len(row) > commentSizeBudget {
			omitted++
			continue
		}
		b.WriteString(row)
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "| … | %d more components | see the workflow summary |\n", omitted)
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "**Total:** %d components, +%d -%d lines\n\n", len(result.Diffs), result.TotalAdded, result.TotalRemoved)
	link := "../actions"
	if runURL != "" {
		link = runURL
	}
	fmt.Fprintf(&b, "📋 Full diff available in the [workflow summary](%s) and as a downloadable artifact.\n", link)
	return b.String()
}
