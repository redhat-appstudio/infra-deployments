// Command changelog-generator posts a changelog comment on infra-deployments PRs
// that bump the Konflux operator ref.
//
// This is step 4 (KFLUXVNGD-1079): it reads the operator ref, lists upstream
// sub-services whose pinned SHA changed, and fetches per-service feat/fix
// conventional commits to display under each bump entry.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
)

// kustomizationPath is the path, relative to the repo root, of the
// kustomization.yaml file that pins the operator ref.
const kustomizationPath = "components/konflux-operator/development/invariant/kustomization.yaml"

// commentMarker identifies the changelog comment for idempotent updates.
// This string is permanent — changing it would orphan existing comments on open PRs.
const commentMarker = "<!-- changelog-generator-comment -->"

type commenter interface {
	UpsertCommentByMarker(ctx context.Context, prNumber int, body, marker string) error
}

func main() {
	repoRoot := flag.String("repo-root", "", "Path to infra-deployments root (default: auto-detect via git)")
	baseRef := flag.String("base-ref", "", "Base git ref to compare against (default: merge-base with main)")
	dryRun := flag.Bool("dry-run", false, "Print comment to stdout instead of posting")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("GITHUB_REPOSITORY")
	prStr := os.Getenv("PR_NUMBER")

	comparer := changelog.NewRepoComparer(token)

	body, err := buildBody(ctx, *repoRoot, *baseRef, comparer)
	if err != nil {
		slog.Error("building changelog body", "err", err)
		os.Exit(1)
	}

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

// buildBody resolves the repo root and base ref, creates a git worktree for the
// base ref, then delegates all comment logic to computeBody.
func buildBody(ctx context.Context, repoRoot, baseRef string, comparer changelog.RepoComparer) (string, error) {
	absRoot, err := resolveRepoRoot(ctx, repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolving repo root: %w", err)
	}

	effectiveBaseRef := baseRef
	if effectiveBaseRef == "" {
		effectiveBaseRef, err = git.MergeBase(ctx, absRoot, "origin/main")
		if err != nil {
			return "", fmt.Errorf("computing merge-base (pass --base-ref to override): %w", err)
		}
	}

	worktreePath, cleanup, err := git.CreateWorktree(ctx, absRoot, effectiveBaseRef)
	if err != nil {
		return "", fmt.Errorf("creating worktree: %w", err)
	}
	defer cleanup()

	basePath := filepath.Join(worktreePath, kustomizationPath)
	headPath := filepath.Join(absRoot, kustomizationPath)

	return computeBody(ctx, basePath, headPath, comparer)
}

// computeBody extracts operator refs from the two kustomization files and returns
// the appropriate comment body. It is the testable core of buildBody — all git
// setup happens in buildBody; this function only does file I/O, API calls, and
// formatting.
//
// If the GitHub API call to fetch operator file diffs fails, the comment still
// includes the compare link but notes that service bump detection was unavailable.
func computeBody(ctx context.Context, basePath, headPath string, comparer changelog.RepoComparer) (string, error) {
	oldRef, newRef, err := changelog.ExtractRefs(basePath, headPath)
	if err != nil {
		return "", fmt.Errorf("extracting operator refs: %w", err)
	}

	if oldRef == newRef {
		slog.Info("Operator ref unchanged — no changelog needed")
		return formatNoChange(), nil
	}
	slog.Info("Operator ref bumped", "old", oldRef, "new", newRef)

	compare, err := changelog.FetchOperatorCompare(ctx, comparer, oldRef, newRef)
	if err != nil {
		slog.Warn("fetching operator compare; service bumps unavailable", "err", err)
		return formatCompare(oldRef, newRef, nil), nil
	}
	if compare.Truncated {
		slog.Warn("operator compare truncated (≥300 files); service bump detection unavailable")
		return formatCompare(oldRef, newRef, nil), nil
	}

	bumps, hasSkipped := changelog.ExtractServiceBumps(compare.Files)
	if hasSkipped {
		// One or more upstream kustomization files had no patch data — we cannot
		// confidently say "no service refs changed", so degrade the same way we
		// do for API failures rather than risk a misleading empty-bumps message.
		slog.Warn("some upstream kustomization patches were empty; service bump detection may be incomplete")
		return formatCompare(oldRef, newRef, nil), nil
	}
	slog.Info("Service bumps detected", "count", len(bumps))

	enriched := enrichWithCommits(ctx, comparer, bumps)
	return formatCompare(oldRef, newRef, enriched), nil
}

// bumpWithCommits pairs a ServiceBump with the conventional commits found
// between its old and new SHAs.
type bumpWithCommits struct {
	Bump      changelog.ServiceBump
	Commits   []changelog.ConventionalCommit
	Failed    bool // true when the commit fetch API call failed
	Truncated bool // true when AheadBy hit the 250-commit API limit
}

// enrichWithCommits fetches feat/fix commits for each bump and returns a
// slice ready for rendering. API errors per service are logged and surfaced
// as Failed=true so the bump header is still shown without commits.
func enrichWithCommits(ctx context.Context, comparer changelog.RepoComparer, bumps []changelog.ServiceBump) []bumpWithCommits {
	// Explicit empty (not nil) so formatCompare shows "no refs changed", not
	// the degraded "detection unavailable" message.
	if len(bumps) == 0 {
		return []bumpWithCommits{}
	}
	result := make([]bumpWithCommits, len(bumps))
	for i, bump := range bumps {
		commits, truncated, err := changelog.FetchServiceCommits(ctx, comparer, bump)
		if err != nil {
			slog.Warn("fetching service commits", "service", bump.Repo, "err", err)
			result[i] = bumpWithCommits{Bump: bump, Failed: true}
			continue
		}
		result[i] = bumpWithCommits{Bump: bump, Commits: commits, Truncated: truncated}
	}
	return result
}

// formatNoChange returns the comment body when the operator ref did not change.
func formatNoChange() string {
	return commentMarker + "\n### Operator Changelog\n\nNo operator ref change detected in this PR.\n"
}

// formatCompare returns the comment body with the operator compare link, the
// list of upstream service bumps, and per-service feat/fix commit summaries.
//
// bumps == nil means the service bump detection API call failed (degraded).
// bumps == [] means the call succeeded but no sub-service SHAs changed.
func formatCompare(oldRef, newRef string, bumps []bumpWithCommits) string {
	const base = "https://github.com/konflux-ci/konflux-ci"
	short := func(ref string) string {
		if len(ref) > 12 {
			return ref[:12]
		}
		return ref
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n### Operator Changelog\n\n", commentMarker)
	fmt.Fprintf(&b, "Comparing [`%s`](%s/commit/%s) → [`%s`](%s/commit/%s)\n\n",
		short(oldRef), base, oldRef,
		short(newRef), base, newRef)
	fmt.Fprintf(&b, "[Full diff](%s/compare/%s...%s)\n\n", base, oldRef, newRef)

	switch {
	case bumps == nil:
		fmt.Fprintln(&b, "_Upstream service bump detection unavailable — check manually._")
	case len(bumps) == 0:
		fmt.Fprintln(&b, "_No upstream service refs changed._")
	default:
		fmt.Fprintln(&b, "#### Upstream Service Bumps")
		fmt.Fprintln(&b)
		for _, bwc := range bumps {
			bump := bwc.Bump
			compareURL := fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s",
				bump.Owner, bump.Repo, bump.OldSHA, bump.NewSHA)
			oldURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", bump.Owner, bump.Repo, bump.OldSHA)
			newURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", bump.Owner, bump.Repo, bump.NewSHA)
			fmt.Fprintf(&b, "**%s** [`%s`](%s) → [`%s`](%s) ([compare](%s))\n",
				bump.Repo,
				short(bump.OldSHA), oldURL,
				short(bump.NewSHA), newURL,
				compareURL)

			switch {
			case bwc.Failed:
				fmt.Fprintf(&b, "\n_Commit details unavailable._\n\n")
			case len(bwc.Commits) == 0:
				fmt.Fprintf(&b, "\n_No notable commits (feat/fix)._\n\n")
			default:
				fmt.Fprintln(&b)
				for _, c := range bwc.Commits {
					if c.Scope != "" {
						fmt.Fprintf(&b, "- %s(%s): %s\n", c.Type, c.Scope, c.Subject)
					} else {
						fmt.Fprintf(&b, "- %s: %s\n", c.Type, c.Subject)
					}
				}
				if bwc.Truncated {
					fmt.Fprintf(&b, "\n_Showing first %d commits only — results may be incomplete._\n", changelog.CommitMaxFromCompare)
				}
				fmt.Fprintln(&b)
			}
		}
	}

	return b.String()
}

// resolveRepoRoot returns the absolute path to the repository root.
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

// post delivers body as a PR comment using the provided commenter.
func post(ctx context.Context, c commenter, prNumber int, body string) error {
	return c.UpsertCommentByMarker(ctx, prNumber, body, commentMarker)
}
