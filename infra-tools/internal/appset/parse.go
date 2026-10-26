// Package appset extracts environment-to-path mappings and cluster names from
// rendered ApplicationSet YAML produced by kustomize build.
package appset

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComponentPath represents a single path extracted from an ApplicationSet with
// an optional cluster directory.  The environment is not stored here; it is
// determined by the overlay that produced the ApplicationSet.
type ComponentPath struct {
	// Path is the resolved filesystem path (relative to repo root) that the
	// ApplicationSet points to. It is empty when the ApplicationSet only
	// references files through ExtraFiles (e.g. a chart-only source).
	Path string
	// ClusterDir is non-empty when the ApplicationSet targets a specific cluster
	// subdirectory (e.g. "stone-prod-p01").
	ClusterDir string
	// ExtraFiles are individual files the ApplicationSet consumes from this
	// repository without deploying their directory, such as Helm values files
	// referenced as "$values/<path>" from a multi-source template. They are
	// matched by exact path.
	ExtraFiles []string
	// PrefixOnly is true when the path template could not be fully resolved
	// (unknown placeholder). Path then holds the longest static prefix, and
	// callers must match changed files by prefix instead of by dependency tree.
	PrefixOnly bool
}

// ParseResult holds the extracted data from all ApplicationSets in one overlay.
type ParseResult struct {
	// Paths contains all resolved component/config paths.
	Paths []ComponentPath
	// Clusters maps cluster names found in list generator elements (or in a
	// clusters generator's name) to the component paths for that cluster.
	Clusters map[string][]string
}

// AppSetsByName parses rendered YAML (multi-document) and returns all
// ApplicationSet resources keyed by metadata.name for change detection.
func AppSetsByName(renderedYAML []byte) (map[string]map[string]interface{}, error) {
	result := make(map[string]map[string]interface{})
	if len(renderedYAML) == 0 {
		return result, nil
	}
	decoder := yaml.NewDecoder(bytes.NewReader(renderedYAML))
	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding YAML document: %w", err)
		}
		if doc == nil {
			continue
		}
		if kind, _ := doc["kind"].(string); kind != "ApplicationSet" {
			continue
		}
		name := ""
		if md, ok := doc["metadata"].(map[string]interface{}); ok {
			name, _ = md["name"].(string)
		}
		if name != "" {
			result[name] = doc
		}
	}
	return result, nil
}

// EnvironmentFromAppSet extracts the environment value from an ApplicationSet's
// generator configuration. It looks for a clusters generator (direct or nested
// inside a merge generator) with a values.environment field.
// Returns the environment string and true if an explicit environment is found.
func EnvironmentFromAppSet(doc map[string]interface{}) (string, bool) {
	spec, ok := doc["spec"].(map[string]interface{})
	if !ok {
		return "", false
	}
	generators, _ := spec["generators"].([]interface{})
	for _, gen := range generators {
		genMap, ok := gen.(map[string]interface{})
		if !ok {
			continue
		}
		if env, ok := envFromClusters(genMap["clusters"]); ok {
			return env, true
		}
		merge, _ := genMap["merge"].(map[string]interface{})
		subGens, _ := merge["generators"].([]interface{})
		for _, sg := range subGens {
			sgMap, ok := sg.(map[string]interface{})
			if !ok {
				continue
			}
			if env, ok := envFromClusters(sgMap["clusters"]); ok {
				return env, true
			}
		}
	}
	return "", false
}

func envFromClusters(v interface{}) (string, bool) {
	clusters, ok := v.(map[string]interface{})
	if !ok {
		return "", false
	}
	values, _ := clusters["values"].(map[string]interface{})
	env, _ := values["environment"].(string)
	if env == "" {
		return "", false
	}
	return env, true
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
			return nil, fmt.Errorf("extracting from ApplicationSet %s: %w", appSetName(doc), err)
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

	sources := templateSources(tmplSpec)
	if len(sources) == 0 {
		return nil, nil, nil
	}

	name := appSetName(doc)
	values := collectGeneratorValues(spec["generators"])
	for _, unsupported := range values.unsupported {
		slog.Warn("unsupported ApplicationSet generator, its component paths are not tracked",
			"appset", name, "generator", unsupported)
	}
	if len(values.unsupported) > 0 {
		return nil, nil, nil
	}

	return resolveSources(name, sources, values.sets())
}

// templateSource is one entry of spec.template.spec.source(s) reduced to the
// parts that reference files in this repository.
type templateSource struct {
	// pathTemplate is the (possibly templated) deployed directory, or "" when
	// the source only contributes files.
	pathTemplate string
	// extraFileTemplates are templated individual files consumed by the source
	// (e.g. Helm values files), with the "$values/" prefix removed.
	extraFileTemplates []string
}

