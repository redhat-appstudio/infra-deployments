package renderdiff

import "strings"

// kustomizationMissingSubstring is the error substring emitted by krusty when a
// directory does not contain a kustomization.yaml, kustomization.yml, or
// Kustomization file. See sigs.k8s.io/kustomize/api/internal/target.errMissingKustomization.
const kustomizationMissingSubstring = "unable to find one of"

// IsNotKustomizationError reports whether the error string indicates that a
// directory is not a kustomization root (no kustomization file found).
func IsNotKustomizationError(errMsg string) bool {
	return strings.Contains(errMsg, kustomizationMissingSubstring)
}
