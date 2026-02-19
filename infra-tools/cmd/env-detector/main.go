// Command env-detector detects which environments and clusters a PR affects
// by analysing kustomize overlays and dependency trees.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"sort"
	"syscall"

	charmlog "github.com/charmbracelet/log"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
	ghclient "github.com/redhat-appstudio/infra-deployments/infra-tools/internal/github"
)

func main() {
	var (
		repoRoot      = flag.String("repo-root", ".", "Path to the repository root")
		baseRef       = flag.String("base-ref", "main", "Base git ref to compare against")
		overlaysDir = flag.String("overlays-dir", "argo-cd-apps/overlays", "Path to overlays directory relative to repo root")
		dryRun        = flag.Bool("dry-run", false, "Print results without calling GitHub API")
		prNumber      = flag.Int("pr-number", 0, "PR number to label (required if not --dry-run)")
		githubToken   = flag.String("github-token", "", "GitHub token (required if not --dry-run)")
		repo          = flag.String("repo", "", "GitHub repository in owner/repo format (required if not --dry-run)")
		clusterLabels = flag.Bool("cluster-labels", false, "Include cluster/<name> labels in addition to environment labels")
		logFile       = flag.String("log-file", "", "Write debug-level logs to this file (in addition to INFO-level logs on stdout)")
	)
	flag.Parse()

	// Set up logging: INFO on stdout (always), DEBUG to file (when --log-file is set).
	logCleanup, err := setupLogging(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logging: %v\n", err)
		os.Exit(1)
	}
	if logCleanup != nil {
		defer logCleanup()
	}

	if !*dryRun {
		if *prNumber == 0 || *githubToken == "" || *repo == "" {
			fatal("--pr-number, --github-token, and --repo are required when not using --dry-run")
		}
	}

	// Set up a context that is cancelled on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Resolve repo root to absolute path
	absRepoRoot, err := filepath.Abs(*repoRoot)
	if err != nil {
		fatal("resolving repo root", "err", err)
	}

	// Resolve HEAD and base-ref to short commit SHAs for the summary.
	headSHA, err := git.ResolveRef(ctx, absRepoRoot, "HEAD")
	if err != nil {
		fatal("resolving HEAD", "err", err)
	}
	baseSHA, err := git.ResolveRef(ctx, absRepoRoot, *baseRef)
	if err != nil {
		fatal("resolving base ref", "err", err)
	}

	// Step 1: Get changed files via git diff
	slog.Info("Getting changed files...")
	changedFiles, err := git.ChangedFiles(ctx, absRepoRoot, *baseRef)
	if err != nil {
		fatal("getting changed files", "err", err)
	}
	if len(changedFiles) == 0 {
		slog.Info("No changed files detected")
		if !*dryRun {
			if err := syncLabels(ctx, *githubToken, *repo, *prNumber, []string{"environment/none"}); err != nil {
				fatal("syncing labels", "err", err)
			}
		}
		return
	}

	// Step 2: Create a temporary git worktree at base-ref
	slog.Info("Creating worktree...", "ref", *baseRef)
	worktreePath, cleanup, err := git.CreateWorktree(ctx, absRepoRoot, *baseRef)
	if err != nil {
		fatal("creating worktree", "err", err)
	}
	defer cleanup()

	// Step 3: Run detection
	slog.Info("Running detection...")
	d, err := detector.NewDetector(
		detector.NewRepoRef(absRepoRoot),
		detector.NewRepoRef(worktreePath),
		*overlaysDir,
	)
	if err != nil {
		fatal("initializing detector", "err", err)
	}
	result, err := d.Detect(changedFiles)
	if err != nil {
		fatal("detection failed", "err", err)
	}

	// Step 4: Output results
	labelSet := result.Labels()

	var labels []string
	if *clusterLabels {
		labels = slices.Collect(labelSet.All())
	} else {
		labels = labelSet.Environments
	}

	// If no environment was affected, use an explicit "none" label so it's
	// clear the tool ran and determined there is no environment impact.
	if len(labelSet.Environments) == 0 {
		labels = append(labels, "environment/none")
	}

	// If production is affected, add a hold label that Prow Tide can use to
	// block merging until a human explicitly removes it after review.
	if result.AffectedEnvironments[detector.Production] {
		labels = append(labels, ghclient.HoldProductionLabel)
	}

	printSummary(result, labels, headSHA, baseSHA)

	if *dryRun {
		return
	}

	// Step 5: Sync labels via GitHub API
	slog.Info("Syncing labels...")
	if err := syncLabels(ctx, *githubToken, *repo, *prNumber, labels); err != nil {
		fatal("syncing labels", "err", err)
	}
	slog.Info("Done!")
}

// fatal logs an error message and exits with code 1.
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// setupLogging configures slog with charmbracelet/log handlers.
// When logFile is non-empty, a multi-handler is set up so that:
//   - stdout receives INFO-level and above (with pretty charmbracelet formatting)
//   - the file receives DEBUG-level and above (with timestamps)
//
// When logFile is empty, only the stdout handler at INFO level is used.
// Returns a cleanup function (may be nil) that closes the log file.
func setupLogging(logFile string) (func(), error) {
	stdoutHandler := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		Level: charmlog.InfoLevel,
	})

	if logFile == "" {
		slog.SetDefault(slog.New(stdoutHandler))
		return nil, nil
	}

	f, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", logFile, err)
	}

	fileHandler := charmlog.NewWithOptions(f, charmlog.Options{
		Level:           charmlog.DebugLevel,
		ReportTimestamp: true,
	})

	multi := &multiHandler{handlers: []slog.Handler{stdoutHandler, fileHandler}}
	slog.SetDefault(slog.New(multi))

	return func() { _ = f.Close() }, nil
}

// multiHandler is an slog.Handler that fans out log records to multiple
// underlying handlers, each of which may have its own level filter.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(_ context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(context.Background(), level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
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

	fmt.Println("\nLabels that would be applied:")
	if len(labels) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, label := range labels {
			fmt.Printf("  - %s\n", label)
		}
	}
}

// syncLabels calls the GitHub API to sync labels on the PR.
func syncLabels(ctx context.Context, token, repoName string, prNumber int, labels []string) error {
	client, err := ghclient.NewClient(token, repoName)
	if err != nil {
		return err
	}
	return client.SyncLabels(ctx, prNumber, labels)
}
