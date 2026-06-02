// Package changelog provides logic for generating a human-readable changelog
// when the Konflux operator ref is bumped in infra-deployments.
package changelog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// operatorResourceURL is the prefix of the GitHub URL used in the resources
// block to pin the operator source. The ?ref= query parameter holds the git ref.
const operatorResourceURL = "https://github.com/konflux-ci/konflux-ci/"

// kustomization is a minimal representation of the fields we need from the
// operator kustomization.yaml. Only the resources block is parsed.
type kustomization struct {
	Resources []string `yaml:"resources"`
}

// ExtractRef reads the kustomization.yaml at the given path and returns the
// git ref from the ?ref= query parameter of the operator resource URL.
//
// Returns an error if the file cannot be read, the YAML cannot be parsed, or
// no operator resource URL is found. The ref value is returned as-is — format
// validation happens later at the point of use.
func ExtractRef(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	var k kustomization
	if err := yaml.Unmarshal(data, &k); err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}

	for _, resource := range k.Resources {
		if !strings.Contains(resource, operatorResourceURL) {
			continue
		}
		parts := strings.SplitN(resource, "?ref=", 2)
		if len(parts) == 2 && parts[1] != "" {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("%s: no operator resource URL containing %q found", path, operatorResourceURL)
}

// ExtractRefs reads the kustomization.yaml at both basePath (base branch) and
// headPath (PR branch) and returns (oldRef, newRef, error). When the two refs
// are equal the operator was not bumped and changelog generation can be skipped.
func ExtractRefs(basePath, headPath string) (string, string, error) {
	oldRef, err := ExtractRef(basePath)
	if err != nil {
		return "", "", fmt.Errorf("extracting base ref: %w", err)
	}

	newRef, err := ExtractRef(headPath)
	if err != nil {
		return "", "", fmt.Errorf("extracting head ref: %w", err)
	}

	return oldRef, newRef, nil
}
