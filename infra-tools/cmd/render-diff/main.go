// Command render-diff computes and displays the kustomize render delta for
// components affected by the current branch's changes.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/logging"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// version is set via -ldflags at build time.
var version = "dev"

// options holds the parsed command-line configuration.
type options struct {
	repoRoot    string
	baseRef     string
	overlaysDir string
	color       string
	outputDir   string
	outputMode  string
	logFile     string
	openDiff    bool
	showVersion bool

	// modes is the validated, deduplicated list of output modes.
	modes []OutputMode
}

func main() {
	var opts options
	flag.StringVar(&opts.repoRoot, "repo-root", "", "Path to the repository root (default: auto-detect via git)")
	flag.StringVar(&opts.baseRef, "base-ref", "", "Base git ref to compare against (default: merge-base with main)")
	flag.StringVar(&opts.overlaysDir, "overlays-dir", "argo-cd-apps/overlays", "Path to overlays directory relative to repo root")
	flag.StringVar(&opts.color, "color", "auto", "Color output: auto, always, never")
	flag.BoolVar(&opts.openDiff, "open", false, "Open diffs in $DIFFTOOL or git difftool")
	flag.StringVar(&opts.outputDir, "output-dir", "", "Write per-component .diff files to this directory")
	flag.StringVar(&opts.outputMode, "output-mode", "local", "Output mode: local, ci-summary, ci-comment, ci-artifact-dir")
	flag.BoolVar(&opts.showVersion, "version", false, "Print version and exit")
	flag.StringVar(&opts.logFile, "log-file", "", "Write debug-level logs to this file")
	flag.Parse()

	if opts.showVersion {
		fmt.Printf("render-diff %s\n", version)
		return
	}

	// Parse and validate output modes (comma-separated).
	opts.modes = parseOutputModes(opts.outputMode)
	if len(opts.modes) == 0 {
		fmt.Fprintf(os.Stderr, "invalid --output-mode %q: must be one or more of local, ci-summary, ci-comment, ci-artifact-dir (comma-separated)\n", opts.outputMode)
		os.Exit(1)
	}

	switch opts.color {
	case "auto", "always", "never":
		// valid
	default:
		fmt.Fprintf(os.Stderr, "invalid --color %q: must be one of auto, always, never\n", opts.color)
		os.Exit(1)
	}

	logCleanup, err := logging.Setup(opts.logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logging: %v\n", err)
		os.Exit(1)
	}
	if logCleanup != nil {
		defer logCleanup()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// run owns the worktree and other deferred cleanup: returning an error
	// instead of calling os.Exit keeps those defers reachable.
	if err := run(ctx, opts); err != nil {
		logging.Fatal("render-diff failed", "err", err)
	}
}

// run executes the whole render-diff pipeline.
func run(ctx context.Context, opts options) error {
	// Auto-detect repo root via git if not specified.
	repoRoot := opts.repoRoot
	if repoRoot == "" {
		detected, err := git.TopLevel(ctx)
		if err != nil {
			return fmt.Errorf("auto-detecting repo root (use --repo-root to specify explicitly): %w", err)
		}
		repoRoot = detected
	}

	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	// Always compare against the merge-base, not the ref itself: with
	// --base-ref=origin/main (how CI invokes this), a main that moved after the
	// PR branch was created would otherwise appear as changes of this PR.
	baseRefName := opts.baseRef
	if baseRefName == "" {
		baseRefName = "main"
	}
	effectiveBaseRef, err := git.MergeBase(ctx, absRepoRoot, baseRefName)
	if err != nil {
		return fmt.Errorf("computing merge-base with %s (use --base-ref to specify explicitly): %w", baseRefName, err)
	}

	baseSHA, err := git.ResolveRef(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		return fmt.Errorf("resolving base ref: %w", err)
	}
	headSHA, err := git.ResolveRef(ctx, absRepoRoot, "HEAD")
	if err != nil {
		return fmt.Errorf("resolving HEAD: %w", err)
	}
	slog.Info("Comparing refs", "head", headSHA, "base", baseSHA)

	// Local runs render the working tree, so uncommitted edits and untracked
	// files are part of the change set. CI renders committed refs only.
	localOnly := len(opts.modes) == 1 && opts.modes[0] == OutputModeLocal

	// Step 1: Get changed files
	var changedFiles []string
	if localOnly {
		changedFiles, err = git.ChangedFilesWorkingTree(ctx, absRepoRoot, effectiveBaseRef)
	} else {
		changedFiles, err = git.ChangedFiles(ctx, absRepoRoot, effectiveBaseRef)
	}
	if err != nil {
		return fmt.Errorf("getting changed files: %w", err)
	}
	if len(changedFiles) == 0 {
		fmt.Println("No changed files detected — nothing to diff.")
		// Best-effort: update CI comment/summary so they don't stay stale.
		// Failures here are non-fatal since there is nothing to report.
		_ = runAllOutputModes(ctx, opts.modes, &renderdiff.DiffResult{}, opts.color, opts.openDiff, opts.outputDir, headSHA, baseSHA)
		return nil
	}
	slog.Info("Changed files detected", "count", len(changedFiles))

	// Step 2: Create worktree at base ref
	worktreePath, cleanup, err := git.CreateWorktree(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}
	defer cleanup()

	headRef := detector.NewRepoRef(absRepoRoot)
	baseRefRepo := detector.NewRepoRef(worktreePath)

	// Step 3: Detect affected components
	slog.Info("Detecting affected components...")
	d, err := detector.NewDetector(headRef, baseRefRepo, opts.overlaysDir)
	if err != nil {
		return fmt.Errorf("initializing detector: %w", err)
	}
	affected, err := d.AffectedComponents(changedFiles)
	if err != nil {
		return fmt.Errorf("detecting affected components: %w", err)
	}

	// Count total jobs
	totalJobs := 0
	for _, paths := range affected {
		totalJobs += len(paths)
	}
	if totalJobs == 0 {
		fmt.Println("No affected components detected — nothing to diff.")
		// Best-effort: update CI comment/summary so they don't stay stale.
		// Failures here are non-fatal since there is nothing to report.
		_ = runAllOutputModes(ctx, opts.modes, &renderdiff.DiffResult{}, opts.color, opts.openDiff, opts.outputDir, headSHA, baseSHA)
		return nil
	}
	slog.Info("Affected component paths detected", "count", totalJobs)

	// Step 4: Run render-diff engine (once for all output modes).
	engine := renderdiff.NewEngine(headRef, baseRefRepo, totalJobs)

	// For local mode (single mode only), use progressive output.
	if localOnly {
		return runLocal(ctx, engine, affected, opts.color, opts.openDiff, opts.outputDir)
	}

	// For CI modes (possibly multiple), build once and share the result.
	result, err := engine.Run(ctx, affected)
	if err != nil {
		return fmt.Errorf("running render-diff engine: %w", err)
	}

	if hadError := runAllOutputModes(ctx, opts.modes, result, opts.color, opts.openDiff, opts.outputDir, headSHA, baseSHA); hadError {
		return errors.New("one or more output modes failed")
	}
	return nil
}
