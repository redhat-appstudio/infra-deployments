// Command render-diff computes and displays the kustomize render delta for
// components affected by the current branch's changes.
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

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/logging"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// version is set via -ldflags at build time.
var version = "dev"

// OutputMode controls how render-diff formats and delivers its output.
type OutputMode string

const (
	OutputModeLocal      OutputMode = "local"
	OutputModeCISummary  OutputMode = "ci-summary"
	OutputModeCIComment  OutputMode = "ci-comment"
	OutputModeCIArtifact OutputMode = "ci-artifact-dir"
)

func main() {
	var (
		repoRoot    = flag.String("repo-root", "", "Path to the repository root (default: auto-detect via git)")
		baseRef     = flag.String("base-ref", "", "Base git ref to compare against (default: merge-base with main)")
		overlaysDir = flag.String("overlays-dir", "argo-cd-apps/overlays", "Path to overlays directory relative to repo root")
		color       = flag.String("color", "auto", "Color output: auto, always, never")
		openDiff    = flag.Bool("open", false, "Open diffs in $DIFFTOOL or git difftool")
		outputDir   = flag.String("output-dir", "", "Write per-component .diff files to this directory")
		outputMode  = flag.String("output-mode", "local", "Output mode: local, ci-summary, ci-comment, ci-artifact-dir")
		showVersion = flag.Bool("version", false, "Print version and exit")
		logFile     = flag.String("log-file", "", "Write debug-level logs to this file")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("render-diff %s\n", version)
		os.Exit(0)
	}

	// Parse and validate output modes (comma-separated).
	modes := parseOutputModes(*outputMode)
	if len(modes) == 0 {
		fmt.Fprintf(os.Stderr, "invalid --output-mode %q: must be one or more of local, ci-summary, ci-comment, ci-artifact-dir (comma-separated)\n", *outputMode)
		os.Exit(1)
	}

	switch *color {
	case "auto", "always", "never":
		// valid
	default:
		fmt.Fprintf(os.Stderr, "invalid --color %q: must be one of auto, always, never\n", *color)
		os.Exit(1)
	}

	// Set up logging
	logCleanup, err := logging.Setup(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logging: %v\n", err)
		os.Exit(1)
	}
	if logCleanup != nil {
		defer logCleanup()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Auto-detect repo root via git if not specified.
	if *repoRoot == "" {
		detected, err := git.TopLevel(ctx)
		if err != nil {
			logging.Fatal("auto-detecting repo root; use --repo-root to specify explicitly", "err", err)
		}
		repoRoot = &detected
	}

	absRepoRoot, err := filepath.Abs(*repoRoot)
	if err != nil {
		logging.Fatal("resolving repo root", "err", err)
	}

	// Resolve base ref: default to merge-base with main
	effectiveBaseRef := *baseRef
	if effectiveBaseRef == "" {
		effectiveBaseRef, err = git.MergeBase(ctx, absRepoRoot, "main")
		if err != nil {
			logging.Fatal("could not compute merge-base with main; use --base-ref to specify explicitly", "err", err)
		}
	}

	baseSHA, err := git.ResolveRef(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		logging.Fatal("resolving base ref", "err", err)
	}

	headSHA, err := git.ResolveRef(ctx, absRepoRoot, "HEAD")
	if err != nil {
		logging.Fatal("resolving HEAD", "err", err)
	}
	slog.Info("Comparing refs", "head", headSHA, "base", baseSHA)

	// Step 1: Get changed files
	changedFiles, err := git.ChangedFiles(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		logging.Fatal("getting changed files", "err", err)
	}
	if len(changedFiles) == 0 {
		fmt.Println("No changed files detected — nothing to diff.")
		return
	}
	slog.Info("Changed files detected", "count", len(changedFiles))

	// Step 2: Create worktree at base ref
	worktreePath, cleanup, err := git.CreateWorktree(ctx, absRepoRoot, effectiveBaseRef)
	if err != nil {
		logging.Fatal("creating worktree", "err", err)
	}
	defer cleanup()

	headRef := detector.NewRepoRef(absRepoRoot)
	baseRefRepo := detector.NewRepoRef(worktreePath)

	// Step 3: Detect affected components
	slog.Info("Detecting affected components...")
	d, err := detector.NewDetector(headRef, baseRefRepo, *overlaysDir)
	if err != nil {
		logging.Fatal("initializing detector", "err", err)
	}
	affected, err := d.AffectedComponents(changedFiles)
	if err != nil {
		logging.Fatal("detecting affected components", "err", err)
	}

	// Count total jobs
	totalJobs := 0
	for _, paths := range affected {
		totalJobs += len(paths)
	}
	if totalJobs == 0 {
		fmt.Println("No affected components detected — nothing to diff.")
		return
	}
	slog.Info("Affected component paths detected", "count", totalJobs)

	// Step 4: Run render-diff engine (once for all output modes).
	engine := renderdiff.NewEngine(headRef, baseRefRepo, totalJobs)

	// For local mode (single mode only), use progressive output.
	if len(modes) == 1 && modes[0] == OutputModeLocal {
		runLocal(ctx, engine, affected, *color, *openDiff, *outputDir)
		return
	}

	// For CI modes (possibly multiple), build once and share the result.
	result, err := engine.Run(ctx, affected)
	if err != nil {
		logging.Fatal("render-diff failed", "err", err)
	}

	var hadError bool
	for _, m := range modes {
		if err := runOutputMode(ctx, m, result, *color, *openDiff, *outputDir, headSHA, baseSHA); err != nil {
			slog.Error("output mode failed, continuing with remaining modes", "mode", m, "err", err)
			hadError = true
		}
	}
	if hadError {
		os.Exit(1)
	}
}

// parseOutputModes splits a comma-separated output-mode string, validates each
// value, and returns the deduplicated list. Returns nil if any value is invalid.
func parseOutputModes(raw string) []OutputMode {
	seen := make(map[OutputMode]bool)
	var modes []OutputMode
	for s := range strings.SplitSeq(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		m := OutputMode(s)
		switch m {
		case OutputModeLocal, OutputModeCISummary, OutputModeCIComment, OutputModeCIArtifact:
			if !seen[m] {
				seen[m] = true
				modes = append(modes, m)
			}
		default:
			return nil
		}
	}
	return modes
}
