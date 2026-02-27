package renderdiff

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

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

	for env, paths := range affected {
		for _, cp := range paths {
			g.Go(func() error {
				if err := ctx.Err(); err != nil {
					return err
				}
				cd := FromComponentPath(cp, env)

				if err := e.buildPair(cd); err != nil {
					slog.Warn("build error for component",
						"path", cp.Path, "env", env, "err", err)
					cd.Error = err.Error()
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

	// If neither side has the directory, nothing to diff.
	if cd.HeadYAML == nil && cd.BaseYAML == nil {
		return fmt.Errorf("component %s does not exist on either ref", cd.Path)
	}

	return nil
}
