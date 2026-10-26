// Command env-detector detects which environments and clusters a PR affects
// by analysing kustomize overlays and dependency trees.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"syscall"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/logging"
)

// errRingViolation signals a ring deployment policy violation. The message has
// already been printed; it only exists so main can exit non-zero after deferred
// cleanup has run.
var errRingViolation = errors.New("ring deployment violation")

// options holds the parsed command-line configuration.
type options struct {
	repoRoot          string
	baseRef           string
	overlaysDir       string
	githubToken       string
	repo              string
	logFile           string
	ringReportFile    string
	prNumber          int
	dryRun            bool
	clusterLabels     bool
	enforceRingDeploy bool
}

func main() {
	var opts options
	flag.StringVar(&opts.repoRoot, "repo-root", ".", "Path to the repository root")
	flag.StringVar(&opts.baseRef, "base-ref", "main", "Base git ref to compare against")
	flag.StringVar(&opts.overlaysDir, "overlays-dir", "argo-cd-apps/overlays", "Path to overlays directory relative to repo root")
	flag.BoolVar(&opts.dryRun, "dry-run", false, "Print results without calling GitHub API")
	flag.IntVar(&opts.prNumber, "pr-number", 0, "PR number to label (required if not --dry-run)")
	flag.StringVar(&opts.githubToken, "github-token", "", "GitHub token (required if not --dry-run)")
	flag.StringVar(&opts.repo, "repo", "", "GitHub repository in owner/repo format (required if not --dry-run)")
	flag.BoolVar(&opts.clusterLabels, "cluster-labels", false, "Include cluster/<name> labels in addition to environment labels")
	flag.StringVar(&opts.logFile, "log-file", "", "Write debug-level logs to this file (in addition to INFO-level logs on stdout)")
	flag.BoolVar(&opts.enforceRingDeploy, "enforce-ring-deployment", false, "Fail when both staging and production overlays are directly modified in the same PR")
	flag.StringVar(&opts.ringReportFile, "ring-report-file", "", "Write ring deployment check result (markdown) to this file for external consumers like PR comments")
	flag.Parse()

	// Set up logging: INFO on stdout (always), DEBUG to file (when --log-file is set).
	logCleanup, err := logging.Setup(opts.logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logging: %v\n", err)
		os.Exit(1)
	}
	if logCleanup != nil {
		defer logCleanup()
	}

	if !opts.dryRun && (opts.prNumber == 0 || opts.githubToken == "" || opts.repo == "") {
		fmt.Fprintln(os.Stderr, "--pr-number, --github-token, and --repo are required when not using --dry-run")
		os.Exit(1)
	}

	// Set up a context that is cancelled on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// run owns the worktree cleanup: returning an error instead of calling
	// os.Exit keeps the deferred cleanup reachable.
	if err := run(ctx, opts); err != nil {
		if !errors.Is(err, errRingViolation) {
			slog.Error("env-detector failed", "err", err)
		}
		os.Exit(1)
	}
}

