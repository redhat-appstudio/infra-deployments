// Command changelog-generator posts a changelog comment on infra-deployments PRs
// that bump the Konflux operator SHA.
//
// This is step 2 (KFLUXVNGD-1023): it reads the operator SHA at the base branch
// and PR head, then posts a comment with the compare link. Upstream service
// changes are introduced in subsequent PRs.
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

	body, err := buildBody(ctx, *repoRoot, *baseRef)
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

// buildBody extracts the old and new operator SHAs and returns the comment body.
func buildBody(ctx context.Context, repoRoot, baseRef string) (string, error) {
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

	return computeBody(basePath, headPath)
}

// computeBody extracts refs from the two kustomization files and returns the
// appropriate comment body. It is the testable core of buildBody — all git
// setup happens in buildBody; this function only does file I/O and formatting.
func computeBody(basePath, headPath string) (string, error) {
	oldRef, newRef, err := changelog.ExtractRefs(basePath, headPath)
	if err != nil {
		return "", fmt.Errorf("extracting operator refs: %w", err)
	}

	if oldRef != newRef {
		slog.Info("Operator ref bumped", "old", oldRef, "new", newRef)
	} else {
		slog.Info("Operator ref unchanged — no changelog needed")
	}
	return selectBody(oldRef, newRef), nil
}

// selectBody returns the appropriate comment body based on whether the ref changed.
func selectBody(oldRef, newRef string) string {
	if oldRef == newRef {
		return formatNoChange()
	}
	return formatCompare(oldRef, newRef)
}

// formatNoChange returns the comment body when the operator ref did not change.
func formatNoChange() string {
	return commentMarker + "\n### Operator Changelog\n\nNo operator ref change detected in this PR.\n"
}

// formatCompare returns the comment body with a compare link between the two refs.
// Upstream service changes will be added in the next PR.
func formatCompare(oldRef, newRef string) string {
	const base = "https://github.com/konflux-ci/konflux-ci"
	short := func(ref string) string {
		if len(ref) > 12 {
			return ref[:12]
		}
		return ref
	}
	return fmt.Sprintf(
		"%s\n### Operator Changelog\n\nComparing [`%s`](%s/commit/%s) → [`%s`](%s/commit/%s)\n\n[Full diff](%s/compare/%s...%s)\n\n_Upstream service changes coming in the next PR._\n",
		commentMarker,
		short(oldRef), base, oldRef,
		short(newRef), base, newRef,
		base, oldRef, newRef,
	)
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
