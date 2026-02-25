// Package appset extracts environment-to-path mappings and cluster names from
// rendered ApplicationSet YAML produced by kustomize build.
package appset

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComponentPath represents a single path extracted from an ApplicationSet with
// an optional cluster directory.  The environment is not stored here; it is
// determined by the overlay that produced the ApplicationSet.
type ComponentPath struct {
	// Path is the resolved filesystem path (relative to repo root) that the
	// ApplicationSet points to.
	Path string
	// ClusterDir is non-empty when the ApplicationSet targets a specific cluster
	// subdirectory (e.g. "stone-prod-p01").
	ClusterDir string
}

// ParseResult holds the extracted data from all ApplicationSets in one overlay.
type ParseResult struct {
	// Paths contains all resolved component/config paths.
	Paths []ComponentPath
	// Clusters maps cluster names found in list.elements[].nameNormalized.
	// Key: cluster name, Value: list of component paths for that cluster.
	Clusters map[string][]string
}

// ParseApplicationSets parses rendered YAML (multi-document) and extracts
// ComponentPaths and cluster names from all ApplicationSet resources.
func ParseApplicationSets(renderedYAML []byte) (*ParseResult, error) {
	result := &ParseResult{
		Clusters: make(map[string][]string),
	}

	decoder := yaml.NewDecoder(bytes.NewReader(renderedYAML))
	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding YAML document: %w", err)
		}
		if doc == nil {
			continue
		}

		kind, _ := doc["kind"].(string)
		if kind != "ApplicationSet" {
			continue
		}

		paths, clusters, err := extractFromAppSet(doc)
		if err != nil {
			name := "unknown"
			if md, ok := doc["metadata"].(map[string]interface{}); ok {
				if n, ok := md["name"].(string); ok {
					name = n
				}
			}
			return nil, fmt.Errorf("extracting from ApplicationSet %s: %w", name, err)
		}
		result.Paths = append(result.Paths, paths...)
		for cluster, cpaths := range clusters {
			result.Clusters[cluster] = append(result.Clusters[cluster], cpaths...)
		}
	}

	return result, nil
}

// extractFromAppSet processes a single ApplicationSet document.
func extractFromAppSet(doc map[string]interface{}) ([]ComponentPath, map[string][]string, error) {
	spec, ok := doc["spec"].(map[string]interface{})
	if !ok {
		return nil, nil, nil
	}
	tmpl, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil, nil, nil
	}
	tmplSpec, ok := tmpl["spec"].(map[string]interface{})
	if !ok {
		return nil, nil, nil
	}
	source, ok := tmplSpec["source"].(map[string]interface{})
	if !ok {
		return nil, nil, nil
	}
	pathTemplate, ok := source["path"].(string)
	if !ok {
		return nil, nil, nil
	}

	generators, _ := spec["generators"].([]interface{})
	if len(generators) == 0 {
		return nil, nil, nil
	}

	// Determine if this is a templated or static path
	if !strings.Contains(pathTemplate, "{{") {
		// Static path — no templates at all
		return []ComponentPath{{
			Path: pathTemplate,
		}}, nil, nil
	}

	// Templated path — extract values from generators
	return extractTemplatedPaths(pathTemplate, generators)
}

// extractTemplatedPaths resolves templated paths from merge/cluster/list generators.
func extractTemplatedPaths(pathTemplate string, generators []interface{}) ([]ComponentPath, map[string][]string, error) {
	var paths []ComponentPath
	clusters := make(map[string][]string)

	for _, gen := range generators {
		genMap, ok := gen.(map[string]interface{})
		if !ok {
			continue
		}

		// Handle merge generator
		if mergeGen, ok := genMap["merge"].(map[string]interface{}); ok {
			subGens, _ := mergeGen["generators"].([]interface{})
			subPaths, subClusters, err := processMergeGenerators(pathTemplate, subGens)
			if err != nil {
				return nil, nil, err
			}
			paths = append(paths, subPaths...)
			for k, v := range subClusters {
				clusters[k] = append(clusters[k], v...)
			}
			continue
		}

		// Handle direct clusters generator
		if clustersGen, ok := genMap["clusters"].(map[string]interface{}); ok {
			subPaths := resolveClusterGenerator(pathTemplate, clustersGen)
			paths = append(paths, subPaths...)
			continue
		}
	}

	return paths, clusters, nil
}