// run executes the detection pipeline and returns any error.
func run(ctx context.Context, opts options) error {
	// Resolve repo root to absolute path
	absRepoRoot, err := filepath.Abs(opts.repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	// Resolve base ref: use merge-base so the diff only contains the PR's
	// own changes, even when the branch is not rebased on top of base-ref
	// or the GitHub merge ref is stale.  This mirrors the approach used by
	// render-diff.
	effectiveBaseRef, err := git.MergeBase(ctx, absRepoRoot, opts.baseRef)
	if err != nil {
		return fmt.Errorf("computing merge-base with %s: %w", opts.baseRef, err)
	}
	slog.Debug("Resolved merge-base", "baseRef", opts.baseRef, "mergeBase", effectiveBaseRef)

	// Resolve HEAD and effective base to short commit SHAs for the summary.
	headSHA, err := git.ResolveRef(ctx, absRepoRoot, "HEAD")
	if err != nil {
		return fmt.Errorf("resolving HEAD: %w", err)
	}
	baseSHA, err := git.ResolveRef(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		return fmt.Errorf("resolving base ref: %w", err)
	}

	// Step 1: Get changed files via git diff
	slog.Info("Getting changed files...")
	changedFiles, err := git.ChangedFiles(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		return fmt.Errorf("getting changed files: %w", err)
	}
	if len(changedFiles) == 0 {
		slog.Info("No changed files detected")
		if !opts.dryRun {
			if err := syncLabels(ctx, opts.githubToken, opts.repo, opts.prNumber, []string{"environment/none"}); err != nil {
				return fmt.Errorf("syncing labels: %w", err)
			}
		}
		return nil
	}

	// Step 2: Create a temporary git worktree at the merge-base
	slog.Info("Creating worktree...", "ref", effectiveBaseRef)
	worktreePath, cleanup, err := git.CreateWorktree(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	defer cleanup()

	// Step 3: Run detection
	slog.Info("Running detection...")
	d, err := detector.NewDetector(
		detector.NewRepoRef(absRepoRoot),
		detector.NewRepoRef(worktreePath),
		opts.overlaysDir,
	)
	if err != nil {
		return fmt.Errorf("initializing detector: %w", err)
	}
	result, err := d.Detect(changedFiles)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Step 4: Output results
	labelSet := result.Labels()

	var labels []string
	if opts.clusterLabels {
		labels = slices.Collect(labelSet.All())
	} else {
		labels = labelSet.Environments
	}

	// If no environment was affected, use an explicit "none" label so it's
	// clear the tool ran and determined there is no environment impact.
	if len(labelSet.Environments) == 0 {
		labels = append(labels, "environment/none")
	}

	// Files under components/ or configs/ that no ApplicationSet references are
	// not deployed by anything: flag them so reviewers know the change is inert
	// instead of reading "environment/none" as "safe to merge".
	if len(result.UnreferencedFiles) > 0 {
		labels = append(labels, ghclient.UnreferencedChangesLabel)
	}

	// If production is affected, add a hold label that Prow Tide can use to
	// block merging until a human explicitly removes it after review. This
	// applies to every PR, including automated promotions: the hold exists so a
	// human looks at production changes before they land.
	if result.AffectedEnvironments[detector.Production] {
		labels = append(labels, ghclient.HoldProductionLabel)
		labels = append(labels, ghclient.NeedsApprovalProductionLabel)
	}

	printSummary(result, labels, headSHA, baseSHA)
	if len(result.UnreferencedFiles) > 0 {
		msg := formatUnreferencedChanges(result.UnreferencedFiles)
		fmt.Println(msg)
		writeStepSummary(msg)
	}

	if !opts.dryRun {
		// Step 5: Sync labels via GitHub API
		slog.Info("Syncing labels...")
		if err := syncLabels(ctx, opts.githubToken, opts.repo, opts.prNumber, labels); err != nil {
			return fmt.Errorf("syncing labels: %w", err)
		}
	}

	// Step 6: Ring deployment enforcement (runs in both dry-run and normal mode)
	if opts.enforceRingDeploy {
		ringResult := detector.CheckRingDeployment(changedFiles, result.AffectedEnvironments, result.FileEnvs)
		if ringResult.DirectConflict {
			msg := formatRingViolation(ringResult)
			fmt.Println(msg)
			writeStepSummary(msg)
			writeReportFile(opts.ringReportFile, msg)
			return errRingViolation
		}
		if ringResult.IndirectConflict {
			msg := formatRingWarning()
			fmt.Println(msg)
			writeStepSummary(msg)
			writeReportFile(opts.ringReportFile, msg)
		}
	}

	slog.Info("Done!")
	return nil
}

// printSummary prints the detection results in a human-friendly format.
func printSummary(result *detector.Result, labels []string, headSHA, baseSHA string) {
	fmt.Printf("\nHEAD: %s\n", headSHA)
	fmt.Printf("Base: %s\n", baseSHA)

	fmt.Println("\nChanged files:")
	sort.Strings(result.ChangedFiles)
	for _, f := range result.ChangedFiles {
		fmt.Printf("  %s\n", f)
	}

	fmt.Println("\nAffected environments:")
	envs := slices.Sorted(maps.Keys(result.AffectedEnvironments))
	if len(envs) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, env := range envs {
			fmt.Printf("  - %s\n", env)
		}
	}

	fmt.Println("\nAffected clusters:")
	clusters := slices.Sorted(maps.Keys(result.AffectedClusters))
	if len(clusters) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, cluster := range clusters {
			fmt.Printf("  - %s\n", cluster)
		}
	}

	if len(result.UnreferencedFiles) > 0 {
		fmt.Println("\nUnreferenced files (not deployed by any ApplicationSet):")
		for _, f := range result.UnreferencedFiles {
			fmt.Printf("  %s\n", f)
		}
	}

	fmt.Println("\nLabels that would be applied:")
	if len(labels) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, label := range labels {
			fmt.Printf("  - %s\n", label)
		}
	}
}

