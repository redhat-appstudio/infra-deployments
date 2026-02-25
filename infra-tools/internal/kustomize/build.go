// Package kustomize provides helpers for building kustomize overlays using the
// krusty library. It is used only for building the ArgoCD overlay layer.
package kustomize

import (
	"fmt"
	"os"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// Build runs kustomize build on the given directory and returns the rendered
// YAML output as a byte slice. Deprecation warnings from kustomize are
// silenced by temporarily redirecting stderr.
func Build(dir string) ([]byte, error) {
	fSys := filesys.MakeFsOnDisk()
	opts := krusty.MakeDefaultOptions()
	// Allow loading files from outside the kustomization root since overlays
	// reference ../../base/ paths.
	opts.LoadRestrictions = 0
	k := krusty.MakeKustomizer(opts)

	// Silence kustomize deprecation warnings (patchesStrategicMerge,
	// commonLabels, etc.) which are written directly to stderr.
	restoreStderr := muteStderr()
	resMap, err := k.Run(fSys, dir)
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

// muteStderr redirects stderr to /dev/null and returns a function that
// restores the original stderr.
func muteStderr() func() {
	origStderr := os.Stderr
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return func() {} // can't mute, no-op restore
	}
	os.Stderr = devNull
	return func() {
		os.Stderr = origStderr
		_ = devNull.Close()
	}
}
