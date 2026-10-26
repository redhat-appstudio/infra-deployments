// Package detector implements the core logic for determining which
// environments and clusters a set of changed files affects.
//
// Environment attribution has two rules:
//
//   - A component path inherits the environment of the overlay whose
//     ApplicationSet produced it (staging-downstream → staging, rd-production →
//     production, ...). Ring layouts (components/X/rings/ring-N/<cluster>) have
//     no environment in their path, so this is their only attribution.
//   - An overlay diff is attributed to the environment declared in the
//     ApplicationSet generator, falling back to the overlay's environment, so a
//     change in a shared argo-cd-apps/base directory does not mark every
//     environment as affected.
package detector

import (
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
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
	UnreferencedFiles []string
	// FileEnvs maps each changed file to the environments it was matched to
	// through a component. A file matched to both staging and production is a
	// shared base file; a file matched to only one is specific to it. Ring-based
	// layouts have no environment in their path, so this is the only reliable
	// signal for ring policy enforcement.
	FileEnvs map[string]map[Environment]bool
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
func (d *Detector) AffectedComponents(changedFiles []string) (map[Environment][]appset.ComponentPath, error) {
	// Phase 1: Build ArgoCD ApplicationSet overlays on HEAD and base-ref
	builds, err := d.buildAppSetOverlays()
	if err != nil {
		return nil, err
	}

	// Phase 2: Extract component paths from ApplicationSets
	envPaths, _, err := extractPathsFromOverlays(builds)
	if err != nil {
		return nil, err
	}

	// Phase 3: Resolve dependency trees for component paths
	resolved := d.resolveComponentDeps(envPaths)

	// Phase 4: Match changed files against resolved dependencies
	affected := make(map[Environment][]appset.ComponentPath)
	for _, mc := range matchComponents(changedFiles, resolved) {
		affected[mc.env] = append(affected[mc.env], mc.cp)
	}

	// Phase 5: An ApplicationSet change (new cluster in a generator list, new
	// ApplicationSet, changed path template) affects components that no changed
	// file points at. Render the paths that only exist because of that change.
	addedPaths, err := d.pathsAddedByOverlayChanges(builds)
	if err != nil {
		return nil, err
	}
	for env, paths := range addedPaths {
		affected[env] = appendNewComponentPaths(affected[env], paths)
	}

	return affected, nil
}

// appendNewComponentPaths appends the component paths of extra that are not yet
// present in existing, preserving order.
func appendNewComponentPaths(existing, extra []appset.ComponentPath) []appset.ComponentPath {
	seen := make(map[string]bool, len(existing))
	for _, cp := range existing {
		seen[componentPathKey(cp)] = true
	}
	for _, cp := range extra {
		key := componentPathKey(cp)
		if seen[key] {
			continue
		}
		seen[key] = true
		existing = append(existing, cp)
	}
	return existing
}

// pathsAddedByOverlayChanges returns, per environment, the component paths that
// are rendered on HEAD but not on the base ref. These appear when a PR adds an
// ApplicationSet, adds a cluster to a generator list, or changes a path
// template, and they must be rendered even though no changed file lives there.
func (d *Detector) pathsAddedByOverlayChanges(builds []overlayBuild) (map[Environment][]appset.ComponentPath, error) {
	added := make(map[Environment][]appset.ComponentPath)

	for _, ob := range builds {
		if string(ob.headYAML) == string(ob.baseYAML) {
			continue
		}

		headPaths, err := appset.ParseApplicationSets(ob.headYAML)
		if err != nil {
			return nil, fmt.Errorf("parsing ApplicationSets from overlay %s: %w", ob.name, err)
		}
		// The base render is best-effort: without it every head path looks new.
		var basePaths *appset.ParseResult
		if len(ob.baseYAML) > 0 {
			basePaths, err = appset.ParseApplicationSets(ob.baseYAML)
			if err != nil {
				slog.Warn("failed to parse base-ref ApplicationSets, treating all paths as new",
					"overlay", ob.name, "err", err)
				basePaths = nil
			}
		}

		known := make(map[string]bool)
		if basePaths != nil {
			for _, cp := range basePaths.Paths {
				known[componentPathKey(cp)] = true
			}
		}
		for _, cp := range headPaths.Paths {
			if known[componentPathKey(cp)] {
				continue
			}
			// Only paths that exist somewhere can be rendered; a templated path
			// pointing at a directory that is neither on HEAD nor on the base ref
			// is a stale reference, not something to diff.
			if cp.Path == "" && len(cp.ExtraFiles) == 0 {
				continue
			}
			if cp.Path != "" && !d.head.DirExists(cp.Path) && !d.base.DirExists(cp.Path) {
				continue
			}
			added[ob.env] = append(added[ob.env], cp)
		}
	}
	return added, nil
}

// componentPathKey identifies a ComponentPath in a comparable form.
func componentPathKey(cp appset.ComponentPath) string {
	return cp.Path + "|" + cp.ClusterDir + "|" + strings.Join(cp.ExtraFiles, ",")
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
		FileEnvs:             make(map[string]map[Environment]bool),
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
	matched := matchComponents(changedFiles, resolved)
	applyComponentMatches(matched, allClusters, result)
	result.UnreferencedFiles = unreferencedFiles(changedFiles, matchedFileSet(matched))

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
// When an ApplicationSet has an explicit environment in its generator, that
// environment is used instead of the overlay's environment. This prevents base
// changes (included by all overlays) from spuriously marking every environment.
func detectOverlayDiffs(builds []overlayBuild, result *Result) {
	for _, ob := range builds {
		if string(ob.headYAML) == string(ob.baseYAML) {
			continue
		}
		slog.Info("Overlay diff detected", "overlay", ob.name)

		headSets, headErr := appset.AppSetsByName(ob.headYAML)
		baseSets, baseErr := appset.AppSetsByName(ob.baseYAML)
		if headErr != nil || baseErr != nil {
			// Can't parse ApplicationSets; conservatively mark overlay environment.
			slog.Warn("Failed to parse ApplicationSets, falling back to overlay env",
				"overlay", ob.name, "headErr", headErr, "baseErr", baseErr)
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
			slog.Info("AppSet added/modified", "overlay", ob.name, "appset", name, "env", env)
			attributed = true
		}
		for name, baseDoc := range baseSets {
			if _, exists := headSets[name]; !exists {
				env := appSetEnv(baseDoc, ob.env)
				result.AffectedEnvironments[env] = true
				slog.Info("AppSet removed", "overlay", ob.name, "appset", name, "env", env)
				attributed = true
			}
		}
		if !attributed {
			// YAML changed but no ApplicationSet changes found (e.g. namespace patch).
			result.AffectedEnvironments[ob.env] = true
		}
	}
}

// appSetEnv extracts the environment from an ApplicationSet's generator. Falls
// back to overlayEnv when no explicit environment is set.
func appSetEnv(doc map[string]interface{}, overlayEnv Environment) Environment {
	if envStr, ok := appset.EnvironmentFromAppSet(doc); ok {
		if e := Environment(envStr); e == Development || e == Staging || e == Production {
			return e
		}
	}
	return overlayEnv
}

// extractPathsFromOverlays parses ApplicationSets from both the HEAD and the
// base builds and returns per-environment component paths and a mapping from
// cluster name to the component paths deployed on that cluster.
//
// Both refs are needed: a component that this PR removes is only referenced on
// the base ref, and a component this PR adds only on HEAD. Parsing HEAD alone
// would hide removals from the diff.
func extractPathsFromOverlays(builds []overlayBuild) (map[Environment][]appset.ComponentPath, map[string][]string, error) {
	envPaths := make(map[Environment][]appset.ComponentPath)
	allClusters := make(map[string][]string)
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
					return nil, nil, fmt.Errorf("parsing ApplicationSets from overlay %s: %w", ob.name, err)
				}
				// The base render is best-effort context (it only adds removed
				// components); a parse failure must not fail the whole run.
				slog.Warn("failed to parse base-ref ApplicationSets, continuing with HEAD only",
					"overlay", ob.name, "err", err)
				continue
			}

			// Every path inherits the environment from the overlay that produced it.
			for _, cp := range parsed.Paths {
				key := string(ob.env) + "|" + cp.Path + "|" + cp.ClusterDir + "|" + strings.Join(cp.ExtraFiles, ",")
				if seen[key] {
					continue
				}
				seen[key] = true
				envPaths[ob.env] = append(envPaths[ob.env], cp)
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
	type componentJob struct {
		env Environment
		cp  appset.ComponentPath
	}

	seen := make(map[string]bool)
	var jobs []componentJob
	for env, paths := range envPaths {
		for _, cp := range paths {
			// The key must include ExtraFiles: several paths can share the same
			// deployed directory while contributing different value files, and
			// ExtraFiles-only entries all have an empty Path.
			key := string(env) + "|" + cp.Path + "|" + strings.Join(cp.ExtraFiles, ",")
			if seen[key] {
				continue
			}
			seen[key] = true

			// A component path may only carry individual files (e.g. a Helm
			// values file from a multi-source template); there is nothing to
			// resolve, the files are matched exactly.
			if cp.Path == "" && len(cp.ExtraFiles) == 0 {
				continue
			}

			// The path may exist on neither ref when the ApplicationSet
			// reference is stale or a templated path resolves to a directory
			// that hasn't been created yet.
			if cp.Path != "" && !cp.PrefixOnly && !d.head.DirExists(cp.Path) && !d.base.DirExists(cp.Path) {
				continue
			}
			jobs = append(jobs, componentJob{env: env, cp: cp})
		}
	}

	// Dependency resolution walks files, so run it in parallel; the RepoRef
	// resolver caches per directory, which makes shared base trees cheap.
	results := make([][]componentDeps, len(jobs))
	g := errgroup.Group{}
	g.SetLimit(runtime.NumCPU())
	for i, job := range jobs {
		g.Go(func() error {
			results[i] = []componentDeps{d.resolveComponent(job.env, job.cp)}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		// The workers never fail, but keep the error path honest.
		slog.Warn("resolving dependency trees failed", "err", err)
	}

	resolved := make([]componentDeps, 0, len(jobs))
	for _, r := range results {
		resolved = append(resolved, r...)
	}
	return resolved
}

// resolveComponent resolves one component path, choosing the matching strategy:
// exact extra files, prefix matching for unresolved templates, or a dependency
// tree resolved on both refs.
func (d *Detector) resolveComponent(env Environment, cp appset.ComponentPath) componentDeps {
	if cp.Path == "" {
		return componentDeps{cp: cp, env: env}
	}
	// A prefix-only path is an unresolved template: the directory may not exist
	// (that is why it is a prefix), so match by prefix.
	if cp.PrefixOnly {
		return componentDeps{cp: cp, env: env}
	}

	deps := d.resolvePathDeps(cp.Path, d.head.DirExists(cp.Path), d.base.DirExists(cp.Path))
	if deps == nil {
		// No kustomization.yaml — the ApplicationSet may deploy the directory
		// as plain YAML manifests.
		slog.Warn("no dep tree, will use prefix match", "path", cp.Path, "env", env)
	} else {
		slog.Debug("resolved dependency tree", "path", cp.Path, "env", env, "deps", deps)
	}
	return componentDeps{cp: cp, env: env, deps: deps}
}

// matchesFile reports whether a single changed file affects this component.
// ExtraFiles (e.g. Helm values files) are matched exactly; dependency trees are
// matched exactly; anything else is matched by path prefix.
func (cd componentDeps) matchesFile(file string) bool {
	for _, f := range cd.cp.ExtraFiles {
		if f == file {
			return true
		}
	}
	if cd.cp.Path == "" {
		return false
	}
	if cd.deps != nil && !cd.cp.PrefixOnly {
		return cd.deps[file]
	}
	return matchesPrefix(file, cd.cp.Path)
}

// matchedComponent is a component affected by at least one changed file.
type matchedComponent struct {
	cp    appset.ComponentPath
	env   Environment
	files []string
}

// matchComponents returns the components affected by changedFiles, together with
// the files that matched each of them. It is the single matching pass shared by
// Detect and AffectedComponents, so both always agree.
func matchComponents(changedFiles []string, resolved []componentDeps) []matchedComponent {
	changedFiles = slices.Clone(changedFiles)
	sort.Strings(changedFiles)

	var matched []matchedComponent
	for _, cd := range resolved {
		var files []string
		for _, f := range changedFiles {
			if cd.matchesFile(f) {
				files = append(files, f)
			}
		}
		if len(files) > 0 {
			matched = append(matched, matchedComponent{cp: cd.cp, env: cd.env, files: files})
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].env != matched[j].env {
			return matched[i].env < matched[j].env
		}
		return matched[i].cp.Path < matched[j].cp.Path
	})
	return matched
}

// matchedFileSet returns the set of changed files that matched a component.
func matchedFileSet(matched []matchedComponent) map[string]bool {
	files := make(map[string]bool)
	for _, mc := range matched {
		for _, f := range mc.files {
			files[f] = true
		}
	}
	return files
}

// resolvePathDeps returns the union of the kustomize dependency trees of rel on
// the refs where it exists. Union matters at the edges of a PR: a kustomization
// that gains a dependency in HEAD must not hide a removal relative to base, and
// vice versa. A nil result means neither ref has a kustomization there.
func (d *Detector) resolvePathDeps(rel string, headExists, baseExists bool) map[string]bool {
	var deps map[string]bool
	merge := func(m map[string]bool) {
		if len(m) == 0 {
			return
		}
		if deps == nil {
			deps = make(map[string]bool, len(m))
		}
		for k := range m {
			deps[k] = true
		}
	}

	if headExists {
		m, err := d.head.ResolveDeps(rel)
		if err != nil {
			slog.Debug("no dependency tree on HEAD", "path", rel, "err", err)
		}
		merge(m)
	}
	if baseExists {
		m, err := d.base.ResolveDeps(rel)
		if err != nil {
			slog.Debug("no dependency tree on base ref", "path", rel, "err", err)
		}
		merge(m)
	}
	return deps
}

// matchChangedFiles checks each resolved component against the set of changed
// files.  For components with a dependency tree it uses an exact match; for
// components without a kustomization.yaml it falls back to prefix matching.
// applyComponentMatches records environments, clusters and per-file environment
// attribution for the components affected by the change.
func applyComponentMatches(matched []matchedComponent, allClusters map[string][]string, result *Result) {
	if result.FileEnvs == nil {
		result.FileEnvs = make(map[string]map[Environment]bool)
	}
	for _, mc := range matched {
		result.AffectedEnvironments[mc.env] = true
		matchClusters(mc.cp, allClusters, result)
		for _, f := range mc.files {
			if result.FileEnvs[f] == nil {
				result.FileEnvs[f] = make(map[Environment]bool)
			}
			result.FileEnvs[f][mc.env] = true
		}
		slog.Info("Changed files match component", "path", mc.cp.Path, "env", mc.env)
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

// isDeployableFile reports whether a changed file may be consumed by kustomize.
func isDeployableFile(file string) bool {
	if nonDeployableFiles[path.Base(file)] {
		return false
	}
	switch strings.ToLower(path.Ext(file)) {
	case ".md", ".png", ".jpg", ".jpeg", ".svg", ".gif":
		return false
	default:
		return true
	}
}

// unreferencedFiles returns changed files under components/ or configs/ that no
// component path, dependency tree or ApplicationSet reference matched. Such a
// file is not deployed by anything, which used to be indistinguishable from a
// change with no environment impact.
func unreferencedFiles(changedFiles []string, matched map[string]bool) []string {
	var out []string
	for _, f := range changedFiles {
		if matched[f] || !isDeployableFile(f) {
			continue
		}
		if !strings.HasPrefix(f, "components/") && !strings.HasPrefix(f, "configs/") {
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
