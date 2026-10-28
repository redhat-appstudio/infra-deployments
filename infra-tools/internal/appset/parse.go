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
	// referenced as "$<ref>/<path>" from a multi-source template, or relative
	// to the source path. They are matched by exact path.
	ExtraFiles []string
	// PrefixOnly is true when the path template could not be fully resolved
	// (unknown placeholder). Path then holds the longest static prefix, and
	// callers must match changed files by prefix, and by the dependency trees
	// of the kustomizations under that prefix.
	PrefixOnly bool
	// MatchAll is true when a template could not be resolved and has no static
	// prefix either, so nothing in the repository can be attributed to it. Path
	// is then empty, and callers must treat any change as affecting the
	// environment (fail closed) instead of ignoring the ApplicationSet or
	// aborting detection.
	MatchAll bool
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
	name := appSetName(doc)

	// spec.templatePatch (goTemplate) is applied over spec.template and can
	// rewrite source.path(s) or Helm value files, so nothing in spec.template
	// can be trusted to locate what is deployed.
	if _, found := spec["templatePatch"]; found {
		slog.Warn("ApplicationSet has a templatePatch, which may replace the source paths; treating the environment as affected by any change",
			"appset", name)
		return []ComponentPath{{MatchAll: true}}, map[string][]string{}, nil
	}

	// Generators are inspected before spec.template so that a generator-level
	// template still fails closed when spec.template has no local source.
	values := collectGeneratorValues(spec["generators"])
	var sources []templateSource
	if tmpl, ok := spec["template"].(map[string]interface{}); ok {
		if tmplSpec, ok := tmpl["spec"].(map[string]interface{}); ok {
			sources = templateSources(tmplSpec)
		}
	}
	if len(sources) == 0 {
		if values.templateOverride() {
			slog.Warn("ApplicationSet generator carries its own template, which may replace the source paths; treating the environment as affected by any change",
				"appset", name)
			return []ComponentPath{{MatchAll: true}}, map[string][]string{}, nil
		}
		return nil, nil, nil
	}

	for _, unsupported := range values.unsupported() {
		slog.Warn("unsupported ApplicationSet generator, falling back to static paths and prefixes for it",
			"appset", name, "generator", unsupported)
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
	// (e.g. Helm values files), relative to the repository root: the
	// "$<ref>/" prefix is removed and chart-relative files are joined to the
	// source path.
	extraFileTemplates []string
	// unattributedFiles are value files referenced through a "$<ref>/" prefix
	// whose ref no source declares. They cannot be located, so the caller
	// must fail closed for them.
	unattributedFiles []string
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

	// Value files of a multi-source template are referenced as "$<ref>/<path>",
	// where <ref> is the "ref" name of any source, not only "values".
	refs := make(map[string]bool)
	for _, source := range raw {
		if ref, _ := source["ref"].(string); ref != "" {
			refs[ref] = true
		}
	}

	var out []templateSource
	for _, source := range raw {
		ts := templateSource{}
		ts.pathTemplate, _ = source["path"].(string)
		ts.extraFileTemplates, ts.unattributedFiles = helmValueFiles(source, ts.pathTemplate, refs)
		if ts.pathTemplate == "" && len(ts.extraFileTemplates) == 0 && len(ts.unattributedFiles) == 0 {
			continue
		}
		out = append(out, ts)
	}
	return out
}