// templateSources extracts the local references from a template spec. Both the
// single-source (source) and multi-source (sources) forms are supported.
func templateSources(tmplSpec map[string]interface{}) []templateSource {
	var raw []map[string]interface{}
	if source, ok := tmplSpec["source"].(map[string]interface{}); ok {
		raw = append(raw, source)
	}
	if sources, ok := tmplSpec["sources"].([]interface{}); ok {
		for _, s := range sources {
			if source, ok := s.(map[string]interface{}); ok {
				raw = append(raw, source)
			}
		}
	}

	var out []templateSource
	for _, source := range raw {
		ts := templateSource{}
		ts.pathTemplate, _ = source["path"].(string)
		ts.extraFileTemplates = helmValueFiles(source)
		if ts.pathTemplate == "" && len(ts.extraFileTemplates) == 0 {
			continue
		}
		out = append(out, ts)
	}
	return out
}

// helmValueFiles returns the valueFiles of a Helm source that live in this
// repository. They are referenced as "$values/<path>"; other value files are
// either chart-relative or baked into the chart and cannot be resolved here.
func helmValueFiles(source map[string]interface{}) []string {
	helm, ok := source["helm"].(map[string]interface{})
	if !ok {
		return nil
	}
	valueFiles, _ := helm["valueFiles"].([]interface{})
	var out []string
	for _, vf := range valueFiles {
		s, _ := vf.(string)
		if after, found := strings.CutPrefix(s, "$values/"); found && after != "" {
			out = append(out, after)
		}
	}
	return out
}

// generatorValues is the set of template values known for one generated
// Application, keyed by the name used in {{...}} placeholders (that is, without
// the "values." prefix).
type generatorValues map[string]string

// valueSet is a single combination of generator values, together with the
// cluster it belongs to (empty for the defaults-only combination).
type valueSet struct {
	values      generatorValues
	clusterName string
}

// collectedValues holds everything gathered from an ApplicationSet's generators.
type collectedValues struct {
	defaults    generatorValues
	elements    []valueSet
	unsupported []string
}

// sets returns the value combinations to interpolate: the generator defaults
// (when any) followed by each list element merged over the defaults. With no
// generators at all it returns a single empty combination so static paths are
// still resolved.
func (c collectedValues) sets() []valueSet {
	var sets []valueSet
	if len(c.defaults) > 0 {
		sets = append(sets, valueSet{values: c.defaults})
	}
	for _, e := range c.elements {
		sets = append(sets, valueSet{values: mergeValues(c.defaults, e.values), clusterName: e.clusterName})
	}
	if len(sets) == 0 {
		sets = append(sets, valueSet{values: generatorValues{}})
	}
	return sets
}

// collectGeneratorValues walks the generator tree (including merge and nested
// merge generators), gathering cluster defaults and list elements.
func collectGeneratorValues(generators interface{}) collectedValues {
	out := collectedValues{defaults: generatorValues{}}
	gens, _ := generators.([]interface{})
	collectGenerators(gens, &out)
	return out
}

func collectGenerators(gens []interface{}, out *collectedValues) {
	for _, gen := range gens {
		genMap, ok := gen.(map[string]interface{})
		if !ok {
			continue
		}

		if clustersGen, ok := genMap["clusters"].(map[string]interface{}); ok {
			values, _ := clustersGen["values"].(map[string]interface{})
			maps.Copy(out.defaults, valuesFromMap(values))
			continue
		}

		if mergeGen, ok := genMap["merge"].(map[string]interface{}); ok {
			subGens, _ := mergeGen["generators"].([]interface{})
			collectGenerators(subGens, out)
			continue
		}

		if listGen, ok := genMap["list"].(map[string]interface{}); ok {
			elements, _ := listGen["elements"].([]interface{})
			for _, elem := range elements {
				elemMap, ok := elem.(map[string]interface{})
				if !ok {
					continue
				}
				out.elements = append(out.elements, valueSet{
					values:      valuesFromElement(elemMap),
					clusterName: clusterNameFromElement(elemMap),
				})
			}
			continue
		}

		// matrix, git, scm, clusterDecisionResource, pullRequest, plugin, ...
		// are not modelled: record them so the caller can warn instead of
		// silently returning no paths.
		keys := make([]string, 0, len(genMap))
		for k := range genMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.unsupported = append(out.unsupported, keys...)
	}
}

