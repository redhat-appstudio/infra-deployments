// Package changelog provides logic for generating a human-readable changelog
// when the Konflux operator SHA is bumped in infra-deployments.
package changelog

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// operatorImageName is the kustomize image name used to pin the operator.
const operatorImageName = "localhost/konflux-operator"

// shaPattern matches a valid 40-character lowercase hex SHA.
var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// kustomization is a minimal representation of the fields we need from the
// operator kustomization.yaml. Only the images block is parsed.
type kustomization struct {
	Images []kustomizeImage `yaml:"images"`
}

type kustomizeImage struct {
	Name    string `yaml:"name"`
	NewName string `yaml:"newName"`
	NewTag  string `yaml:"newTag"`
}

// ExtractSHA reads the kustomization.yaml at the given path and returns the
// operator commit SHA from the newTag field of the localhost/konflux-operator
// image entry.
//
// Returns an error if the file cannot be read, the YAML cannot be parsed, the
// expected image entry is missing, or the tag is not a 40-character hex SHA.
func ExtractSHA(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	var k kustomization
	if err := yaml.Unmarshal(data, &k); err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}

	for _, img := range k.Images {
		if img.Name == operatorImageName {
			if !shaPattern.MatchString(img.NewTag) {
				return "", fmt.Errorf("%s: newTag %q is not a 40-character hex SHA", path, img.NewTag)
			}
			return img.NewTag, nil
		}
	}

	return "", fmt.Errorf("%s: no image entry found with name %q", path, operatorImageName)
}

// ExtractSHAs reads the kustomization.yaml at both basePath (base branch) and
// headPath (PR branch) and returns (oldSHA, newSHA, error). When the two SHAs
// are equal the operator was not bumped and changelog generation can be skipped.
func ExtractSHAs(basePath, headPath string) (string, string, error) {
	oldSHA, err := ExtractSHA(basePath)
	if err != nil {
		return "", "", fmt.Errorf("extracting base SHA: %w", err)
	}

	newSHA, err := ExtractSHA(headPath)
	if err != nil {
		return "", "", fmt.Errorf("extracting head SHA: %w", err)
	}

	return oldSHA, newSHA, nil
}
