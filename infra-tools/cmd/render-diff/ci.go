package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

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

	if len(result.Diffs) == 0 {
		_, _ = fmt.Fprintln(w, "No render differences detected.")
		return w.Flush()
	}

	sortDiffs(result.Diffs)

	_, _ = fmt.Fprintln(w, "# Kustomize Render Diff")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "**%d components** with differences (+%d -%d lines)\n\n", len(result.Diffs), result.TotalAdded, result.TotalRemoved)

	const truncateThreshold = 50 * 1024 // 50KB
	for _, d := range result.Diffs {
		if d.Error != "" {
			summary := fmt.Sprintf("%s (%s) — build error", d.Path, d.Env)
			_, _ = fmt.Fprintf(w, "<details>\n<summary>%s</summary>\n\n", summary)
			_, _ = fmt.Fprintf(w, "```\n%s\n```\n\n", d.Error)
			_, _ = fmt.Fprintln(w, "</details>")
			_, _ = fmt.Fprintln(w)
			continue
		}
		summary := fmt.Sprintf("%s (%s) — +%d -%d", d.Path, d.Env, d.Added, d.Removed)
		_, _ = fmt.Fprintf(w, "<details>\n<summary>%s</summary>\n\n", summary)
		if len(d.Diff) > truncateThreshold {
			_, _ = fmt.Fprintf(w, "```diff\n%s\n```\n\n", d.Diff[:truncateThreshold])
			_, _ = fmt.Fprintln(w, "⚠️ Diff truncated. Download the full artifact for the complete diff.")
		} else {
			_, _ = fmt.Fprintf(w, "```diff\n%s\n```\n\n", d.Diff)
		}
		_, _ = fmt.Fprintln(w, "</details>")
		_, _ = fmt.Fprintln(w)
	}
	return w.Flush()
}

// postCIComment generates the PR comment markdown and posts it to GitHub.
// CI-specific configuration is read from environment variables:
//   - GITHUB_TOKEN: API token for authentication
//   - GITHUB_REPOSITORY: repository in "owner/repo" format
//   - PR_NUMBER: pull request number to comment on
//
// If any of these are missing, the comment body is printed to stdout instead.
func postCIComment(ctx context.Context, result *renderdiff.DiffResult, headSHA, baseSHA string) error {
	body := buildCommentBody(result, headSHA, baseSHA)

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

// buildCommentBody generates the markdown for a PR comment.
func buildCommentBody(result *renderdiff.DiffResult, headSHA, baseSHA string) string {
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
	for _, d := range result.Diffs {
		if d.Error != "" {
			fmt.Fprintf(&b, "| `%s` | %s | build error |\n", d.Path, d.Env)
		} else {
			fmt.Fprintf(&b, "| `%s` | %s | +%d -%d |\n", d.Path, d.Env, d.Added, d.Removed)
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "**Total:** %d components, +%d -%d lines\n\n", len(result.Diffs), result.TotalAdded, result.TotalRemoved)
	fmt.Fprintln(&b, "📋 Full diff available in the [workflow summary](../actions) and as a downloadable artifact.")
	return b.String()
}
