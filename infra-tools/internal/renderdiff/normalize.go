package renderdiff

import (
	"bytes"
	"io"
	"log/slog"
	"sort"

	"gopkg.in/yaml.v3"
)

// resourceKey holds the identity fields used to sort Kubernetes resources.
type resourceKey struct {
	apiVersion string
	kind       string
	namespace  string
	name       string
}

// normalizeYAML sorts a multi-document YAML stream by resource identity
// (apiVersion, kind, namespace, name) to minimize diff noise from resource
// reordering across kustomize builds. If parsing fails, the original input
// is returned unchanged.
func normalizeYAML(input []byte) []byte {
	if len(input) == 0 {
		return input
	}

	type doc struct {
		key  resourceKey
		node yaml.Node
	}

	var docs []doc
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Debug("YAML normalization: parse error, returning original", "err", err)
			return input
		}
		docs = append(docs, doc{key: extractKey(&node), node: node})
	}

	if len(docs) <= 1 {
		return input
	}

	sort.SliceStable(docs, func(i, j int) bool {
		a, b := docs[i].key, docs[j].key
		if a.apiVersion != b.apiVersion {
			return a.apiVersion < b.apiVersion
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		if a.namespace != b.namespace {
			return a.namespace < b.namespace
		}
		return a.name < b.name
	})

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	for i := range docs {
		if err := encoder.Encode(&docs[i].node); err != nil {
			slog.Debug("YAML normalization: encode error, returning original", "err", err)
			return input
		}
	}
	_ = encoder.Close()
	return buf.Bytes()
}

// extractKey pulls apiVersion, kind, namespace, and name from a YAML document node.
func extractKey(node *yaml.Node) resourceKey {
	if node == nil || node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return resourceKey{}
	}
	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return resourceKey{}
	}

	var key resourceKey
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		v := mapping.Content[i+1]
		if k.Kind != yaml.ScalarNode {
			continue
		}
		switch k.Value {
		case "apiVersion":
			if v.Kind == yaml.ScalarNode {
				key.apiVersion = v.Value
			}
		case "kind":
			if v.Kind == yaml.ScalarNode {
				key.kind = v.Value
			}
		case "metadata":
			if v.Kind == yaml.MappingNode {
				for j := 0; j+1 < len(v.Content); j += 2 {
					mk := v.Content[j]
					mv := v.Content[j+1]
					if mk.Kind != yaml.ScalarNode || mv.Kind != yaml.ScalarNode {
						continue
					}
					switch mk.Value {
					case "name":
						key.name = mv.Value
					case "namespace":
						key.namespace = mv.Value
					}
				}
			}
		}
	}
	return key
}
