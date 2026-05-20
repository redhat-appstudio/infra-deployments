// Command changelog-generator generates a human-readable changelog for a PR
// that bumps the Konflux operator SHA in infra-deployments. It posts (or
// updates) a PR comment containing operator-level changes and upstream service
// version bumps.
//
// Local dry-run (no git worktree, prints markdown to stdout):
//
//	changelog-generator \
//	  --old-sha=<40-char-sha> \
//	  --new-sha=<40-char-sha> \
//	  --token=$GITHUB_TOKEN \
//	  --dry-run
//
// CI (reads PR_NUMBER / GITHUB_TOKEN / GITHUB_REPOSITORY from env):
//
//	changelog-generator --base-ref=${{ github.event.pull_request.base.sha }}
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
)

// kustomizationPath is the path, relative to the repo root, of the
// kustomization.yaml file that pins the operator SHA.
const kustomizationPath = "components/konflux-operator/development/invariant/kustomization.yaml"

// version is set via -ldflags at build time.
var version = "dev"

func main() {
	var (
		repoRoot    = flag.String("repo-root", "", "Path to infra-deployments root (default: auto-detect via git)")
		baseRef     = flag.String("base-ref", "", "Base git ref for worktree comparison (default: merge-base with main)")
		oldSHA      = flag.String("old-sha", "", "Override old operator SHA — skips git worktree, useful for local testing")
		newSHA      = flag.String("new-sha", "", "Override new operator SHA — skips git worktree, useful for local testing")
		token       = flag.String("token", "", "GitHub API token (default: $GITHUB_TOKEN)")
		prStr       = flag.String("pr", "", "PR number to comment on (default: $PR_NUMBER)")
		repo        = flag.String("repo", "", "GitHub repo in owner/name format (default: $GITHUB_REPOSITORY)")
		dryRun      = flag.Bool("dry-run", false, "Print generated markdown to stdout instead of posting a PR comment")
		showVersion = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("changelog-generator %s\n", version)
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Resolve credentials and CI identifiers from flags or environment.
	if *token == "" {
		*token = os.Getenv("GITHUB_TOKEN")
	}
	if *repo == "" {
		*repo = os.Getenv("GITHUB_REPOSITORY")
	}
	if *prStr == "" {
		*prStr = os.Getenv("PR_NUMBER")
	}
	runURL := buildRunURL(
		os.Getenv("GITHUB_SERVER_URL"),
		os.Getenv("GITHUB_REPOSITORY"),
		os.Getenv("GITHUB_RUN_ID"),
	)

	// Step 1: Determine old and new operator SHAs.
	effectiveOldSHA, effectiveNewSHA := *oldSHA, *newSHA

	if effectiveOldSHA == "" || effectiveNewSHA == "" {
		// No SHA overrides — read them from the kustomization file at base and HEAD.
		absRoot, err := resolveRepoRoot(ctx, *repoRoot)
		if err != nil {
			slog.Error("resolving repo root", "err", err)
			postOrPrint(ctx, *dryRun, *token, *repo, *prStr, changelog.FormatError(err, runURL))
			os.Exit(0)
		}

		effectiveBaseRef := *baseRef
		if effectiveBaseRef == "" {
			effectiveBaseRef, err = git.MergeBase(ctx, absRoot, "main")
			if err != nil {
				slog.Error("computing merge-base; pass --base-ref to override", "err", err)
				postOrPrint(ctx, *dryRun, *token, *repo, *prStr, changelog.FormatError(err, runURL))
				os.Exit(0)
			}
		}

		worktreePath, cleanup, err := git.CreateWorktree(ctx, absRoot, effectiveBaseRef)
		if err != nil {
			slog.Error("creating worktree", "err", err)
			postOrPrint(ctx, *dryRun, *token, *repo, *prStr, changelog.FormatError(err, runURL))
			os.Exit(0)
		}
		defer cleanup()

		basePath := filepath.Join(worktreePath, kustomizationPath)
		headPath := filepath.Join(absRoot, kustomizationPath)

		effectiveOldSHA, effectiveNewSHA, err = changelog.ExtractSHAs(basePath, headPath)
		if err != nil {
			slog.Error("extracting operator SHAs", "err", err)
			postOrPrint(ctx, *dryRun, *token, *repo, *prStr, changelog.FormatError(err, runURL))
			os.Exit(0)
		}
	}

	// Step 2: If SHA is unchanged, post a minimal "no change" comment and exit.
	if effectiveOldSHA == effectiveNewSHA {
		slog.Info("Operator SHA unchanged — no changelog needed")
		postOrPrint(ctx, *dryRun, *token, *repo, *prStr, changelog.FormatNoChange())
		return
	}
	slog.Info("Operator SHA bumped", "old", effectiveOldSHA[:12], "new", effectiveNewSHA[:12])

	// Step 3: Fetch the operator repo comparison (commits + file diffs).
	comparer := changelog.NewRepoComparer(*token)
	compare, err := changelog.FetchOperatorCompare(ctx, comparer, effectiveOldSHA, effectiveNewSHA)
	if err != nil {
		slog.Error("fetching operator comparison", "err", err)
		postOrPrint(ctx, *dryRun, *token, *repo, *prStr, changelog.FormatError(err, runURL))
		return
	}

	// Step 4: Parse file diffs to find which upstream services were bumped.
	bumps := changelog.ExtractServiceBumps(compare.Files)
	slog.Info("Service bumps detected", "count", len(bumps))

	// Step 5: Fetch commits for each bumped service (degraded on error).
	var serviceChanges []changelog.ServiceChange
	for _, bump := range bumps {
		commits, err := changelog.FetchServiceCommits(ctx, comparer, bump.Owner, bump.Repo, bump.OldSHA, bump.NewSHA)
		if err != nil {
			slog.Warn("Failed to fetch service commits — including bump without commit list",
				"repo", bump.Repo, "err", err)
		}
		serviceChanges = append(serviceChanges, changelog.ServiceChange{
			Bump:    bump,
			Commits: changelog.AllNonMergeCommitsFilter(commits),
		})
	}

	// Step 6: Filter operator-level commits into notable (feat/fix) and remaining.
	operatorResult := changelog.FilterOperatorCommits(compare.Commits)
	slog.Info("Operator commits filtered",
		"notable", len(operatorResult.Notable),
		"remaining", len(operatorResult.Remaining))

	// Step 7: Format and post/print.
	body := changelog.Format(changelog.ChangelogData{
		OldSHA:         effectiveOldSHA,
		NewSHA:         effectiveNewSHA,
		OperatorResult: operatorResult,
		ServiceChanges: serviceChanges,
	})
	postOrPrint(ctx, *dryRun, *token, *repo, *prStr, body)
}

// resolveRepoRoot returns the absolute path to the repository root,
// auto-detecting via git if repoRoot is empty.
func resolveRepoRoot(ctx context.Context, repoRoot string) (string, error) {
	if repoRoot == "" {
		detected, err := git.TopLevel(ctx)
		if err != nil {
			return "", fmt.Errorf("auto-detecting repo root (use --repo-root to override): %w", err)
		}
		repoRoot = detected
	}
	return filepath.Abs(repoRoot)
}

// postOrPrint either posts body as a PR comment or prints it to stdout.
// Falls back to stdout if any required CI identifier is missing.
func postOrPrint(ctx context.Context, dryRun bool, token, repo, prStr, body string) {
	if dryRun || token == "" || repo == "" || prStr == "" {
		fmt.Print(body)
		return
	}

	prNumber := 0
	if _, err := fmt.Sscanf(prStr, "%d", &prNumber); err != nil || prNumber == 0 {
		slog.Error("invalid PR number — printing to stdout", "pr", prStr)
		fmt.Print(body)
		return
	}

	client, err := ghclient.NewCommentClient(token, repo)
	if err != nil {
		slog.Error("creating GitHub client — printing to stdout", "err", err)
		fmt.Print(body)
		return
	}

	if err := client.UpsertCommentByMarker(ctx, prNumber, body, changelog.CommentMarker); err != nil {
		slog.Error("posting PR comment — printing to stdout as fallback", "err", err)
		fmt.Print(body)
	}
}

// buildRunURL constructs a direct link to the GitHub Actions workflow run.
// Returns empty string if any component is missing.
func buildRunURL(serverURL, repo, runID string) string {
	if serverURL == "" || repo == "" || runID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
}
