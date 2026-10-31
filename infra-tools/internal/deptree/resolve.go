// Package deptree walks kustomization.yaml files recursively to build a
// dependency tree of local files. This is used to determine if a changed file
// affects a particular kustomize overlay without actually running kustomize
// build (which would require network access for external URLs).
package deptree

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/logging"
)

// ErrIncomplete is returned together with the dependencies found so far when
// some generator or chart input could not be inspected. Callers must fall back
// to prefix matching as well as using the partial tree.
var ErrIncomplete = errors.New("dependency tree incomplete")

// kustomizationFileNames are the recognized filenames for kustomization files.
var kustomizationFileNames = []string{
	"kustomization.yaml",
	"kustomization.yml",
	"Kustomization",
}

// Resolve recursively walks the kustomization at dir and returns the set of
// all local file paths (relative to repoRoot) that are dependencies. The
// returned map keys are repo-root-relative paths.
func Resolve(repoRoot, dir string) (map[string]bool, error) {
	visited := make(map[string]bool)
	deps := make(map[string]bool)
	absDir, err := filepath.Abs(filepath.Join(repoRoot, dir))
	if err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	if !withinRoot(absRoot, absDir) {
		return nil, fmt.Errorf("path %q escapes repository root", dir)
	}
	incomplete := false
	if err := resolve(absRoot, absDir, deps, visited, &incomplete); err != nil {
		return nil, err
	}
	if incomplete {
		return deps, ErrIncomplete
	}
	return deps, nil
}

// lexicallyWithin reports whether path is inside root without resolving
// symlinks. A symlink that points outside is still a dependency of the
// repository when its own path is local: changing the link changes the build.
func lexicallyWithin(root, path string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	return err == nil && filepath.IsLocal(rel)
}

// withinRoot reports whether path is lexically inside root and, when it and
// root exist, does not escape root through a symlink. Use it before following
// or reading a path; use lexicallyWithin to record an in-repo symlink itself.
func withinRoot(root, path string) bool {
	if !lexicallyWithin(root, path) {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		// The root is not resolvable yet (for example a path that does not
		// exist); the lexical check above is the best we can do.
		return true
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// Missing paths are still allowed: kustomize may generate them.
		return true
	}
	rel, err := filepath.Rel(realRoot, realPath)
	return err == nil && filepath.IsLocal(rel)
}

