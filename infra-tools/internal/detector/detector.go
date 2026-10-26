// Package detector implements the core logic for determining which
// environments and clusters a set of changed files affects.
package detector

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/deptree"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/logging"
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
	"development-operator":      Development,
	"konflux-public-staging":    Staging,
	"staging-downstream":        Staging,
	"konflux-public-production": Production,
	"production-downstream":     Production,
	"rd-dev":                    Development,
	"rd-staging":                Staging,
	"rd-production":             Production,
}

// kustomizeReservedDirs are directory names that are kustomize conventions and
// should not be treated as cluster names. empty-base is the placeholder
// clusterDir on ring-based ApplicationSets (not a real cluster).
var kustomizeReservedDirs = map[string]bool{
	"base":       true,
	"overlay":    true,
	"empty-base": true,
}

// Result holds the detection output.
type Result struct {
	// AffectedEnvironments is the set of affected environment names.
	AffectedEnvironments map[Environment]bool
	// AffectedClusters is the set of affected cluster names.
	AffectedClusters map[string]bool
	// ChangedFiles is the list of changed files.
	ChangedFiles []string
	// UnreferencedFiles are changed files under components/ or configs/ that
	// no ApplicationSet references, so they will not be deployed by anything.
	// They are reported as a warning instead of being treated as "no impact".
	// A MatchAll entry matches every file, so it suppresses this list.
	UnreferencedFiles []string
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

// AffectedComponents builds ArgoCD overlays, parses ApplicationSets, resolves
// kustomize dependency trees, and matches changedFiles to determine which
// component paths are affected, grouped by environment.  Unlike Detect, it does
// not diff overlays or apply static rules — it only returns the component-level
// mapping needed by tools like render-diff.
//
// Only entries with a concrete deployed directory are returned. MatchAll,
// PrefixOnly and value-file-only entries still affect Detect's environment and
// cluster labels, but they have no kustomization for render-diff to build.
func (d *Detector) AffectedComponents(changedFiles []string) (map[Environment][]appset.ComponentPath, error) {
	// Phase 1: Build ArgoCD ApplicationSet overlays on HEAD and base-ref
	builds, err := d.buildAppSetOverlays()
	if err != nil {
		return nil, err
	}

	// Phase 3: Extract component paths from ApplicationSets
	envPaths, allClusters, _, err := extractPathsFromOverlays(builds)
	if err != nil {
		return nil, err
	}

	// Phase 4: Resolve dependency trees for component paths
	resolved := d.resolveComponentDeps(envPaths)

	// Phase 5: Match changed files against resolved dependencies
	return matchAffectedComponents(changedFiles, resolved, allClusters), nil
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
	envPaths, allClusters, headKeys, err := extractPathsFromOverlays(builds)
	if err != nil {
		return nil, err
	}

	// Phase 4: Resolve dependency trees for component paths
	resolved := d.resolveComponentDepsWithHead(envPaths, headKeys)

	// Phase 5: Match changed files against resolved dependencies. A file that
	// HEAD still contains but only a removed (base-only) ApplicationSet
	// references is dead after merge, so the unreferenced check uses HEAD
	// paths alone.
	matchChangedFiles(changedFiles, resolved, allClusters, result)
	result.UnreferencedFiles = d.unreferencedFiles(changedFiles, matchedHeadFiles(changedFiles, resolved))

	// Phase 6: Static rules (app-of-app-sets)
	applyStaticRules(changedFiles, result)

	return result, nil
}

// buildAppSetOverlays builds the kustomize overlays that produce ArgoCD
// ApplicationSet manifests.  It runs kustomize build on every overlay
// directory for both HEAD and the base-ref in parallel using an errgroup,
// returning a build result per overlay.
//
// Overlay directories are taken from both refs: an overlay that a PR deletes
// exists only on the base-ref, and its ApplicationSets still reference the
// components whose removal must be visible to render-diff.
func (d *Detector) buildAppSetOverlays() ([]overlayBuild, error) {
	headNames, err := d.head.ListSubDirs(d.overlaysDir)
	if err != nil {
		return nil, fmt.Errorf("listing overlay dirs: %w", err)
	}
	baseNames, _ := d.base.ListSubDirs(d.overlaysDir) // no overlays dir yet on base is fine

	headExists := make(map[string]bool, len(headNames))
	for _, name := range headNames {
		headExists[name] = true
	}

	overlayNames := make([]string, 0, len(headNames)+len(baseNames))
	seen := make(map[string]bool, len(headNames)+len(baseNames))
	for _, name := range append(headNames, baseNames...) {
		if !seen[name] {
			seen[name] = true
			overlayNames = append(overlayNames, name)
		}
	}
	sort.Strings(overlayNames)

	var (
		results = make(chan overlayBuild, len(overlayNames))
		g       errgroup.Group
	)

	for _, overlayName := range overlayNames {
		g.Go(func() error {
			env := d.overlayEnvs[overlayName] // pre-validated by NewDetector
			overlayRel := filepath.Join(d.overlaysDir, overlayName)

			// The overlay may not exist on one of the refs: an added overlay
			// has no base, a removed one has no HEAD.
			var headYAML []byte
			if headExists[overlayName] {
				var err error
				headYAML, err = d.head.BuildKustomization(overlayRel)
				if err != nil {
					return fmt.Errorf("building overlay %s on HEAD: %w", overlayName, err)
				}
			} else {
				slog.Info("Overlay does not exist on HEAD (removed overlay), treating as empty", "overlay", logging.Sanitize(overlayName))
			}

			var baseYAML []byte
			if d.base.DirExists(overlayRel) {
				var err error
				baseYAML, err = d.base.BuildKustomization(overlayRel)
				if err != nil {
					return fmt.Errorf("building overlay %s on base-ref: %w", overlayName, err)
				}
			} else {
				slog.Info("Overlay does not exist on base-ref (new overlay), treating as empty", "overlay", logging.Sanitize(overlayName))
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
		slog.Info("Overlay removed in HEAD", "overlay", logging.Sanitize(overlayName), "env", env)
		result.AffectedEnvironments[env] = true
	}
}

// detectOverlayDiffs marks environments as affected when the rendered overlay
// YAML differs between HEAD and the base-ref (i.e. the ArgoCD config changed).
// When an ApplicationSet has an explicit environment in its generator, that
// environment is used instead of the overlay's environment. This prevents base
// changes (included by all overlays) from spuriously marking every environment.
func detectOverlayDiffs(builds []overlayBuild, result *Result) {
	for _, ob := range builds {
		if string(ob.headYAML) == string(ob.baseYAML) {
			continue
		}
		slog.Info("Overlay diff detected", "overlay", logging.Sanitize(ob.name))

		headSets, headErr := appset.AppSetsByName(ob.headYAML)
		baseSets, baseErr := appset.AppSetsByName(ob.baseYAML)
		if headErr != nil || baseErr != nil {
			// Can't parse ApplicationSets; conservatively mark overlay environment.
			slog.Warn("Failed to parse ApplicationSets, falling back to overlay env",
				"overlay", logging.Sanitize(ob.name), "headErr", sanitizedError(headErr), "baseErr", sanitizedError(baseErr))
			result.AffectedEnvironments[ob.env] = true
			continue
		}

		attributed := false
		for name, headDoc := range headSets {
			if baseDoc, existed := baseSets[name]; existed {
				if reflect.DeepEqual(headDoc, baseDoc) {
					continue // this ApplicationSet is unchanged
				}
			}
			env := appSetEnv(headDoc, ob.env)
			result.AffectedEnvironments[env] = true
			slog.Info("AppSet added/modified", "overlay", logging.Sanitize(ob.name), "appset", logging.Sanitize(name), "env", env)
			attributed = true
		}
		for name, baseDoc := range baseSets {
			if _, exists := headSets[name]; !exists {
				env := appSetEnv(baseDoc, ob.env)
				result.AffectedEnvironments[env] = true
				slog.Info("AppSet removed", "overlay", logging.Sanitize(ob.name), "appset", logging.Sanitize(name), "env", env)
				attributed = true
			}
		}
		if !attributed {
			// YAML changed but no ApplicationSet changes found (e.g. namespace patch).
			result.AffectedEnvironments[ob.env] = true
		}
	}
}

// environmentFor returns the explicit environment declared by an ApplicationSet,
// or fallback when none (or an unknown value) is set.
func environmentFor(explicit string, fallback Environment) Environment {
	switch e := Environment(explicit); e {
	case Development, Staging, Production:
		return e
	}
	return fallback
}

// sanitizedError renders an error for a log line without letting PR-controlled
// content break out of it. A nil error renders empty.
func sanitizedError(err error) string {
	if err == nil {
		return ""
	}
	return logging.Sanitize(err.Error())
}

// appSetEnv extracts the environment from an ApplicationSet's generator. Falls
// back to overlayEnv when no explicit environment is set.
func appSetEnv(doc map[string]interface{}, overlayEnv Environment) Environment {
	if envStr, ok := appset.EnvironmentFromAppSet(doc); ok {
		return environmentFor(envStr, overlayEnv)
	}
	return overlayEnv
}

// extractPathsFromOverlays parses ApplicationSets from both the HEAD and the
// base builds and returns per-environment component paths, a mapping from
// cluster name to the component paths deployed on that cluster, and the set of
// path keys that exist on HEAD (for telling base-only references apart).
//
// Both refs are needed: a component that this PR removes is only referenced on
// the base ref, and a component this PR adds only on HEAD. Parsing HEAD alone
// would hide removals from the diff.
func extractPathsFromOverlays(builds []overlayBuild) (map[Environment][]appset.ComponentPath, map[string][]string, map[string]bool, error) {
	envPaths := make(map[Environment][]appset.ComponentPath)
	allClusters := make(map[string][]string)
	headKeys := make(map[string]bool)
	seen := make(map[string]bool)

	for _, ob := range builds {
		for i, rendered := range [][]byte{ob.headYAML, ob.baseYAML} {
			if len(rendered) == 0 {
				continue
			}
			parsed, err := appset.ParseApplicationSets(rendered)
			if err != nil {
				if i == 0 {
					// HEAD must be parseable: without it nothing can be detected.
					return nil, nil, nil, fmt.Errorf("parsing ApplicationSets from overlay %s: %w", ob.name, err)
				}
				// The base render is best-effort context (it only adds removed
				// components); a parse failure must not fail the whole run.
				slog.Warn("failed to parse base-ref ApplicationSets, continuing with HEAD only",
					"overlay", logging.Sanitize(ob.name), "err", sanitizedError(err))
				continue
			}

			for _, cp := range parsed.Paths {
				// The overlay environment is the deployment target: a
				// production overlay that includes a base ApplicationSet
				// unpatched still deploys it to production, so never narrow
				// attribution below the overlay. The declared environment is
				// usually a path-template variable; record it as well so a path
				// rendered by several overlays keeps its own environment too.
				envs := []Environment{ob.env}
				if declared := environmentFor(cp.Environment, ob.env); declared != ob.env {
					envs = append(envs, declared)
				}
				for _, env := range envs {
					key := componentPathKey(env, cp)
					if i == 0 {
						headKeys[key] = true
					}
					if seen[key] {
						continue
					}
					seen[key] = true
					envPaths[env] = append(envPaths[env], cp)
				}
			}
			for cluster, cpaths := range parsed.Clusters {
				for _, p := range cpaths {
					key := "cluster|" + cluster + "|" + p
					if seen[key] {
						continue
					}
					seen[key] = true
					allClusters[cluster] = append(allClusters[cluster], p)
				}
			}
		}
	}
	return envPaths, allClusters, headKeys, nil
}

// componentPathKey identifies a resolved path for deduplication. The
// environment is part of the key: the same path can be deployed by overlays in
// different environments.
func componentPathKey(env Environment, cp appset.ComponentPath) string {
	return string(env) + "|" + cp.Path + "|" + cp.ClusterDir + "|" +
		strings.Join(cp.ExtraFiles, ",") + "|" +
		strconv.FormatBool(cp.PrefixOnly) + "|" + strconv.FormatBool(cp.MatchAll)
}

// componentDeps holds the resolved dependencies for a single component path.
type componentDeps struct {
	cp                 appset.ComponentPath
	env                Environment
	deps               map[string]bool // nil means no kustomization.yaml; use prefix matching
	prefixFallback     bool            // at least one ref exists without a complete tree
	headDeps           map[string]bool // HEAD-only dependency tree for the unreferenced check
	headPrefixFallback bool            // HEAD alone exists without a complete tree
	fromHead           bool            // the component path exists on HEAD
}

// resolveComponentDeps walks every component path extracted from ApplicationSets
// and resolves its kustomize dependency tree.  When no kustomization.yaml exists,
// deps is left nil to signal that prefix matching should be used instead.
func (d *Detector) resolveComponentDeps(envPaths map[Environment][]appset.ComponentPath) []componentDeps {
	return d.resolveComponentDepsWithHead(envPaths, nil)
}

// resolveComponentDepsWithHead is resolveComponentDeps, additionally marking
// which component paths exist on HEAD. headKeys is the key set returned by
// extractPathsFromOverlays; nil marks nothing as HEAD-derived.
func (d *Detector) resolveComponentDepsWithHead(envPaths map[Environment][]appset.ComponentPath, headKeys map[string]bool) []componentDeps {
	seen := make(map[string]bool)

	var resolved []componentDeps
	for env, paths := range envPaths {
		for _, cp := range paths {
			// The key must include ClusterDir, ExtraFiles and the fallback flags:
			// several list elements can share the same deployed directory while
			// targeting different clusters, contributing different value files,
			// or using a different matching strategy. Collapsing them would lose
			// cluster attribution or narrow a prefix-only match.
			key := componentPathKey(env, cp)
			if seen[key] {
				continue
			}
			seen[key] = true
			fromHead := headKeys[key]

			// An unresolvable template with no static prefix: nothing to
			// resolve, any change affects the environment (see matches).
			if cp.MatchAll {
				resolved = append(resolved, componentDeps{cp: cp, env: env, fromHead: fromHead})
				continue
			}

			// A component path may only carry individual files (e.g. a Helm
			// values file from a multi-source template); there is nothing to
			// resolve, the files are matched exactly.
			if cp.Path == "" {
				if len(cp.ExtraFiles) > 0 {
					resolved = append(resolved, componentDeps{cp: cp, env: env, fromHead: fromHead})
				}
				continue
			}

			// A prefix-only path is an unresolved template: the deployed
			// directory is one of those under the prefix (which may not even
			// exist), so match by prefix and also by the dependency trees of
			// every kustomization under it, which reach shared bases outside
			// the prefix (e.g. ../../base/member).
			if cp.PrefixOnly {
				resolved = append(resolved, componentDeps{
					cp: cp, env: env,
					deps:     d.prefixDeps(cp.Path),
					headDeps: prefixDepsFor(d.head, cp.Path),
					fromHead: fromHead,
				})
				continue
			}

			// The path may exist on neither ref when the ApplicationSet
			// reference is stale or a templated path resolves to a directory
			// that hasn't been created yet. It also happens for a source whose
			// path lives in another repository while its "$values/" files live
			// here: keep those files, they are still matched exactly.
			headExists := d.head.DirExists(cp.Path)
			baseExists := d.base.DirExists(cp.Path)
			if !headExists && !baseExists {
				if len(cp.ExtraFiles) > 0 {
					filesOnly := cp
					filesOnly.Path = ""
					resolved = append(resolved, componentDeps{cp: filesOnly, env: env, fromHead: fromHead})
				}
				continue
			}

			deps, headDeps, prefixFallback, headPrefixFallback := d.resolvePathDeps(cp.Path, headExists, baseExists)
			switch {
			case deps == nil:
				// No kustomization.yaml — the ApplicationSet may deploy the
				// directory as plain YAML manifests.
				slog.Warn("no dep tree, will use prefix match", "path", logging.Sanitize(cp.Path), "env", env)
			case prefixFallback:
				slog.Debug("partial dep tree, will also use prefix match", "path", logging.Sanitize(cp.Path), "env", env, "deps", len(deps))
			default:
				slog.Debug("resolved dependency tree", "path", logging.Sanitize(cp.Path), "env", env, "deps", deps)
			}
			resolved = append(resolved, componentDeps{
				cp: cp, env: env, deps: deps, prefixFallback: prefixFallback,
				headDeps: headDeps, headPrefixFallback: headPrefixFallback, fromHead: fromHead,
			})
		}
	}
	return resolved
}

// prefixDeps returns the union of the dependency trees of every kustomization
// in or below prefix on HEAD and base, or nil when neither has one.
func (d *Detector) prefixDeps(prefix string) map[string]bool {
	deps := prefixDepsFor(d.head, prefix)
	baseDeps := prefixDepsFor(d.base, prefix)
	if deps == nil {
		deps = baseDeps
	} else {
		maps.Copy(deps, baseDeps)
	}
	slog.Debug("resolved dependency trees under prefix", "prefix", logging.Sanitize(prefix), "deps", len(deps))
	return deps
}

// prefixDepsFor returns the union of dependency trees in or below prefix on one
// ref, or nil when the ref has no contained prefix or no kustomization there.
func prefixDepsFor(repo RepoQuerier, prefix string) map[string]bool {
	if repo == nil {
		return nil
	}
	// The prefix comes from ApplicationSet YAML. The parser only emits
	// repository-relative prefixes, but never walk the repository root, an
	// absolute path or anything escaping the checkout, lexically or through
	// a symlink in the prefix or its ancestors. (Symlinks below the prefix
	// are not followed: ListSubDirs only returns real directories.)
	if prefix == "." || !filepath.IsLocal(prefix) || !repo.ContainedDirExists(prefix) {
		return nil
	}
	var deps map[string]bool
	var walk func(dir string)
	walk = func(dir string) {
		tree, err := repo.ResolveDeps(dir)
		if len(tree) > 0 {
			if deps == nil {
				deps = make(map[string]bool, len(tree))
			}
			maps.Copy(deps, tree)
		}
		if err != nil && !errors.Is(err, deptree.ErrIncomplete) {
			// No kustomization here; still walk children that may have one.
			slog.Debug("no dependency tree for prefix directory", "dir", logging.Sanitize(dir), "err", sanitizedError(err))
		}
		subDirs, err := repo.ListSubDirs(dir)
		if err != nil {
			return
		}
		for _, sub := range subDirs {
			if sub == ".git" {
				continue
			}
			walk(filepath.Join(dir, sub))
		}
	}
	walk(prefix)
	return deps
}

// matchesFile reports whether a single changed file affects this component.
// ExtraFiles (e.g. Helm values files) are matched exactly; dependency trees are
// matched exactly; anything else is matched by path prefix. A prefix-only path
// matches by prefix or by the dependency trees found under the prefix. A
// MatchAll entry matches any change.
func (cd componentDeps) matchesFile(file string) bool {
	if cd.cp.MatchAll {
		return true
	}
	for _, f := range cd.cp.ExtraFiles {
		if f == file {
			return true
		}
	}
	if cd.cp.Path == "" {
		return false
	}
	if cd.cp.PrefixOnly {
		return matchesPrefix(file, cd.cp.Path) || cd.deps[file]
	}
	if cd.prefixFallback {
		return cd.deps[file] || matchesPrefix(file, cd.cp.Path)
	}
	if cd.deps != nil {
		return cd.deps[file]
	}
	return matchesPrefix(file, cd.cp.Path)
}

// matches reports whether any changed file affects this component. When matched
// is non-nil, every matching file is recorded in it.
func (cd componentDeps) matches(changedSet map[string]bool, matched map[string]bool) bool {
	found := false
	for f := range changedSet {
		if cd.matchesFile(f) {
			found = true
			if matched != nil {
				matched[f] = true
			}
		}
	}
	return found
}

// resolvePathDeps returns the union of the kustomize dependency trees of rel on
// the refs where it exists, plus the HEAD-only view.  Union matters at the
// edges of a PR: a kustomization that gains a dependency in HEAD must not hide
// a removal relative to base, and vice versa. A nil result means neither ref
// has a kustomization there.
//
// prefixFallback is true when at least one ref has the directory but no
// complete dependency tree (plain manifests, or an unreadable generator/chart).
// headPrefixFallback is the same signal for HEAD alone: callers that ask
// whether a file is still referenced on HEAD must not use base-only
// dependencies, and must use HEAD's own prefix-fallback state.
func (d *Detector) resolvePathDeps(rel string, headExists, baseExists bool) (deps, headDeps map[string]bool, prefixFallback, headPrefixFallback bool) {
	merge := func(dst *map[string]bool, m map[string]bool) {
		if len(m) == 0 {
			return
		}
		if *dst == nil {
			*dst = make(map[string]bool, len(m))
		}
		for k := range m {
			(*dst)[k] = true
		}
	}
	resolve := func(repo RepoQuerier, exists bool, name string, head bool) {
		if !exists {
			return
		}
		m, err := repo.ResolveDeps(rel)
		switch {
		case err == nil:
			merge(&deps, m)
			if head {
				merge(&headDeps, m)
			}
		case errors.Is(err, deptree.ErrIncomplete):
			prefixFallback = true
			merge(&deps, m)
			if head {
				headPrefixFallback = true
				merge(&headDeps, m)
			}
			slog.Warn("partial dependency tree, also using prefix match",
				"path", logging.Sanitize(rel), "ref", name, "err", sanitizedError(err))
		default:
			prefixFallback = true
			if head {
				headPrefixFallback = true
			}
			slog.Debug("no dependency tree, also using prefix match",
				"path", logging.Sanitize(rel), "ref", name, "err", sanitizedError(err))
		}
	}

	resolve(d.head, headExists, "HEAD", true)
	resolve(d.base, baseExists, "base", false)
	if deps == nil {
		prefixFallback = true
	}
	if headExists && headDeps == nil {
		headPrefixFallback = true
	}
	return deps, headDeps, prefixFallback, headPrefixFallback
}

// matchChangedFiles checks each resolved component against the set of changed
// files.  For components with a dependency tree it uses an exact match; for
// components without a kustomization.yaml it falls back to prefix matching.
// It returns the changed files that matched at least one component.
func matchChangedFiles(changedFiles []string, resolved []componentDeps, allClusters map[string][]string, result *Result) map[string]bool {
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
	}
	matched := make(map[string]bool)

	for _, cd := range resolved {
		if cd.matches(changedSet, matched) {
			if cd.cp.MatchAll {
				slog.Warn("An ApplicationSet template could not be resolved, conservatively marking its environment as affected", "env", cd.env)
			}
			result.AffectedEnvironments[cd.env] = true
			matchClusters(cd.cp, allClusters, result)
			slog.Info("Changed files match component", "path", logging.Sanitize(cd.cp.Path), "env", cd.env)
		}
	}
	return matched
}

// matchedHeadFiles records changed files that a component path existing on HEAD
// references. Base-only paths deliberately do not count: a file HEAD still
// contains but no current ApplicationSet deploys is dead after merge, which is
// exactly what UnreferencedFiles reports.
func matchedHeadFiles(changedFiles []string, resolved []componentDeps) map[string]bool {
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	matched := make(map[string]bool)
	for _, cd := range resolved {
		if !cd.fromHead {
			continue
		}
		head := cd
		head.deps = cd.headDeps
		head.prefixFallback = cd.headPrefixFallback
		head.matches(changedSet, matched)
	}
	return matched
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

// matchAffectedComponents returns the renderable component paths affected by
// changedFiles, grouped by environment. It uses the same matching logic as
// matchChangedFiles (dep tree or prefix fallback) but returns the matched
// ComponentPath values instead of mutating a Result.
//
// Entries without a deployed directory are omitted: render-diff can only build
// a kustomization, so MatchAll, PrefixOnly and value-file-only matches would
// otherwise be built as the repository root or as an unrelated prefix. Paths
// are deduplicated per environment and cluster because render-diff builds each
// one once.
func matchAffectedComponents(changedFiles []string, resolved []componentDeps, allClusters map[string][]string) map[Environment][]appset.ComponentPath {
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	affected := make(map[Environment][]appset.ComponentPath)
	seen := make(map[string]bool)
	for _, cd := range resolved {
		if !cd.matches(changedSet, nil) {
			continue
		}
		if cd.cp.MatchAll || cd.cp.Path == "" || cd.cp.PrefixOnly {
			slog.Warn("affected component has no renderable directory; render-diff will not show a diff",
				"path", logging.Sanitize(cd.cp.Path), "extraFiles", cd.cp.ExtraFiles, "env", cd.env)
			continue
		}
		key := string(cd.env) + "|" + cd.cp.Path + "|" + cd.cp.ClusterDir
		if seen[key] {
			continue
		}
		seen[key] = true
		affected[cd.env] = append(affected[cd.env], cd.cp)
		slog.Info("Changed files match component", "path", logging.Sanitize(cd.cp.Path), "env", cd.env)
	}
	return affected
}

// --- helper functions --------------------------------------------------------

// matchByPrefix returns true if any file in changedSet lives under pathPrefix.
// Used as a fallback when a directory has no kustomization.yaml but might be
// deployed by ArgoCD as plain YAML manifests.
func matchByPrefix(changedSet map[string]bool, pathPrefix string) bool {
	for f := range changedSet {
		if matchesPrefix(f, pathPrefix) {
			return true
		}
	}
	return false
}

// matchesPrefix reports whether file is pathPrefix itself or lives under it.
func matchesPrefix(file, pathPrefix string) bool {
	if file == pathPrefix {
		return true
	}
	prefix := pathPrefix
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.HasPrefix(file, prefix)
}

// nonDeployableFiles are file names that live next to deployable manifests but
// are never consumed by kustomize. Reporting them as unreferenced would be
// technically true and practically noise.
var nonDeployableFiles = map[string]bool{
	"README.md":      true,
	"OWNERS":         true,
	"OWNERS_ALIASES": true,
	".gitignore":     true,
}

// nonDeployablePaths is an escape hatch for individual files consumed by other
// tooling (CI checks, schema validation) rather than by kustomize. Prefer a
// basename or path-segment rule when one applies.
var nonDeployablePaths = map[string]bool{
	"components/konflux-info/banner-schema.json": true,
	"components/kubearchive/upgrade.sh":          true,
}

// nonDeployablePathSegments are directories that hold tests or CI tooling for a
// component, not manifests consumed by kustomize.
var nonDeployablePathSegments = []string{
	"/.chainsaw-test/",
	"/ci/",
}

// nonDeployablePathPrefixes are component trees applied outside ArgoCD, for
// example by the cluster bootstrap scripts.
var nonDeployablePathPrefixes = []string{
	"components/openshift-gitops/",
}

// isDeployableFile reports whether a changed file may be consumed by kustomize.
func isDeployableFile(file string) bool {
	if nonDeployableFiles[filepath.Base(file)] || nonDeployablePaths[file] {
		return false
	}
	for _, segment := range nonDeployablePathSegments {
		if strings.Contains(file, segment) {
			return false
		}
	}
	for _, prefix := range nonDeployablePathPrefixes {
		if strings.HasPrefix(file, prefix) {
			return false
		}
	}
	switch strings.ToLower(filepath.Ext(file)) {
	case ".md", ".png", ".jpg", ".jpeg", ".svg", ".gif":
		return false
	default:
		return true
	}
}

// isRingBaseFile reports whether file is a Tier 1 component base file whose
// component uses the ring layout. Kargo copies that base into each ring's
// base-snapshot, so it is deployed even though no kustomization references it
// directly.
func (d *Detector) isRingBaseFile(file string) bool {
	parts := strings.Split(file, "/")
	if len(parts) < 4 || parts[0] != "components" || parts[2] != "base" {
		return false
	}
	return d.head.DirExists("components/" + parts[1] + "/rings")
}

// unreferencedFiles returns changed files under components/ or configs/ that no
// component path, dependency tree or ApplicationSet reference matched. Such a
// file is not deployed by anything, which used to be indistinguishable from a
// change with no environment impact.
func (d *Detector) unreferencedFiles(changedFiles []string, matched map[string]bool) []string {
	var out []string
	for _, f := range changedFiles {
		if matched[f] || !isDeployableFile(f) {
			continue
		}
		if !strings.HasPrefix(f, "components/") && !strings.HasPrefix(f, "configs/") {
			continue
		}
		// Tier 1 base changes are consumed by Kargo through the ring
		// base-snapshot copy, not by a direct kustomization reference.
		if d.isRingBaseFile(f) {
			continue
		}
		// A deleted file was referenced on the base ref and is not dead code
		// just because it no longer exists on HEAD. Base-ref dependency
		// resolution is handled separately.
		if !d.head.FileExists(f) {
			continue
		}
		out = append(out, f)
	}
	sort.Strings(out)
	return out
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
	// A prefix-only path is an unresolved template shared by several
	// clusters, and a MatchAll entry has no path at all: attributing
	// clusters to either would over-label.
	if cp.PrefixOnly || cp.MatchAll {
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
