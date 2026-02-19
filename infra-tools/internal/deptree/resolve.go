// Package deptree walks kustomization.yaml files recursively to build a
// dependency tree of local files. This is used to determine if a changed file
// affects a particular kustomize overlay without actually running kustomize
// build (which would require network access for external URLs).
package deptree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

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
	if err := resolve(absRoot, absDir, deps, visited); err != nil {
		return nil, err
	}
	return deps, nil
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

func resolve(repoRoot, absDir string, deps, visited map[string]bool) error {
	// Avoid infinite loops from circular references
	if visited[absDir] {
		return nil
	}
	visited[absDir] = true

	k, kustomFile, err := loadKustomization(absDir)
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
				if subErr := resolve(repoRoot, subDir, deps, visited); subErr == nil {
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
			if err := resolve(repoRoot, absPath, deps, visited); err != nil {
				return err
			}
		} else {
			rel, err := filepath.Rel(repoRoot, absPath)
			if err != nil {
				return err
			}
			deps[rel] = true
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
		if err := resolve(repoRoot, absComp, deps, visited); err != nil {
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

// addFile resolves a relative file path and adds it to the dependency set.
func addFile(repoRoot, absDir, relFilePath string, deps map[string]bool) {
	absPath := filepath.Join(absDir, relFilePath)
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return
	}
	deps[rel] = true
}

// loadKustomization finds and parses the kustomization file in dir.
func loadKustomization(dir string) (*types.Kustomization, string, error) {
	for _, name := range kustomizationFileNames {
		path := filepath.Join(dir, name)
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