// hasKustomization returns true if the directory contains a kustomization file.
func hasKustomization(dir string) bool {
	for _, name := range kustomizationFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func resolve(repoRoot, absDir string, deps, visited map[string]bool, incomplete *bool) error {
	// Avoid infinite loops from circular references
	if visited[absDir] {
		return nil
	}
	visited[absDir] = true

	k, kustomFile, err := loadKustomization(repoRoot, absDir)
	if err != nil {
		// No kustomization file in this directory — fall back to scanning
		// subdirectories. This handles the common pattern where an
		// ApplicationSet path like components/X/production/ is a parent
		// directory containing cluster-specific subdirs (base/, kflux-ocp-p01/, etc.).
		entries, readErr := os.ReadDir(absDir)
		if readErr != nil {
			return err // return original kustomization error
		}
		found := false
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subDir := filepath.Join(absDir, entry.Name())
			if hasKustomization(subDir) {
				if subErr := resolve(repoRoot, subDir, deps, visited, incomplete); subErr == nil {
					found = true
				}
			}
		}
		if !found {
			return err // no subdirs had kustomizations either
		}
		return nil
	}

	// Add the kustomization file itself
	relPath, err := filepath.Rel(repoRoot, kustomFile)
	if err != nil {
		return err
	}
	deps[relPath] = true

	// Resources — directories are recursed, files are added, URLs are skipped
	for _, res := range k.Resources {
		if isRemoteURL(res) {
			continue
		}
		absPath := filepath.Join(absDir, res)
		info, err := os.Stat(absPath)
		if err != nil {
			// File/dir doesn't exist — skip gracefully (could be generated)
			continue
		}
		if info.IsDir() {
			if lst, lerr := os.Lstat(absPath); lerr == nil && lst.Mode()&os.ModeSymlink != 0 {
				addDependency(repoRoot, absPath, deps)
			}
			if !withinRoot(repoRoot, absPath) {
				slog.Warn("skipping kustomization resource outside repository", "path", logging.Sanitize(absPath))
				continue
			}
			if err := resolve(repoRoot, absPath, deps, visited, incomplete); err != nil {
				return err
			}
		} else {
			// Record the lexical path even when it is a symlink whose target
			// lives outside the checkout: editing the link changes the build,
			// but its target must not be followed.
			addDependency(repoRoot, absPath, deps)
		}
	}

	// Patches (typed as []types.Patch)
	for _, p := range k.Patches {
		if p.Path != "" {
			addFile(repoRoot, absDir, p.Path, deps)
		}
	}

	// PatchesStrategicMerge — can be file paths or inline YAML.
	// We must support this deprecated field because existing kustomization files use it.
	for _, p := range k.PatchesStrategicMerge { //nolint:staticcheck // deprecated but still in use
		s := string(p)
		// Inline YAML typically contains newlines or starts with a YAML/JSON marker
		if strings.Contains(s, "\n") || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "-") {
			continue
		}
		addFile(repoRoot, absDir, s, deps)
	}

	// PatchesJson6902 — deprecated but still referenced by existing kustomization files.
	for _, p := range k.PatchesJson6902 { //nolint:staticcheck // deprecated but still in use
		if p.Path != "" {
			addFile(repoRoot, absDir, p.Path, deps)
		}
	}

	// Components — these are directories with their own kustomization
	for _, comp := range k.Components {
		if isRemoteURL(comp) {
			continue
		}
		absComp := filepath.Join(absDir, comp)
		if lst, lerr := os.Lstat(absComp); lerr == nil && lst.Mode()&os.ModeSymlink != 0 {
			addDependency(repoRoot, absComp, deps)
		}
		if !withinRoot(repoRoot, absComp) {
			slog.Warn("skipping kustomization component outside repository", "path", logging.Sanitize(absComp))
			continue
		}
		if err := resolve(repoRoot, absComp, deps, visited, incomplete); err != nil {
			return err
		}
	}

	// ConfigMapGenerator
	for i := range k.ConfigMapGenerator {
		addGeneratorSources(repoRoot, absDir, &k.ConfigMapGenerator[i].GeneratorArgs, deps)
	}

	// SecretGenerator
	for i := range k.SecretGenerator {
		addGeneratorSources(repoRoot, absDir, &k.SecretGenerator[i].GeneratorArgs, deps)
	}

	// Generators — kustomize exec/container plugin config files.
	// Changing a generator definition (e.g. HelmChartInflationGenerator YAML)
	// changes the output of the build.
	for _, g := range k.Generators {
		if isRemoteURL(g) {
			continue
		}
		addFile(repoRoot, absDir, g, deps)
		// If the generator is a HelmChartInflationGenerator, its valuesFile
		// and additionalValuesFiles are also dependencies.
		if err := addHelmGeneratorDeps(repoRoot, absDir, g, deps, incomplete); err != nil {
			return err
		}
	}

	// Transformers — kustomize transformer plugin config files.
	for _, t := range k.Transformers {
		if isRemoteURL(t) {
			continue
		}
		addFile(repoRoot, absDir, t, deps)
	}

	// Validators — kustomize validator plugin config files.
	for _, v := range k.Validators {
		if isRemoteURL(v) {
			continue
		}
		addFile(repoRoot, absDir, v, deps)
	}

	// Configurations — transformer configuration files.
	for _, c := range k.Configurations {
		addFile(repoRoot, absDir, c, deps)
	}

	// HelmCharts — the chart directory and any values files are dependencies.
	// chartHome defaults to "charts" per kustomize convention.
	chartHome := types.HelmDefaultHome
	if k.HelmGlobals != nil && k.HelmGlobals.ChartHome != "" {
		chartHome = k.HelmGlobals.ChartHome
	}
	for _, hc := range k.HelmCharts {
		if hc.Name != "" {
			chartDir := filepath.Join(absDir, chartHome, hc.Name)
			if err := addDirTree(repoRoot, chartDir, deps, incomplete); err != nil {
				return fmt.Errorf("walking helm chart %s: %w", hc.Name, err)
			}
		}
		if hc.ValuesFile != "" {
			addFile(repoRoot, absDir, hc.ValuesFile, deps)
		}
		for _, vf := range hc.AdditionalValuesFiles {
			addFile(repoRoot, absDir, vf, deps)
		}
	}

	// CRDs
	for _, crd := range k.Crds {
		addFile(repoRoot, absDir, crd, deps)
	}

	// OpenAPI — if specified
	if k.OpenAPI != nil {
		for _, p := range k.OpenAPI {
			addFile(repoRoot, absDir, p, deps)
		}
	}

	return nil
}

