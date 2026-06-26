package detector

import (
	"sort"
	"strings"
)

// segmentEnvironment maps directory names that appear as path segments to
// their environment. This covers both component overlay directories (e.g.
// staging/, production/) and ArgoCD overlay directories (e.g.
// konflux-public-staging/).
var segmentEnvironment = map[string]Environment{
	"staging":                   Staging,
	"staging-downstream":        Staging,
	"konflux-public-staging":    Staging,
	"production":                Production,
	"production-downstream":     Production,
	"konflux-public-production": Production,
}

// RingCheckResult holds the outcome of the ring deployment validation.
type RingCheckResult struct {
	// DirectConflict is true when changed files directly modify both staging
	// and production overlay directories in the same PR.
	DirectConflict bool
	// IndirectConflict is true when both environments are affected (via
	// kustomize dependency trees) but no changed file directly lives in
	// both a staging and a production overlay directory.
	IndirectConflict bool
	// StagingFiles lists changed files that directly belong to staging overlays.
	StagingFiles []string
	// ProductionFiles lists changed files that directly belong to production overlays.
	ProductionFiles []string
}

// CheckRingDeployment determines whether a set of changed files violates the
// ring deployment policy.
//
// A direct conflict means files under both staging and production directories
// are modified — the PR must be split. An indirect conflict means both
// environments are affected (per the detector's kustomize analysis) but only
// through shared base files — this is allowed with a warning.
func CheckRingDeployment(changedFiles []string, affectedEnvs map[Environment]bool) *RingCheckResult {
	result := &RingCheckResult{}

	for _, f := range changedFiles {
		switch ClassifyFileEnv(f) {
		case Staging:
			result.StagingFiles = append(result.StagingFiles, f)
		case Production:
			result.ProductionFiles = append(result.ProductionFiles, f)
		}
	}

	sort.Strings(result.StagingFiles)
	sort.Strings(result.ProductionFiles)

	if len(result.StagingFiles) > 0 && len(result.ProductionFiles) > 0 {
		result.DirectConflict = true
	} else if affectedEnvs[Staging] && affectedEnvs[Production] {
		result.IndirectConflict = true
	}

	return result
}

// ClassifyFileEnv returns the environment a file directly belongs to based
// on its path segments, or "" if the file is not in an environment-specific
// overlay directory.
func ClassifyFileEnv(path string) Environment {
	for _, segment := range strings.Split(path, "/") {
		if env, ok := segmentEnvironment[segment]; ok {
			return env
		}
	}
	return ""
}
