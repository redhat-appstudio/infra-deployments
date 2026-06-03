// Command changelog-generator posts a changelog comment on infra-deployments PRs
// that bump the Konflux operator SHA.
//
// This is the initial placeholder (KFLUXVNGD-1022). It proves the workflow
// trigger and comment pipeline work end-to-end on real PRs before any
// changelog logic is added. SHA extraction, upstream comparison, and full
// formatting are introduced in subsequent PRs.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
)

// commentMarker identifies the changelog comment for idempotent updates.
// This string is permanent — changing it would orphan existing comments on
// open PRs. It moves to internal/changelog/formatter.go in a later PR;
// the string itself never changes.
const commentMarker = "<!-- changelog-generator-comment -->"

// commenter is the subset of the GitHub comment API used by this binary.
// Defined as an interface so tests can inject a fake without network calls.
type commenter interface {
	UpsertCommentByMarker(ctx context.Context, prNumber int, body, marker string) error
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Print comment to stdout instead of posting")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")
	prStr := os.Getenv("PR_NUMBER")

	body := commentMarker + "\n### Operator Changelog\n\n" +
		"_This comment will show operator commits and upstream service changes in upcoming PRs._\n"

	if *dryRun {
		fmt.Print(body)
		return
	}

	if token == "" || repo == "" || prStr == "" {
		slog.Error("missing required env vars", "GITHUB_TOKEN_set", token != "", "GITHUB_REPOSITORY_set", repo != "", "PR_NUMBER_set", prStr != "")
		os.Exit(1)
	}

	prNumber := 0
	if _, err := fmt.Sscanf(prStr, "%d", &prNumber); err != nil || prNumber == 0 {
		slog.Error("invalid PR number", "pr", prStr)
		os.Exit(1)
	}

	client, err := ghclient.NewCommentClient(token, repo)
	if err != nil {
		slog.Error("creating GitHub client", "err", err)
		os.Exit(1)
	}

	if err := post(ctx, client, prNumber, body); err != nil {
		slog.Error("posting PR comment", "err", err)
		os.Exit(1)
	}
}

// post delivers body as a PR comment using the provided commenter.
// Keeping the commenter as an injected interface allows tests to verify
// the correct body and PR number are passed without making real API calls.
func post(ctx context.Context, c commenter, prNumber int, body string) error {
	return c.UpsertCommentByMarker(ctx, prNumber, body, commentMarker)
}