// helmValueFiles returns the valueFiles of a Helm source that live in this
// repository: "$<ref>/<path>" files of a declared ref, and chart-relative files
// of a source that has a path in this repository (joined to that path). Value
// files under an undeclared ref are returned separately as unattributed.
// Chart-relative files of a source without a path live in a remote chart, and
// remote URLs are not in this repository either; both are only logged.
func helmValueFiles(source map[string]interface{}, pathTemplate string, refs map[string]bool) (files, unattributed []string) {
	helm, ok := source["helm"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	valueFiles, _ := helm["valueFiles"].([]interface{})
	for _, vf := range valueFiles {
		s, _ := vf.(string)
		switch {
		case s == "":
			continue
		case strings.HasPrefix(s, "$"):
			ref, rest, _ := strings.Cut(s[1:], "/")
			if refs[ref] && rest != "" {
				files = append(files, rest)
			} else {
				unattributed = append(unattributed, s)
			}
		case strings.Contains(s, "://"):
			slog.Warn("Helm value file is a remote URL, not tracked", "valueFile", s)
		case pathTemplate != "":
			files = append(files, pathTemplate+"/"+s)
		default:
			slog.Warn("Helm value file is relative to a remote chart, not tracked", "valueFile", s)
		}
	}
	return files, unattributed
}

// generatorValues is the set of template parameters known for one generated
// Application, keyed by the exact name used in {{...}} placeholders. ArgoCD
// treats "{{values.X}}" and "{{X}}" as different parameters, so generator
// "values" are stored as "values.<key>" and list element parameters under
// their literal names.
type generatorValues map[string]string

// valueSet is a single combination of generator values, together with the
// cluster it belongs to (empty for the defaults-only combination).
type valueSet struct {
	values      generatorValues
	clusterName string
	// defaults holds the cluster defaults a merged list element overrides. It
	// is non-nil only for elements merged into a clusters generator, and is
	// used to attribute the element's cluster only when the element actually
	// changes the deployed path.
	defaults generatorValues
	// prefixOnly is set when the values are incomplete because an unmodelled
	// generator may supply or override any of them: templates resolved from
	// this set are only tracked by the static prefix of the raw template.
	prefixOnly bool
	// matchAll is set when a generator-level template may replace the source
	// paths entirely, so nothing derived from spec.template can be trusted.
	matchAll bool
}

// valueGroup holds the values produced by one top-level generator (and, for a
// merge generator, everything nested in it). ArgoCD combines top-level
// generators as a union, so values never flow from one group to another.
type valueGroup struct {
	defaults    generatorValues
	elements    []valueSet
	hasClusters bool
	unsupported []string
	// templateOverride is set when a generator in the group carries its own
	// "template", which ArgoCD merges over spec.template and which can
	// replace source.path(s) completely.
	templateOverride bool
}

// sets returns the value combinations of one group: the cluster defaults
// followed by each list element merged over the defaults.
func (g valueGroup) sets() []valueSet {
	if g.templateOverride {
		return []valueSet{{matchAll: true}}
	}
	// An unmodelled generator may supply or override any value of its group,
	// so no known value can be trusted to locate the deployed directory: a
	// static path still resolves exactly and a templated one is tracked by
	// the static prefix of the raw template (MatchAll when it has none). A
	// single set is enough since the values are not used.
	if len(g.unsupported) > 0 {
		return []valueSet{{values: g.defaults, prefixOnly: true}}
	}
	var sets []valueSet
	// A clusters generator generates one Application per selected cluster,
	// even when it carries no values and even when list elements are merged
	// over it (they only override the clusters they match), so its defaults
	// always form a combination of their own.
	if g.hasClusters || len(g.defaults) > 0 {
		sets = append(sets, valueSet{values: g.defaults})
	}
	for _, e := range g.elements {
		set := valueSet{values: mergeValues(g.defaults, e.values), clusterName: e.clusterName}
		if g.hasClusters {
			set.defaults = g.defaults
		}
		sets = append(sets, set)
	}
	return sets
}

// collectedValues holds everything gathered from an ApplicationSet's generators,
// one group per top-level generator.
type collectedValues struct {
	groups []valueGroup
}

// unsupported returns the unmodelled generator types found in any group.
func (c collectedValues) unsupported() []string {
	n := 0
	for _, g := range c.groups {
		n += len(g.unsupported)
	}
	out := make([]string, 0, n)
	for _, g := range c.groups {
		out = append(out, g.unsupported...)
	}
	return out
}

// templateOverride reports whether any group carries a generator-level
// template.
func (c collectedValues) templateOverride() bool {
	for _, g := range c.groups {
		if g.templateOverride {
			return true
		}
	}
	return false
}

// sets returns the value combinations to interpolate: the union of the value
// sets of every top-level generator. With no generators at all it returns a
// single empty combination so static paths are still resolved.
func (c collectedValues) sets() []valueSet {
	var sets []valueSet
	for _, g := range c.groups {
		sets = append(sets, g.sets()...)
	}
	if len(sets) == 0 {
		sets = append(sets, valueSet{values: generatorValues{}})
	}
	return sets
}

// collectGeneratorValues walks the generator tree (including merge and nested
// merge generators), gathering cluster defaults and list elements separately
// for each top-level generator.
func collectGeneratorValues(generators interface{}) collectedValues {
	var out collectedValues
	gens, _ := generators.([]interface{})
	for _, gen := range gens {
		group := valueGroup{defaults: generatorValues{}}
		collectGenerators([]interface{}{gen}, &group)
		out.groups = append(out.groups, group)
	}
	return out
}

func collectGenerators(gens []interface{}, out *valueGroup) {
	for _, gen := range gens {
		genMap, ok := gen.(map[string]interface{})
		if !ok {
			continue
		}

		// Any generator may carry its own template, merged over spec.template
		// by ArgoCD. It can move source.path(s) anywhere, so the group cannot
		// be tracked from spec.template at all.
		kinds := make([]string, 0, len(genMap))
		for kind := range genMap {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		for _, kind := range kinds {
			if bodyMap, ok := genMap[kind].(map[string]interface{}); ok {
				if _, found := bodyMap["template"]; found {
					out.unsupported = append(out.unsupported, kind+".template")
					out.templateOverride = true
				}
			}
		}

		if clustersGen, ok := genMap["clusters"].(map[string]interface{}); ok {
			values, _ := clustersGen["values"].(map[string]interface{})
			maps.Copy(out.defaults, valuesFromMap(values))
			out.hasClusters = true
			continue
		}

		if mergeGen, ok := genMap["merge"].(map[string]interface{}); ok {
			subGens, _ := mergeGen["generators"].([]interface{})
			collectGenerators(subGens, out)
			continue
		}

		// A matrix is not modelled (its children are combined as a cartesian
		// product), but the values its modelled children carry are still
		// known and give the group a usable static prefix.
		if matrixGen, ok := genMap["matrix"].(map[string]interface{}); ok {
			out.unsupported = append(out.unsupported, "matrix")
			subGens, _ := matrixGen["generators"].([]interface{})
			collectGenerators(subGens, out)
			continue
		}

		if listGen, ok := genMap["list"].(map[string]interface{}); ok {
			// Only "elements" is read. Elements supplied through elementsYaml
			// are not modelled and may override any value: the group falls
			// back to the static prefix of the raw template. (A generator-level
			// template is handled above.) An empty "elements" list is
			// legitimate and not unsupported.
			if _, found := listGen["elementsYaml"]; found {
				out.unsupported = append(out.unsupported, "list.elementsYaml")
			}
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

		// git, scm, clusterDecisionResource, pullRequest, plugin, ...
		// are not modelled: record them so the caller can warn and the group
		// falls back to static paths and prefixes.
		keys := make([]string, 0, len(genMap))
		for k := range genMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.unsupported = append(out.unsupported, keys...)
	}
}

// valuesFromMap converts a generator "values" map into template values, keyed
// as the "{{values.<key>}}" placeholders that reference them.
func valuesFromMap(values map[string]interface{}) generatorValues {
	out := make(generatorValues, len(values))
	for k, v := range values {
		if s, ok := v.(string); ok {
			out["values."+k] = s
		}
	}
	return out
}

// valuesFromElement converts a list generator element into template values.
// Every string-valued key is kept under its literal name ("values.clusterDir"
// stays "values.clusterDir", "nameNormalized" stays "nameNormalized"), and a
// nested "values" map is flattened into "values.<key>" entries the way ArgoCD
// flattens element parameters. A flat "values.<key>" entry wins over the
// nested form.
func valuesFromElement(elem map[string]interface{}) generatorValues {
	out := make(generatorValues, len(elem))
	if nested, ok := elem["values"].(map[string]interface{}); ok {
		maps.Copy(out, valuesFromMap(nested))
	}
	for k, v := range elem {
		if s, ok := v.(string); ok {
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
// Template expressions placeholderRe does not understand (for example the
// Go-template form "{{ .name | lower }}") also count as unresolved, so they
// are never mistaken for a literal path.
func interpolatePath(pathTemplate string, values generatorValues) (string, bool) {
	resolved := true
	out := placeholderRe.ReplaceAllStringFunc(pathTemplate, func(match string) string {
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
		if v, ok := values[key]; ok {
			return v
		}
		resolved = false
		return match
	})
	if !resolved || strings.Contains(out, "{{") {
		return out, false
	}
	return path.Clean(out), true
}

// inRepo reports whether a cleaned path names something inside the repository
// other than its root. The root ("." or ""), absolute paths and paths escaping
// the repository ("..", "../...") can never match a repository-relative
// changed file, and walking them would scan the whole checkout or the host
// filesystem, so they must not be tracked as a path or prefix.
func inRepo(p string) bool {
	return p != "" && p != "." && p != ".." && !strings.HasPrefix(p, "../") && !path.IsAbs(p)
}

// staticPrefix returns the longest directory prefix of a path template that is
// free of template expressions. It is used to keep tracking a path that cannot
// be fully resolved, instead of dropping the ApplicationSet entirely. It
// returns "" when that prefix is not a usable path inside the repository.
func staticPrefix(pathTemplate string) string {
	prefix := pathTemplate
	if idx := strings.Index(pathTemplate, "{{"); idx >= 0 {
		prefix = pathTemplate[:idx]
		if i := strings.LastIndex(prefix, "/"); i >= 0 {
			prefix = prefix[:i]
		} else {
			prefix = ""
		}
	}
	prefix = path.Clean(prefix)
	if !inRepo(prefix) {
		return ""
	}
	return prefix
}

// resolveTemplate resolves a path or file template against one value set. When
// the template cannot be resolved exactly it returns the longest static prefix
// of the partially resolved template and prefixOnly=true; that prefix is empty
// (MatchAll) when nothing static precedes the first unresolved placeholder, or
// when the result is not a usable path inside the repository.
func resolveTemplate(template string, set valueSet) (resolvedPath string, prefixOnly bool) {
	// An unmodelled generator may override any value of the set, so only the
	// static part of the raw template can be trusted. A template without
	// placeholders is still exact: no generator can change it.
	if set.prefixOnly {
		if !strings.Contains(template, "{{") {
			if p := path.Clean(template); inRepo(p) {
				return p, false
			}
			return "", true
		}
		return staticPrefix(template), true
	}
	resolved, ok := interpolatePath(template, set.values)
	if ok {
		if inRepo(resolved) {
			return resolved, false
		}
		return "", true
	}
	// interpolatePath returns the template with every resolvable placeholder
	// substituted, so the static prefix is computed on the partially resolved
	// path.
	return staticPrefix(resolved), true
}

// resolveSources turns the template sources of one ApplicationSet into
// ComponentPaths, one per generator value combination.
//
// A template that cannot be resolved exactly is tracked by its static prefix
// (PrefixOnly). A template that has no static prefix either is reported as a
// MatchAll entry: detection then treats the environment as affected by any
// change, rather than silently ignoring the source or aborting detection for
// every overlay.
func resolveSources(name string, sources []templateSource, sets []valueSet) ([]ComponentPath, map[string][]string, error) {
	var paths []ComponentPath
	clusters := make(map[string][]string)
	seen := make(map[string]bool)
	seenCluster := make(map[string]bool)

	add := func(cp ComponentPath) {
		key := cp.Path + "|" + cp.ClusterDir + "|" + strings.Join(cp.ExtraFiles, ",") + "|" +
			strconv.FormatBool(cp.PrefixOnly) + "|" + strconv.FormatBool(cp.MatchAll)
		if seen[key] {
			return
		}
		seen[key] = true
		paths = append(paths, cp)
	}
	matchAll := func(kind, template string) {
		slog.Warn("ApplicationSet "+kind+" template cannot be resolved and has no static prefix, treating the environment as affected by any change",
			"appset", name, "template", template)
		add(ComponentPath{MatchAll: true})
	}

	for _, source := range sources {
		// A value file under a ref no source declares cannot be located.
		for _, file := range source.unattributedFiles {
			slog.Warn("ApplicationSet value file references an undeclared source ref, treating the environment as affected by any change",
				"appset", name, "valueFile", file)
			add(ComponentPath{MatchAll: true})
		}
		for _, set := range sets {
			if set.matchAll {
				slog.Warn("ApplicationSet generator carries its own template, which may replace the source paths; treating the environment as affected by any change",
					"appset", name)
				add(ComponentPath{MatchAll: true})
				continue
			}
			cp := ComponentPath{}
			// Values from an incomplete set may be overridden by an unmodelled
			// generator, so they cannot attribute a cluster directory.
			if !set.prefixOnly {
				cp.ClusterDir = set.values["values.clusterDir"]
			}

			if source.pathTemplate != "" {
				resolved, prefixOnly := resolveTemplate(source.pathTemplate, set)
				switch {
				case !prefixOnly:
					cp.Path = resolved
				case resolved == "":
					matchAll("path", source.pathTemplate)
				default:
					cp.Path = resolved
					cp.PrefixOnly = true
					slog.Warn("ApplicationSet path template has unresolved placeholders, falling back to prefix matching",
						"appset", name, "template", source.pathTemplate, "prefix", cp.Path)
				}
			}

			// Value files are matched exactly when resolved. An unresolved one
			// is handled like an unresolved path: its own prefix-only entry,
			// or a MatchAll entry when it has no static prefix.
			for _, fileTemplate := range source.extraFileTemplates {
				resolved, prefixOnly := resolveTemplate(fileTemplate, set)
				switch {
				case !prefixOnly:
					cp.ExtraFiles = append(cp.ExtraFiles, resolved)
				case resolved == "":
					matchAll("value file", fileTemplate)
				default:
					slog.Warn("ApplicationSet value file template has unresolved placeholders, falling back to prefix matching",
						"appset", name, "template", fileTemplate, "prefix", resolved)
					add(ComponentPath{Path: resolved, ClusterDir: cp.ClusterDir, PrefixOnly: true})
				}
			}

			if cp.Path == "" && len(cp.ExtraFiles) == 0 {
				continue
			}

			// Only a fully resolved path can be attributed to a cluster; a
			// prefix is shared by several clusters and would over-match. The
			// attribution happens before path deduplication: several clusters
			// may deploy the same path and each must be recorded. An element
			// merged into a clusters generator is only attributed when it
			// changes the deployed path: ArgoCD discards merge elements that
			// match no selected cluster, so an element that merely restates the
			// defaults' path (or overrides unrelated values) may name a cluster
			// of another environment.
			if set.clusterName != "" && cp.Path != "" && !cp.PrefixOnly &&
				changesPath(source.pathTemplate, set, cp.Path) {
				clusterKey := set.clusterName + "|" + cp.Path
				if !seenCluster[clusterKey] {
					seenCluster[clusterKey] = true
					clusters[set.clusterName] = append(clusters[set.clusterName], cp.Path)
				}
			}

			add(cp)
		}
	}

	return paths, clusters, nil
}

// changesPath reports whether a list element's set resolves the path template
// differently from the cluster defaults it was merged over. Sets without
// defaults (top-level list elements) always generate their own Application.
func changesPath(pathTemplate string, set valueSet, resolved string) bool {
	if set.defaults == nil {
		return true
	}
	defaultPath, prefixOnly := resolveTemplate(pathTemplate, valueSet{values: set.defaults})
	return prefixOnly || defaultPath != resolved
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
