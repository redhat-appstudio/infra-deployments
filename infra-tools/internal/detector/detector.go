// Package detector implements the core logic for determining which
// environments and clusters a set of changed files affects.
package detector

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
)

type Environment string

var (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// OverlayEnvironment maps an overlay directory name to an environment name.
// Multiple overlays can map to the same environment.
var OverlayEnvironment = map[string]Environment{
	"development":               Development,
	"konflux-public-staging":    Staging,
	"staging-downstream":        Staging,
	"konflux-public-production": Production,
	"production-downstream":     Production,
}

// kustomizeReservedDirs are directory names that are kustomize conventions and
// should not be treated as cluster names.
var kustomizeReservedDirs = map[string]bool{
	"base":    true,
	"overlay": true,
}

// Result holds the detection output.
type Result struct {
	// AffectedEnvironments is the set of affected environment names.
	AffectedEnvironments map[Environment]bool
	// AffectedClusters is the set of affected cluster names.
	AffectedClusters map[string]bool
	// ChangedFiles is the list of changed files.
	ChangedFiles []string
}

// Detector holds the validated configuration and runs the detection pipeline.
// Create one with NewDetector; all fields are private to ensure the overlay
// validation performed at construction time cannot be bypassed.
type Detector struct {
	head        RepoQuerier // working copy (PR branch)
	base        RepoQuerier // worktree checked out at the target branch
	overlaysDir string      // relative to repo root, e.g. "argo-cd-apps/overlays"

	// overlayEnvs is the validated overlay-name → environment mapping for the
	// union of overlay directories found on both refs.
	overlayEnvs map[string]Environment
}

// NewDetector creates a Detector and validates that every overlay directory
// present on either ref is known to OverlayEnvironment.
func NewDetector(head, base RepoQuerier, overlaysDir string) (*Detector, error) {
	// Collect overlay names from both refs (union).
	names := make(map[string]bool)

	headNames, err := head.ListSubDirs(overlaysDir)
	if err != nil {
		return nil, fmt.Errorf("listing overlay dirs on HEAD: %w", err)
	}
	for _, n := range headNames {
		names[n] = true
	}

	// Base overlay dir may not exist yet (e.g. first PR that adds overlays).
	baseNames, _ := base.ListSubDirs(overlaysDir)
	for _, n := range baseNames {
		names[n] = true
	}

	// Validate every name.
	envs := make(map[string]Environment, len(names))
	for name := range names {
		env, ok := OverlayEnvironment[name]
		if !ok {
			return nil, fmt.Errorf("unknown overlay %q in %s: not present in OverlayEnvironment map", name, overlaysDir)
		}
		envs[name] = env
	}

	return &Detector{
		head:        head,
		base:        base,
		overlaysDir: overlaysDir,
		overlayEnvs: envs,
	}, nil
}

// overlayBuild holds the kustomize build output for one overlay on both refs.
type overlayBuild struct {
	name     string
	env      Environment
	headYAML []byte
	baseYAML []byte
}

// Detect runs the full detection pipeline:
//  1. Build ArgoCD overlays on both refs
//  2. Detect overlay diffs (ArgoCD config changes)
//  3. Extract component paths from ApplicationSets
//  4. Resolve dependency trees for component paths
//  5. Match changed files against resolved dependencies
//  6. Apply static rules for app-of-app-sets
func (d *Detector) Detect(changedFiles []string) (*Result, error) {
	result := &Result{
		AffectedEnvironments: make(map[Environment]bool),
		AffectedClusters:     make(map[string]bool),
		ChangedFiles:         changedFiles,
	}

	// Phase 1: Build ArgoCD ApplicationSet overlays on HEAD and base-ref
	builds, err := d.buildAppSetOverlays()
	if err != nil {
		return nil, err
	}

	// Check for overlays removed in HEAD
	d.detectRemovedAppSetOverlays(result)

	// Phase 2: Detect overlay diffs (ArgoCD config changes)
	detectOverlayDiffs(builds, result)

	// Phase 3: Extract component paths from ApplicationSets
	envPaths, allClusters, err := extractPathsFromOverlays(builds)
	if err != nil {
		return nil, err
	}

	// Phase 4: Resolve dependency trees for component paths
	resolved := d.resolveComponentDeps(envPaths)

	// Phase 5: Match changed files against resolved dependencies
	matchChangedFiles(changedFiles, resolved, allClusters, result)

	// Phase 6: Static rules (app-of-app-sets)
	applyStaticRules(changedFiles, result)

	return result, nil
}

// buildAppSetOverlays builds the kustomize overlays that produce ArgoCD
// ApplicationSet manifests.  It runs kustomize build on every overlay
// directory for both HEAD and the base-ref in parallel using an errgroup,
// returning a build result per overlay.
func (d *Detector) buildAppSetOverlays() ([]overlayBuild, error) {
	overlayNames, err := d.head.ListSubDirs(d.overlaysDir)
	if err != nil {
		return nil, fmt.Errorf("listing overlay dirs: %w", err)
	}

	var (
		results = make(chan overlayBuild, len(overlayNames))
		g       errgroup.Group
	)

	for _, overlayName := range overlayNames {
		g.Go(func() error {
			env := d.overlayEnvs[overlayName] // pre-validated by NewDetector
			overlayRel := filepath.Join(d.overlaysDir, overlayName)

			headYAML, err := d.head.BuildKustomization(overlayRel)
			if err != nil {
				return fmt.Errorf("building overlay %s on HEAD: %w", overlayName, err)
			}

			// The overlay may not exist on the base-ref (new overlay in HEAD).
			var baseYAML []byte
			if d.base.DirExists(overlayRel) {
				baseYAML, err = d.base.BuildKustomization(overlayRel)
				if err != nil {
					return fmt.Errorf("building overlay %s on base-ref: %w", overlayName, err)
				}
			} else {
				slog.Info("Overlay does not exist on base-ref (new overlay), treating as empty", "overlay", overlayName)
			}

			results <- overlayBuild{
				name:     overlayName,
				env:      env,
				headYAML: headYAML,
				baseYAML: baseYAML,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(results)

	builds := make([]overlayBuild, 0, len(overlayNames))
	for ob := range results {
		builds = append(builds, ob)
	}
	return builds, nil
}

// detectRemovedAppSetOverlays marks environments as affected when an overlay
// exists on the base-ref but was removed in HEAD.
func (d *Detector) detectRemovedAppSetOverlays(result *Result) {
	baseOverlayNames, err := d.base.ListSubDirs(d.overlaysDir)
	if err != nil {
		return
	}
	for _, overlayName := range baseOverlayNames {
		env := d.overlayEnvs[overlayName] // pre-validated by NewDetector
		overlayRel := filepath.Join(d.overlaysDir, overlayName)
		if d.head.DirExists(overlayRel) {
			continue // still exists on HEAD, handled by buildAppSetOverlays
		}
		slog.Info("Overlay removed in HEAD", "overlay", overlayName, "env", env)
		result.AffectedEnvironments[env] = true
	}
}

// detectOverlayDiffs marks environments as affected when the rendered overlay
// YAML differs between HEAD and the base-ref (i.e. the ArgoCD config changed).
func detectOverlayDiffs(builds []overlayBuild, result *Result) {
	for _, ob := range builds {
		if string(ob.headYAML) != string(ob.baseYAML) {
			result.AffectedEnvironments[ob.env] = true
			slog.Info("Overlay diff detected", "overlay", ob.name, "env", ob.env)
		}
	}
}

// extractPathsFromOverlays parses ApplicationSets from the HEAD builds and
// returns per-environment component paths and a mapping from cluster name to
// the component paths deployed on that cluster.
func extractPathsFromOverlays(builds []overlayBuild) (map[Environment][]appset.ComponentPath, map[string][]string, error) {
	envPaths := make(map[Environment][]appset.ComponentPath)
	allClusters := make(map[string][]string)

	for _, ob := range builds {
		parsed, err := appset.ParseApplicationSets(ob.headYAML)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing ApplicationSets from overlay %s: %w", ob.name, err)
		}

		// Every path inherits the environment from the overlay that produced it.
		envPaths[ob.env] = append(envPaths[ob.env], parsed.Paths...)
		for cluster, cpaths := range parsed.Clusters {
			allClusters[cluster] = append(allClusters[cluster], cpaths...)
		}
	}
	return envPaths, allClusters, nil
}

// componentDeps holds the resolved dependencies for a single component path.
type componentDeps struct {
	cp   appset.ComponentPath
	env  Environment
	deps map[string]bool // nil means no kustomization.yaml; use prefix matching
}

// resolveComponentDeps walks every component path extracted from ApplicationSets
// and resolves its kustomize dependency tree.  When no kustomization.yaml exists,
// deps is left nil to signal that prefix matching should be used instead.
func (d *Detector) resolveComponentDeps(envPaths map[Environment][]appset.ComponentPath) []componentDeps {
	type envPath struct {
		env  Environment
		path string
	}
	seen := make(map[envPath]bool)

	var resolved []componentDeps
	for env, paths := range envPaths {
		for _, cp := range paths {
			key := envPath{env, cp.Path}
			if seen[key] {
				continue
			}
			seen[key] = true

			// The path may not exist on HEAD when a component was deleted,
			// the ApplicationSet reference is stale, or a templated path
			// resolves to a directory that hasn't been created yet.
			if !d.head.DirExists(cp.Path) {
				continue
			}

			deps, err := d.head.ResolveDeps(cp.Path)
			if err != nil {
				// No kustomization.yaml — the ApplicationSet may deploy the
				// directory as plain YAML manifests.
				slog.Warn("no dep tree, will use prefix match", "path", cp.Path, "err", err)
				resolved = append(resolved, componentDeps{cp: cp, env: env, deps: nil})
				continue
			}
			slog.Debug("resolved dependency tree", "path", cp.Path, "env", env, "deps", deps)
			resolved = append(resolved, componentDeps{cp: cp, env: env, deps: deps})
		}
	}
	return resolved
}

// matchChangedFiles checks each resolved component against the set of changed
// files.  For components with a dependency tree it uses an exact match; for
// components without a kustomization.yaml it falls back to prefix matching.
func matchChangedFiles(changedFiles []string, resolved []componentDeps, allClusters map[string][]string, result *Result) {
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	for _, cd := range resolved {
		var matched bool
		if cd.deps != nil {
			matched = matchDepTree(changedSet, cd.deps)
		} else {
			matched = matchByPrefix(changedSet, cd.cp.Path)
		}

		if matched {
			result.AffectedEnvironments[cd.env] = true
			matchClusters(cd.cp, allClusters, result)
			slog.Info("Changed files match component", "path", cd.cp.Path, "env", cd.env)
		}
	}
}

// applyStaticRules handles paths with hard-coded environment mappings that
// aren't covered by overlay diffs or dependency trees.
func applyStaticRules(changedFiles []string, result *Result) {
	for _, f := range changedFiles {
		// Any change under app-of-app-sets affects all environments because
		// these are the root ArgoCD Applications that deploy every overlay.
		if strings.HasPrefix(f, "argo-cd-apps/app-of-app-sets/") {
			for _, env := range OverlayEnvironment {
				result.AffectedEnvironments[env] = true
			}
			return
		}
	}
}

// --- helper functions --------------------------------------------------------

// matchByPrefix returns true if any file in changedSet lives under pathPrefix.
// Used as a fallback when a directory has no kustomization.yaml but might be
// deployed by ArgoCD as plain YAML manifests.
func matchByPrefix(changedSet map[string]bool, pathPrefix string) bool {
	prefix := pathPrefix
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for f := range changedSet {
		if strings.HasPrefix(f, prefix) || f == pathPrefix {
			return true
		}
	}
	return false
}

// matchDepTree returns true if any key in changedSet is present in deps.
func matchDepTree(changedSet, deps map[string]bool) bool {
	for f := range changedSet {
		if deps[f] {
			return true
		}
	}
	return false
}

// matchClusters adds cluster labels when a component path corresponds to a
// specific cluster.
func matchClusters(cp appset.ComponentPath, allClusters map[string][]string, result *Result) {
	if cp.ClusterDir != "" && !kustomizeReservedDirs[cp.ClusterDir] {
		result.AffectedClusters[cp.ClusterDir] = true
		return
	}
	// ClusterDir is empty or a reserved name like "base" — check which real
	// clusters map to paths under this component.
	for cluster, paths := range allClusters {
		if kustomizeReservedDirs[cluster] {
			continue
		}
		for _, p := range paths {
			if p == cp.Path || strings.HasPrefix(p, cp.Path+"/") {
				result.AffectedClusters[cluster] = true
			}
		}
	}
}