// helmGeneratorConfig captures the fields we need from a
// HelmChartInflationGenerator YAML file to resolve its file dependencies.
type helmGeneratorConfig struct {
	Kind                  string   `yaml:"kind"`
	Name                  string   `yaml:"name"`
	ValuesFile            string   `yaml:"valuesFile"`
	AdditionalValuesFiles []string `yaml:"additionalValuesFiles"`
}

// addHelmGeneratorDeps reads a generator YAML file and, if it is a
// HelmChartInflationGenerator, adds its valuesFile, additionalValuesFiles,
// and local chart directory (charts/<name>) as dependencies.
// Non-HelmChartInflationGenerator files are silently skipped.
func addHelmGeneratorDeps(repoRoot, absDir, generatorPath string, deps map[string]bool, incomplete *bool) error {
	absPath := filepath.Join(absDir, generatorPath)
	// The generator path itself is already recorded by addFile. Do not read
	// through a symlink whose target leaves the checkout.
	if !withinRoot(repoRoot, absPath) {
		slog.Warn("not reading generator outside repository", "path", logging.Sanitize(absPath))
		return nil
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		// A generator may be generated at build time or live outside the
		// repo; keep the dependencies found so far, but tell the caller the
		// tree is incomplete so it also uses prefix matching.
		*incomplete = true
		slog.Warn("cannot read generator, skipping its dependencies",
			"generator", logging.Sanitize(generatorPath), "err", logging.Sanitize(err.Error()))
		return nil
	}

	var cfg helmGeneratorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		*incomplete = true
		slog.Warn("cannot parse generator, skipping its dependencies",
			"generator", logging.Sanitize(generatorPath), "err", logging.Sanitize(err.Error()))
		return nil
	}
	if cfg.Kind != "HelmChartInflationGenerator" {
		return nil
	}
	if cfg.ValuesFile != "" {
		addFile(repoRoot, absDir, cfg.ValuesFile, deps)
	}
	for _, vf := range cfg.AdditionalValuesFiles {
		addFile(repoRoot, absDir, vf, deps)
	}
	// If a local chart directory exists (charts/<name>), track it.
	if cfg.Name != "" {
		chartDir := filepath.Join(absDir, types.HelmDefaultHome, cfg.Name)
		if info, err := os.Stat(chartDir); err == nil && info.IsDir() {
			if err := addDirTree(repoRoot, chartDir, deps, incomplete); err != nil {
				return fmt.Errorf("walking local chart %s: %w", cfg.Name, err)
			}
		}
	}
	return nil
}