// valuesFromMap converts a generator "values" map into template values.
func valuesFromMap(values map[string]interface{}) generatorValues {
	out := make(generatorValues, len(values))
	for k, v := range values {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// valuesFromElement converts a list generator element into template values.
// Both "values.<key>" entries and "nameNormalized"/"name" are supported.
func valuesFromElement(elem map[string]interface{}) generatorValues {
	out := make(generatorValues, len(elem))
	for k, v := range elem {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if key, found := strings.CutPrefix(k, "values."); found {
			out[key] = s
			continue
		}
		switch k {
		case "nameNormalized", "name":
			out[k] = s
		}
	}
	return out
}

// clusterNameFromElement returns the cluster a list element refers to, or "".
func clusterNameFromElement(elem map[string]interface{}) string {
	if nn, ok := elem["nameNormalized"].(string); ok && nn != "" {
		return nn
	}
	name, _ := elem["name"].(string)
	return name
}

// mergeValues returns base overlaid with overrides.
func mergeValues(base, overrides generatorValues) generatorValues {
	out := make(generatorValues, len(base)+len(overrides))
	maps.Copy(out, base)
	maps.Copy(out, overrides)
	return out
}

// placeholderRe matches ApplicationSet template placeholders, with or without
// the "values." prefix.
var placeholderRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.\-]+)\s*\}\}`)

// interpolatePath substitutes placeholders in an ApplicationSet source path.
// The second return value is false when at least one placeholder could not be
// resolved, in which case the returned path still contains the placeholder.
func interpolatePath(pathTemplate string, values generatorValues) (string, bool) {
	resolved := true
	out := placeholderRe.ReplaceAllStringFunc(pathTemplate, func(match string) string {
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
		key = strings.TrimPrefix(key, "values.")
		if v, ok := values[key]; ok {
			return v
		}
		resolved = false
		return match
	})
	if !resolved {
		return out, false
	}
	return path.Clean(strings.TrimSuffix(out, "/")), true
}

// staticPrefix returns the longest directory prefix of a path template that is
// free of placeholders. It is used to keep tracking a path that cannot be fully
// resolved, instead of dropping the ApplicationSet entirely.
func staticPrefix(pathTemplate string) string {
	loc := placeholderRe.FindStringIndex(pathTemplate)
	if loc == nil {
		return path.Clean(strings.TrimSuffix(pathTemplate, "/"))
	}
	prefix := pathTemplate[:loc[0]]
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		prefix = prefix[:i]
	} else {
		prefix = ""
	}
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return ""
	}
	return path.Clean(prefix)
}

// resolveSources turns the template sources of one ApplicationSet into
// ComponentPaths, one per generator value combination.
func resolveSources(name string, sources []templateSource, sets []valueSet) ([]ComponentPath, map[string][]string, error) {
	var paths []ComponentPath
	clusters := make(map[string][]string)
	seen := make(map[string]bool)

	for _, source := range sources {
		for _, set := range sets {
			cp := ComponentPath{ClusterDir: set.values["clusterDir"]}

			if source.pathTemplate != "" {
				resolved, ok := interpolatePath(source.pathTemplate, set.values)
				if ok {
					cp.Path = resolved
				} else {
					// interpolatePath returns the template with every resolvable
					// placeholder substituted, so the static prefix is computed
					// on the partially resolved path.
					cp.Path = staticPrefix(resolved)
					cp.PrefixOnly = true
					if cp.Path == "" {
						slog.Warn("ApplicationSet path template cannot be resolved and has no static prefix",
							"appset", name, "template", source.pathTemplate)
						continue
					}
					// Expected for templates keyed on a list element field (for
					// example {{nameNormalized}}): the defaults-only combination
					// cannot be resolved, and the prefix keeps the shared
					// dependencies of the whole environment tracked.
					slog.Debug("ApplicationSet path template has unresolved placeholders, falling back to prefix matching",
						"appset", name, "template", source.pathTemplate, "prefix", cp.Path)
				}
			}

			for _, fileTemplate := range source.extraFileTemplates {
				resolved, ok := interpolatePath(fileTemplate, set.values)
				if !ok {
					slog.Warn("ApplicationSet value file template has unresolved placeholders, skipping",
						"appset", name, "template", fileTemplate)
					continue
				}
				cp.ExtraFiles = append(cp.ExtraFiles, resolved)
			}

			if cp.Path == "" && len(cp.ExtraFiles) == 0 {
				continue
			}
			key := cp.Path + "|" + cp.ClusterDir + "|" + strings.Join(cp.ExtraFiles, ",") + "|" + strconv.FormatBool(cp.PrefixOnly)
			if seen[key] {
				continue
			}
			seen[key] = true
			paths = append(paths, cp)

			// Only a fully resolved path can be attributed to a cluster; a
			// prefix is shared by several clusters and would over-match.
			if set.clusterName != "" && cp.Path != "" && !cp.PrefixOnly {
				clusters[set.clusterName] = append(clusters[set.clusterName], cp.Path)
			}
		}
	}

	return paths, clusters, nil
}

// appSetName returns metadata.name of an ApplicationSet document, or "unknown".
func appSetName(doc map[string]interface{}) string {
	if md, ok := doc["metadata"].(map[string]interface{}); ok {
		if n, ok := md["name"].(string); ok && n != "" {
			return n
		}
	}
	return "unknown"
}
