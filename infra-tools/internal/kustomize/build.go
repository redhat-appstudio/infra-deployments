// Package kustomize provides helpers for building kustomize overlays using the
// krusty library. It is used only for building the ArgoCD overlay layer.
package kustomize

import (
	"fmt"
	"os"
	"sync"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// newKustomizer builds a kustomizer configured the way every build needs it.
// A new instance per build is cheap and keeps the options immutable, so builds
// can run concurrently.
func newKustomizer() *krusty.Kustomizer {
	opts := krusty.MakeDefaultOptions()
	// Allow loading files from outside the kustomization root since overlays
	// reference ../../base/ paths.
	opts.LoadRestrictions = 0
	// Enable Helm chart inflation so components using
	// HelmChartInflationGenerator can be built.
	opts.PluginConfig.HelmConfig.Enabled = true
	opts.PluginConfig.HelmConfig.Command = "helm"
	return krusty.MakeKustomizer(opts)
}

// Build runs kustomize build on the given directory and returns the rendered
// YAML output as a byte slice. Deprecation warnings from kustomize are
// silenced by temporarily redirecting stderr.
func Build(dir string) ([]byte, error) {
	// Silence kustomize deprecation warnings (patchesStrategicMerge,
	// commonLabels, etc.) which are written directly to stderr.
	restoreStderr := muteStderr()
	resMap, err := newKustomizer().Run(filesys.MakeFsOnDisk(), dir)
	restoreStderr()

	if err != nil {
		return nil, fmt.Errorf("kustomize build %s: %w", dir, err)
	}
	yamlBytes, err := resMap.AsYaml()
	if err != nil {
		return nil, fmt.Errorf("converting resmap to yaml for %s: %w", dir, err)
	}
	return yamlBytes, nil
}

// Stderr muting is process-global because kustomize writes deprecation warnings
// to os.Stderr directly. Builds run concurrently (overlay builds in the
// detector and component builds in the render-diff engine), so the swap is
// reference-counted: the first build redirects stderr, the last one restores it.
var (
	stderrMu      sync.Mutex
	stderrRefs    int
	stderrSaved   *os.File
	stderrDevNull *os.File
)

// muteStderr redirects stderr to /dev/null until the returned function is
// called. It is safe for concurrent use.
func muteStderr() func() {
	stderrMu.Lock()
	defer stderrMu.Unlock()

	if stderrRefs == 0 {
		devNull, err := os.Open(os.DevNull)
		if err != nil {
			// Can't mute: return a no-op restore rather than failing the build.
			return func() {}
		}
		stderrSaved = os.Stderr
		stderrDevNull = devNull
		os.Stderr = devNull
	}
	stderrRefs++

	var once sync.Once
	return func() {
		once.Do(func() {
			stderrMu.Lock()
			defer stderrMu.Unlock()

			stderrRefs--
			if stderrRefs > 0 {
				return
			}
			if stderrSaved != nil {
				os.Stderr = stderrSaved
				stderrSaved = nil
			}
			if stderrDevNull != nil {
				_ = stderrDevNull.Close()
				stderrDevNull = nil
			}
		})
	}
}
