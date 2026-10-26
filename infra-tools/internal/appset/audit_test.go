//go:build realrepo

package appset

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/kustomize"
)

// repoRoot is the path to the infra-deployments checkout, relative to this
// package directory.
const repoRoot = "../../.."

// knownGaps lists ApplicationSets whose component paths cannot be extracted by
// the current parser, keyed by "<overlay>/<appset>" with the reason. Every
// entry is a silent detection miss: changes under the real source path are
// ignored by env-detector and render-diff. Entries must be removed as the
// parser learns the corresponding generator shape — an entry that no longer
// fails is reported as stale so the list cannot rot.
//
// TODO(render-diff-audit step 2): implement top-level list generators and
// multi-source templates, then remove the entries below.
var knownGaps = map[string]string{
	"rd-production/konflux-verifications-rd":                    "top-level list generator is not supported",
	"development/tracing-workload-otel-collector":               "multi-source template (sources) is not supported",
	"development-operator/tracing-workload-otel-collector":      "multi-source template (sources) is not supported",
	"konflux-public-production/tracing-workload-otel-collector": "multi-source template (sources) is not supported",
	"konflux-public-staging/tracing-workload-otel-collector":    "multi-source template (sources) is not supported",
	"production-downstream/tracing-workload-otel-collector":     "multi-source template (sources) is not supported",
	"rd-dev/tracing-workload-otel-collector":                    "multi-source template (sources) is not supported",
	"staging-downstream/tracing-workload-otel-collector":        "multi-source template (sources) is not supported",
}

// knownUnresolved lists ApplicationSets whose rendered path template contains
// placeholders that interpolatePath does not know about.
var knownUnresolved = map[string]string{
	"konflux-public-production/k8s-groups": "{{values.useCaseDir}} is not interpolated",
	"konflux-public-staging/k8s-groups":    "{{values.useCaseDir}} is not interpolated",
	"production-downstream/k8s-groups":     "{{values.useCaseDir}} is not interpolated",
	"staging-downstream/k8s-groups":        "{{values.useCaseDir}} is not interpolated",
}

type auditFinding struct {
	key    string
	detail string
}

// TestRealRepoApplicationSets renders every ArgoCD overlay and checks that each
// ApplicationSet template is turned into at least one usable component path.
// It catches parser gaps that would otherwise degrade silently into
// "environment/none".
//
// Run with: make audit
func TestRealRepoApplicationSets(t *testing.T) {
	overlaysDir := filepath.Join(repoRoot, "argo-cd-apps", "overlays")
	entries, err := os.ReadDir(overlaysDir)
	if err != nil {
		t.Fatalf("reading overlays dir: %v", err)
	}

	var gaps, unresolved, missing []auditFinding

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		overlay := entry.Name()
		rendered, err := kustomize.Build(filepath.Join(overlaysDir, overlay))
		if err != nil {
			t.Fatalf("building overlay %s: %v", overlay, err)
		}

		for _, doc := range decodeDocuments(t, overlay, rendered) {
			name := appSetName(doc)
			if name == "" {
				continue
			}
			key := overlay + "/" + name

			paths, _, err := extractFromAppSet(doc)
			if err != nil {
				t.Errorf("%s: extractFromAppSet failed: %v", key, err)
				continue
			}

			spec, _ := doc["spec"].(map[string]interface{})
			tmpl, _ := spec["template"].(map[string]interface{})
			tmplSpec, _ := tmpl["spec"].(map[string]interface{})
			hasTemplatePath := templateHasPath(tmplSpec)

			switch {
			case hasTemplatePath && len(paths) == 0:
				gaps = append(gaps, auditFinding{key, "template has a source path but no component path was extracted"})
			case len(paths) == 0:
				gaps = append(gaps, auditFinding{key, "no source path in template (multi-source or chart-only)"})
			}

			for _, p := range paths {
				if p.Path == "" {
					gaps = append(gaps, auditFinding{key, "empty component path"})
					continue
				}
				if strings.Contains(p.Path, "{{") {
					unresolved = append(unresolved, auditFinding{key, p.Path})
					continue
				}
				abs := filepath.Join(repoRoot, filepath.FromSlash(p.Path))
				if info, statErr := os.Stat(abs); statErr != nil || !info.IsDir() {
					missing = append(missing, auditFinding{key, p.Path})
				}
			}
		}
	}

	checkFindings(t, "ApplicationSets with no extracted component path", gaps, knownGaps)
	checkFindings(t, "component paths with unresolved placeholders", unresolved, knownUnresolved)

	// Paths whose directory is absent are informational: ApplicationSets often
	// list every cluster while only some clusters need a cluster-specific
	// directory, so this is expected and not a parser gap.
	if len(missing) > 0 {
		t.Logf("note: %d extracted paths point at directories that do not exist on disk", len(missing))
		for i, f := range missing {
			if i == 3 {
				t.Logf("  ... and %d more", len(missing)-3)
				break
			}
			t.Logf("  - %s: %s", f.key, f.detail)
		}
	}
}

// checkFindings compares the findings produced by the audit with the known-gap
// allow-list. Unexpected findings fail the test; stale allow-list entries are
// reported so the list stays accurate.
func checkFindings(t *testing.T, title string, found []auditFinding, known map[string]string) {
	t.Helper()

	seen := make(map[string]bool)
	for _, f := range found {
		seen[f.key] = true
		if _, ok := known[f.key]; !ok {
			t.Errorf("%s: %s: %s (not in the allow-list)", title, f.key, f.detail)
		}
	}
	for key, reason := range known {
		if !seen[key] {
			t.Errorf("stale allow-list entry %q (%s): the parser now handles it, remove it from the list", key, reason)
		}
	}

	if len(found) > 0 {
		sort.Slice(found, func(i, j int) bool { return found[i].key < found[j].key })
		t.Logf("%s (%d, all allow-listed):", title, len(found))
		for _, f := range found {
			t.Logf("  - %s: %s", f.key, f.detail)
		}
	}
}

// decodeDocuments decodes a rendered YAML stream into document maps.
func decodeDocuments(t *testing.T, overlay string, rendered []byte) []map[string]interface{} {
	t.Helper()

	var docs []map[string]interface{}
	decoder := yaml.NewDecoder(strings.NewReader(string(rendered)))
	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err != nil {
			if strings.Contains(err.Error(), "EOF") {
				break
			}
			t.Fatalf("decoding rendered YAML for overlay %s: %v", overlay, err)
		}
		if doc == nil {
			continue
		}
		if kind, _ := doc["kind"].(string); kind != "ApplicationSet" {
			continue
		}
		docs = append(docs, doc)
	}
	return docs
}

// appSetName returns metadata.name of a document, or "" when absent.
func appSetName(doc map[string]interface{}) string {
	md, _ := doc["metadata"].(map[string]interface{})
	name, _ := md["name"].(string)
	return name
}

// templateHasPath reports whether the template declares a local source path,
// either single-source or as one of several sources.
func templateHasPath(tmplSpec map[string]interface{}) bool {
	if source, ok := tmplSpec["source"].(map[string]interface{}); ok {
		if p, _ := source["path"].(string); p != "" {
			return true
		}
	}
	sources, _ := tmplSpec["sources"].([]interface{})
	for _, s := range sources {
		if source, ok := s.(map[string]interface{}); ok {
			if p, _ := source["path"].(string); p != "" {
				return true
			}
		}
	}
	return false
}
