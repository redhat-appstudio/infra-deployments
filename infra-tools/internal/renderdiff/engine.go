package renderdiff

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
)

// RepoBuilder abstracts the ability to check directory existence and build
// kustomizations on a specific git ref.
type RepoBuilder interface {
	DirExists(rel string) bool
	BuildKustomization(rel string) ([]byte, error)
}

// Engine computes kustomize render diffs for affected component paths.
type Engine struct {
	head        RepoBuilder
	base        RepoBuilder
	affected    int
	concurrency int
}

// NewEngine creates an Engine with the given head and base repo references.
// Concurrency defaults to runtime.NumCPU() if zero.
func NewEngine(head, base RepoBuilder, affected int) *Engine {
	return &Engine{head: head, base: base, affected: affected, concurrency: runtime.NumCPU()}
}

// DiffResult holds the complete output of a render-diff run.
type DiffResult struct {
	// Diffs contains only components with actual differences.
	Diffs []ComponentDiff
	// TotalAdded is the aggregate lines added across all diffs.
	TotalAdded int
	// TotalRemoved is the aggregate lines removed across all diffs.
	TotalRemoved int
	// Skipped lists affected components that could not be rendered at all,
	// so that output never claims "no differences" for them.
	Skipped []SkippedComponent
}

// Reasons an affected component is skipped instead of rendered.
const (
	SkipReasonUnresolved = "ApplicationSet template could not be resolved; no directory to render"
	SkipReasonNoPath     = "only consumes repository files (e.g. Helm value files); they are not rendered"
	SkipReasonPrefixOnly = "deployed directory only known by its static prefix; it is not rendered"
)

// SkippedComponent is an affected component that render-diff did not render.
type SkippedComponent struct {
	Env detector.Environment
	// Path is the deployed directory or, for a prefix-only entry, its static
	// prefix. It is empty when the template could not be resolved or the
	// component only consumes ExtraFiles.
	Path string
	// ExtraFiles are the individual files the component consumes.
	ExtraFiles []string
	// Reason explains why the component was not rendered.
	Reason string
}

// Label returns a human-readable identifier for the skipped component.
func (s SkippedComponent) Label() string {
	switch {
	case s.Path != "":
		return s.Path
	case len(s.ExtraFiles) > 0:
		return strings.Join(s.ExtraFiles, ", ")
	default:
		return "(unresolved ApplicationSet)"
	}
}

// Run builds each affected component path on both refs in parallel, computes
// unified diffs, and returns only those with actual differences.
func (e *Engine) Run(ctx context.Context, affected map[detector.Environment][]appset.ComponentPath) (*DiffResult, error) {
	// Buffer the channel to hold all possible results so sends never block.
	// RunProgressive sends each diff to the channel; without a reader (Run
	// doesn't need progressive output), an unbuffered or undersized channel
	// would deadlock the errgroup goroutines.
	ch := make(chan ComponentDiff, e.affected)
	return e.RunProgressive(ctx, affected, ch)
}