// formatUnreferencedChanges returns a markdown warning listing changed files
// that no ApplicationSet references, so they will not be deployed.
func formatUnreferencedChanges(files []string) string {
	var b strings.Builder
	b.WriteString("\n## Unreferenced changes\n\n")
	b.WriteString("These files changed in this PR, but no ApplicationSet in any overlay references them, so the change will not be deployed:\n\n")
	for _, f := range files {
		fmt.Fprintf(&b, "- `%s`\n", f)
	}
	b.WriteString("\nIf you expected these changes to reach a cluster, check that the component is still deployed by an overlay ")
	b.WriteString("(the ApplicationSet may have moved to a ring layout or been removed).\n")
	return b.String()
}

// syncLabels calls the GitHub API to sync labels on the PR.
func syncLabels(ctx context.Context, token, repoName string, prNumber int, labels []string) error {
	client, err := ghclient.NewClient(token, repoName)
	if err != nil {
		return err
	}
	return client.SyncLabels(ctx, prNumber, labels)
}

// formatRingViolation returns a markdown message for a direct staging+production conflict.
func formatRingViolation(r *detector.RingCheckResult) string {
	var b strings.Builder
	b.WriteString("\n## Ring Deployment Violation\n\n")
	b.WriteString("This PR modifies both **staging** and **production** overlays, which violates the ring deployment policy.\n")
	b.WriteString("Changes must be validated in staging before promoting to production.\n\n")
	b.WriteString("Please split this PR into two:\n")
	b.WriteString("1. First PR: staging changes only\n")
	b.WriteString("2. Second PR: production changes (after staging is validated)\n")

	b.WriteString("\n### Staging files\n")
	for _, f := range r.StagingFiles {
		fmt.Fprintf(&b, "- `%s`\n", f)
	}
	b.WriteString("\n### Production files\n")
	for _, f := range r.ProductionFiles {
		fmt.Fprintf(&b, "- `%s`\n", f)
	}
	return b.String()
}

// formatRingWarning returns a markdown message for an indirect (base-only) conflict.
func formatRingWarning() string {
	var b strings.Builder
	b.WriteString("\n## Ring Deployment Warning\n\n")
	b.WriteString("This PR modifies shared base files that affect both **staging** and **production** environments.\n\n")
	b.WriteString("If possible, split this PR into two:\n")
	b.WriteString("1. First PR: staging changes only\n")
	b.WriteString("2. Second PR: revert staging-only changes and apply changes to the shared base files (after staging is validated)\n\n")
	b.WriteString("If this is not possible, **excercise extreme caution**!\n")
	return b.String()
}

// writeReportFile writes markdown to a report file for use by external
// consumers (e.g. a workflow step that posts a PR comment).
func writeReportFile(path, markdown string) {
	if path == "" {
		return
	}
	if err := os.WriteFile(path, []byte(markdown), 0600); err != nil {
		slog.Warn("failed to write ring report file", "path", path, "err", err)
	}
}

// writeStepSummary appends markdown to the GitHub Actions step summary file.
func writeStepSummary(markdown string) {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		return
	}
	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Warn("failed to open GITHUB_STEP_SUMMARY", "err", err)
		return
	}
	defer f.Close()
	fmt.Fprintln(f, markdown)
}