// addGeneratorSources adds file sources and env sources from a generator.
func addGeneratorSources(repoRoot, absDir string, gen *types.GeneratorArgs, deps map[string]bool) {
	for _, f := range gen.FileSources {
		// Format: "key=path" or just "path"
		path := f
		if idx := strings.Index(f, "="); idx >= 0 {
			path = f[idx+1:]
		}
		addFile(repoRoot, absDir, path, deps)
	}
	for _, e := range gen.EnvSources {
		addFile(repoRoot, absDir, e, deps)
	}
	if gen.EnvSource != "" {
		addFile(repoRoot, absDir, gen.EnvSource, deps)
	}
}

// addDirTree recursively walks a directory tree and adds every file it
// contains to the dependency set. This is used for Helm chart directories
// where any file change (templates, helpers, Chart.yaml, etc.) affects the
// build output.
//
// A missing directory is not an error: charts are often pulled from a remote
// repository at build time, and failing here would discard the rest of the
// dependency tree.
func addDirTree(repoRoot, absDir string, deps map[string]bool, incomplete *bool) error {
	if _, err := os.Stat(absDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Debug("chart directory not present locally, skipping",
				"dir", logging.Sanitize(absDir), "err", logging.Sanitize(err.Error()))
		} else {
			*incomplete = true
			slog.Warn("cannot read chart directory, skipping",
				"dir", logging.Sanitize(absDir), "err", logging.Sanitize(err.Error()))
		}
		return nil
	}
	if !withinRoot(repoRoot, absDir) {
		slog.Warn("skipping chart directory outside repository", "path", logging.Sanitize(absDir))
		return nil
	}
	return filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			*incomplete = true
			slog.Warn("cannot walk chart directory entry, skipping",
				"path", logging.Sanitize(path), "err", logging.Sanitize(err.Error()))
			return nil //nolint:nilerr // best-effort: keep the rest of the tree
		}
		if info.IsDir() {
			return nil
		}
		addDependency(repoRoot, path, deps)
		return nil
	})
}

// addDependency records a lexically contained path. For an in-repo symlink it
// also records the target when that target stays inside the checkout; an
// external target is never followed.
func addDependency(repoRoot, absPath string, deps map[string]bool) {
	if !lexicallyWithin(repoRoot, absPath) {
		return
	}
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return
	}
	deps[rel] = true

	info, err := os.Lstat(absPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return
	}
	target, err := filepath.EvalSymlinks(absPath)
	if err != nil || !withinRoot(repoRoot, target) {
		return
	}
	if relTarget, err := filepath.Rel(repoRoot, target); err == nil {
		deps[relTarget] = true
	}
}

// addFile resolves a relative file path and adds it to the dependency set. The
// path is recorded even when it is a symlink to an external target: changing
// the link itself changes the build, but the target is never followed here.
func addFile(repoRoot, absDir, relFilePath string, deps map[string]bool) {
	addDependency(repoRoot, filepath.Join(absDir, relFilePath), deps)
}

// loadKustomization finds and parses the kustomization file in dir. A
// kustomization file that is a symlink out of the checkout is not read.
func loadKustomization(repoRoot, dir string) (*types.Kustomization, string, error) {
	for _, name := range kustomizationFileNames {
		path := filepath.Join(dir, name)
		if !withinRoot(repoRoot, path) {
			slog.Warn("skipping kustomization file outside repository", "path", logging.Sanitize(path))
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, "", err
		}
		var k types.Kustomization
		if err := yaml.Unmarshal(data, &k); err != nil {
			return nil, "", fmt.Errorf("parsing %s: %w", path, err)
		}
		// Apply kustomize's own normalisation so deprecated fields are handled
		// the way kustomize handles them: 'bases' become 'resources',
		// 'imageTags' become 'images', and the legacy
		// HelmChartInflationGenerator entries become HelmCharts.
		k.FixKustomization()
		return &k, path, nil
	}
	return nil, "", fmt.Errorf("no kustomization file found in %s", dir)
}

// isRemoteURL checks if a resource reference is a remote URL.
func isRemoteURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "ssh://") ||
		strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "git://")
}