// RunProgressive is like Run but sends each completed diff to the provided
// channel as it finishes, enabling progressive output. The channel is closed
// when all jobs complete. Returns aggregate stats and any error.
func (e *Engine) RunProgressive(ctx context.Context, affected map[detector.Environment][]appset.ComponentPath, out chan<- ComponentDiff) (*DiffResult, error) {
	defer close(out)

	// Check if there's anything to process.
	if e.affected == 0 {
		return &DiffResult{}, nil
	}

	// Internal channel for collecting results from workers.
	results := make(chan ComponentDiff, e.affected)

	// Collector goroutine: forwards diffs to the progressive output channel
	// and aggregates stats. No mutex needed — only this goroutine writes
	// to result.
	var result DiffResult
	done := make(chan struct{})
	go func() {
		defer close(done)
		for cd := range results {
			out <- cd
			result.Diffs = append(result.Diffs, cd)
			if cd.Error == "" {
				result.TotalAdded += cd.Added
				result.TotalRemoved += cd.Removed
			}
		}
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(e.concurrency)

	// Skipped components are collected by this goroutine only and attached
	// to the result once the workers are done.
	var skipped []SkippedComponent
	skip := func(env detector.Environment, cp appset.ComponentPath, reason string) {
		slog.Warn("skipping affected component, it is not rendered",
			"env", env, "path", cp.Path, "extraFiles", cp.ExtraFiles, "reason", reason)
		skipped = append(skipped, SkippedComponent{Env: env, Path: cp.Path, ExtraFiles: cp.ExtraFiles, Reason: reason})
	}

	for env, paths := range affected {
		for _, cp := range paths {
			// A component without a deployed directory only consumes
			// repository files (e.g. "$values/" Helm value files of a chart
			// source). There is no kustomization to render, so report it
			// instead of building the repository root.
			if cp.MatchAll {
				skip(env, cp, SkipReasonUnresolved)
				continue
			}
			if cp.Path == "" {
				skip(env, cp, SkipReasonNoPath)
				continue
			}
			// A prefix-only Path is the static prefix of an unresolved
			// template, not the deployed kustomization: building it would
			// either fail as a non-kustomization (and be dropped silently)
			// or render an unrelated parent overlay.
			if cp.PrefixOnly {
				skip(env, cp, SkipReasonPrefixOnly)
				continue
			}
			g.Go(func() error {
				if err := ctx.Err(); err != nil {
					return err
				}
				cd := FromComponentPath(cp, env)

				if err := e.buildPair(cd); err != nil {
					cd.Error = err.Error()
					if IsNotKustomizationError(cd.Error) {
						slog.Warn("skipping non-kustomization directory",
							"path", cp.Path, "env", env, "err", err)
						cd.SkipOutput = true
						return nil
					}
					slog.Warn("build error for component",
						"path", cp.Path, "env", env, "err", err)
					results <- *cd
					return nil
				}

				if err := cd.computeDiff(); err != nil {
					return fmt.Errorf("computing diff for %s (%s): %w", cp.Path, env, err)
				}

				if cd.HasDiff() {
					results <- *cd
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		close(results)
		<-done
		return nil, err
	}
	close(results)
	<-done
	sort.SliceStable(skipped, func(i, j int) bool {
		if skipped[i].Env != skipped[j].Env {
			return skipped[i].Env < skipped[j].Env
		}
		return skipped[i].Label() < skipped[j].Label()
	})
	result.Skipped = skipped
	return &result, nil
}

// buildPair builds the kustomization on both refs, populating BaseYAML and HeadYAML.
// Handles new components (no base), removed components (no head), and build errors.
//
// The DirExists checks are necessary because the caller (Run/RunProgressive) passes
// all component paths found in the ApplicationSets, but a path may only exist on one
// ref: new components don't exist on the base, removed components don't exist on HEAD.
// When a directory doesn't exist, kustomize build would fail, so we skip that side
// and let computeDiff treat the missing YAML as empty (showing a full add or remove).
func (e *Engine) buildPair(cd *ComponentDiff) error {
	// Build HEAD — may not exist for removed components.
	if e.head.DirExists(cd.Path) {
		headYAML, err := e.head.BuildKustomization(cd.Path)
		if err != nil {
			return fmt.Errorf("building %s on HEAD: %w", cd.Path, err)
		}
		cd.HeadYAML = headYAML
	}

	// Build base — may not exist for new components.
	if e.base.DirExists(cd.Path) {
		baseYAML, err := e.base.BuildKustomization(cd.Path)
		if err != nil {
			return fmt.Errorf("building %s on base: %w", cd.Path, err)
		}
		cd.BaseYAML = baseYAML
	}

	// If neither side produced output, check whether the directory exists.
	// Empty-base overlays (resources: []) intentionally produce no output;
	// treat them as a no-op instead of an error.
	if cd.HeadYAML == nil && cd.BaseYAML == nil {
		if e.head.DirExists(cd.Path) || e.base.DirExists(cd.Path) {
			return nil
		}
		return fmt.Errorf("component %s does not exist on either ref", cd.Path)
	}

	return nil
}
