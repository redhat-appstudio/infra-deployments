//go:build realrepo

package appset

import (
	"errors"
	"io"
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

// knownFinding is an allow-listed audit finding. The detail is the exact
// finding text expected for the key, so a different failure mode on an
// already-listed ApplicationSet still fails the audit instead of being
// silently accepted.
type knownFinding struct {
	detail string
	reason string
}

// knownGaps lists ApplicationSets whose component paths cannot be extracted by
// the current parser, keyed by "<overlay>/<appset>" with the expected finding
// and the reason. Every entry is a silent detection miss: changes under the
// real source path are ignored by env-detector and render-diff. Entries must be
// removed as the parser learns the corresponding generator shape — an entry
// that no longer fails is reported as stale so the list cannot rot.
var knownGaps = map[string]knownFinding{}

// knownUnresolved lists ApplicationSets whose rendered path template still
// contains placeholders after interpolation.
var knownUnresolved = map[string]knownFinding{}

type auditFinding struct {
	key    string
	detail string
}

// TestRealRepoApplicationSets renders every ArgoCD overlay and checks that each
// ApplicationSet template is turned into at least one usable component path.
// It catches the failure mode behind "environment/none" reports: a shape the
// parser does not understand, which silently contributes no paths at all.
//
// Scope: this is a completeness check, not a correctness oracle. A shape that
// yields paths but with the wrong values still passes here; that is covered by
// the unit tests in parse_test.go (for example the values.* override cases).
//
// Run with: make audit (also runs in CI, see .github/workflows/infra-tools-ci.yaml).
func TestRealRepoApplicationSets(t *testing.T) {
	overlaysDir := filepath.Join(repoRoot, "argo-cd-apps", "overlays")
	entries, err := os.ReadDir(overlaysDir)
	if err != nil {
		t.Fatalf("reading overlays dir: %v", err)
	}

	var gaps, unresolved, missing []auditFinding
	// renderedKeys records every "<overlay>/<appset>" the audit actually sees,
	// so stale allow-list entries can be distinguished from entries that now
	// pass.
	renderedKeys := make(map[string]bool)

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
			if name == "unknown" {
				continue
			}
			key := overlay + "/" + name
			renderedKeys[key] = true

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
					// A path-less entry is valid when it only consumes
					// repository files (multi-source Helm value files).
					if len(p.ExtraFiles) == 0 {
						gaps = append(gaps, auditFinding{key, "empty component path"})
					}
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

	checkFindings(t, "ApplicationSets with no extracted component path", gaps, knownGaps, renderedKeys)
	checkFindings(t, "component paths with unresolved placeholders", unresolved, knownUnresolved, renderedKeys)

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
// allow-list. Unexpected findings, including a different finding than the one
// allow-listed for a key, fail the test. Stale allow-list entries are reported
// so the list stays accurate, distinguishing entries whose ApplicationSet is
// no longer rendered from entries the parser now resolves.
func checkFindings(t *testing.T, title string, found []auditFinding, known map[string]knownFinding, rendered map[string]bool) {
	t.Helper()

	seen := make(map[string]bool)
	for _, f := range found {
		seen[f.key] = true
		allowed, ok := known[f.key]
		switch {
		case !ok:
			t.Errorf("%s: %s: %s (not in the allow-list)", title, f.key, f.detail)
		case f.detail != allowed.detail:
			t.Errorf("%s: %s: %s (allow-listed only for %q: %s)", title, f.key, f.detail, allowed.detail, allowed.reason)
		}
	}
	for key, allowed := range known {
		if seen[key] {
			continue
		}
		if !rendered[key] {
			t.Errorf("stale allow-list entry %q (%s): ApplicationSet is no longer rendered (removed or renamed), remove it from the list", key, allowed.reason)
			continue
		}
		t.Errorf("stale allow-list entry %q (%s): rendered and now resolved by the parser, remove it from the list", key, allowed.reason)
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
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// A truncated stream (io.ErrUnexpectedEOF) must not look like the
			// end of the document: the remaining ApplicationSets would be
			// skipped without being audited.
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
