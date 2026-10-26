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
	"rd-staging":                Staging,
	"production":                Production,
	"production-downstream":     Production,
	"konflux-public-production": Production,
	"rd-production":             Production,
	"rd-dev":                    Development,
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
// A direct conflict means files that belong to staging only and files that
// belong to production only are modified — the PR must be split. An indirect
// conflict means both environments are affected (per the detector's kustomize
// analysis) but through shared files — this is allowed with a warning.
//
// fileEnvs carries the environments each changed file was matched to by the
// detector. It is what makes ring-based layouts work: a file under
// components/X/rings/ring-1/ carries no environment in its path, its
// environment is the overlay that deploys it. When fileEnvs has no entry for a
// file (or is nil), the path segments are used instead.
func CheckRingDeployment(changedFiles []string, affectedEnvs map[Environment]bool, fileEnvs map[string]map[Environment]bool) *RingCheckResult {
	result := &RingCheckResult{}

	for _, f := range changedFiles {
		envs := fileEnvironments(f, fileEnvs)
		switch {
		case envs[Staging] && envs[Production]:
			// Shared by both environments: a base file, not a direct conflict.
		case envs[Staging]:
			result.StagingFiles = append(result.StagingFiles, f)
		case envs[Production]:
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

// fileEnvironments returns the environments a changed file belongs to, falling
// back to the path-based classification when the detector has no entry.
func fileEnvironments(file string, fileEnvs map[string]map[Environment]bool) map[Environment]bool {
	if envs := fileEnvs[file]; len(envs) > 0 {
		return envs
	}
	envs := make(map[Environment]bool, 1)
	if env := ClassifyFileEnv(file); env != "" {
		envs[env] = true
	}
	return envs
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