// processMergeGenerators processes generators inside a merge generator.
func processMergeGenerators(pathTemplate string, subGens []interface{}) ([]ComponentPath, map[string][]string, error) {
	// First pass: extract base values from the clusters generator
	var sourceRoot, environment, clusterDir string
	var hasClusterDir bool

	// Collect list elements for cluster overrides
	type listElement struct {
		nameNormalized string
		clusterDir     string
	}
	var listElements []listElement

	for _, sg := range subGens {
		sgMap, ok := sg.(map[string]interface{})
		if !ok {
			continue
		}

		if clustersGen, ok := sgMap["clusters"].(map[string]interface{}); ok {
			values, _ := clustersGen["values"].(map[string]interface{})
			if sr, ok := values["sourceRoot"].(string); ok {
				sourceRoot = sr
			}
			if env, ok := values["environment"].(string); ok {
				environment = env
			}
			if cd, ok := values["clusterDir"].(string); ok {
				clusterDir = cd
				hasClusterDir = true
			}
		}

		if listGen, ok := sgMap["list"].(map[string]interface{}); ok {
			elements, _ := listGen["elements"].([]interface{})
			for _, elem := range elements {
				elemMap, ok := elem.(map[string]interface{})
				if !ok {
					continue
				}
				le := listElement{}
				if nn, ok := elemMap["nameNormalized"].(string); ok {
					le.nameNormalized = nn
				}
				if cd, ok := elemMap["values.clusterDir"].(string); ok {
					le.clusterDir = cd
				}
				listElements = append(listElements, le)
			}
		}
	}

	var paths []ComponentPath
	clusters := make(map[string][]string)

	// Resolve the path template
	if sourceRoot == "" {
		// Can't resolve without sourceRoot
		return nil, nil, nil
	}

	// Check if the path uses {{nameNormalized}} — treat as prefix
	if strings.Contains(pathTemplate, "{{nameNormalized}}") {
		basePath := sourceRoot + "/" + environment + "/"
		paths = append(paths, ComponentPath{
			Path: basePath,
		})

		// Also add cluster-specific paths from list elements
		for _, le := range listElements {
			dir := le.clusterDir
			if dir == "" {
				dir = le.nameNormalized
			}
			if dir != "" {
				p := sourceRoot + "/" + environment + "/" + dir
				paths = append(paths, ComponentPath{
					Path:       p,
					ClusterDir: dir,
				})
				clusters[le.nameNormalized] = append(clusters[le.nameNormalized], p)
			}
		}
		return paths, clusters, nil
	}

	// Standard template: {{values.sourceRoot}}/{{values.environment}}/{{values.clusterDir}}
	// or: {{values.sourceRoot}}/{{values.environment}}

	// Base path (from clusters generator defaults)
	if hasClusterDir && clusterDir == "" {
		// clusterDir is explicitly empty string — the path resolves to sourceRoot/environment/
		basePath := sourceRoot + "/" + environment
		paths = append(paths, ComponentPath{
			Path: basePath,
		})
	} else if hasClusterDir && clusterDir != "" {
		basePath := sourceRoot + "/" + environment + "/" + clusterDir
		paths = append(paths, ComponentPath{
			Path:       basePath,
			ClusterDir: clusterDir,
		})
	} else {
		// No clusterDir in template
		basePath := sourceRoot + "/" + environment
		paths = append(paths, ComponentPath{
			Path: basePath,
		})
	}

	// Add cluster-specific overrides from list elements
	for _, le := range listElements {
		if le.clusterDir != "" {
			p := sourceRoot + "/" + environment + "/" + le.clusterDir
			paths = append(paths, ComponentPath{
				Path:       p,
				ClusterDir: le.clusterDir,
			})
			if le.nameNormalized != "" {
				clusters[le.nameNormalized] = append(clusters[le.nameNormalized], p)
			}
		}
	}

	return paths, clusters, nil
}

// resolveClusterGenerator processes a direct clusters generator (without merge).
func resolveClusterGenerator(pathTemplate string, clustersGen map[string]interface{}) []ComponentPath {
	values, _ := clustersGen["values"].(map[string]interface{})
	sourceRoot, _ := values["sourceRoot"].(string)
	environment, _ := values["environment"].(string)

	if sourceRoot == "" {
		return nil
	}

	basePath := sourceRoot + "/" + environment
	return []ComponentPath{{
		Path: basePath,
	}}
}
