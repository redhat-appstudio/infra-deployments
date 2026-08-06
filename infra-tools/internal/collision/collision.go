// Package collision detects ArgoCD ApplicationSets that would template
// colliding generated Application names when their overlays are deployed
// together to the same cluster / ArgoCD control-plane namespace.
//
// Background: argo-cd-apps overlays generate child Applications from an
// ApplicationSet's spec.template.metadata.name, e.g. "my-app-{{nameNormalized}}".
// Kustomize's nameSuffix only renames the ApplicationSet object itself
// (e.g. "my-app" -> "my-app-ring-0"); it does NOT rewrite the templated string
// inside spec.template.metadata.name, since that's opaque spec data to
// Kustomize, not a tracked object reference. So if two *different*
// ApplicationSets are deployed to the same cluster and both template the
// same generated Application name (e.g. both produce
// "my-app-{{nameNormalized}}"), ArgoCD's ApplicationSet controller will fight
// over ownership of the resulting Application object -- one of them loses,
// and that Application gets stuck/flapping.
//
// This matters during the argo-cd-apps/overlays/development ->
// argo-cd-apps/overlays/rd-dev ring-based migration: a component can
// temporarily exist in both trees while being migrated, and both overlays
// are deployed to the same cluster during e2e (see the dual
// apply_and_wait_for_root_application calls in hack/preview.sh).
package collision

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/kustomize"
)

// -------------------- Types --------------------

// OverlayGroup is a set of overlays that are deployed together to the same cluster
// / ArgoCD control-plane namespace (e.g. during e2e bootstrap, or one
// overlay importing another's ApplicationSets directly), and therefore must
// not template colliding generated Application names.
type OverlayGroup struct {
	// OverlayPaths are paths (relative to the repo root) to the overlays that
	// are co-deployed.
	OverlayPaths []string
}

// DefaultGroups lists the known sets of overlays that are deployed together.
// Add new pairings here if new overlays are wired into shared bootstrap
// targets.
var DefaultGroups = []OverlayGroup{
	{OverlayPaths: []string{"argo-cd-apps/overlays/development", "argo-cd-apps/overlays/rd-dev"}},
	{OverlayPaths: []string{"argo-cd-apps/overlays/development-operator"}},
}

// AppSet represents one ApplicationSet found while building an overlay: the
// generated Application name template it produces
// (spec.template.metadata.name), its own object name, and the overlay it
// was found in.
type AppSet struct {
	AppNameTemplate string // e.g. "my-app-{{nameNormalized}}"
	Name            string // e.g. "my-app"
	OverlayPath     string // relative to the repo root; e.g. "argo-cd-apps/overlays/my-overlay"
}

// Collision groups together the AppSets that all produce the same generated
// Application name template.
type Collision struct {
	AppNameTemplate string // e.g. "my-app-{{nameNormalized}}"
	AppSets         []AppSet
}

// -------------------- Functions --------------------

// CheckOverlayGroup builds every overlay in the group to verify that no
// generated Application name templates collide.
func CheckOverlayGroup(repoRoot string, group OverlayGroup) ([]Collision, error) {
	var appSets []AppSet
	for _, overlayPath := range group.OverlayPaths {
		dir := filepath.Join(repoRoot, overlayPath)
		rendered, err := kustomize.Build(dir)
		if err != nil {
			return nil, fmt.Errorf("building overlay %s: %w", overlayPath, err)
		}

		overlayAppSets, err := extractAppSets(rendered, overlayPath)
		if err != nil {
			return nil, fmt.Errorf("parsing ApplicationSets in %s: %w", overlayPath, err)
		}
		appSets = append(appSets, overlayAppSets...)
	}

	return findCollisions(appSets), nil
}

// ExtractAppSets decodes rendered multi-document YAML and creates an
// AppSet for every ApplicationSet resource found. ApplicationSets with no
// resolvable spec.template.metadata.name are skipped (nothing to compare).
func extractAppSets(rendered []byte, overlayPath string) ([]AppSet, error) {
	var appSets []AppSet

	decoder := yaml.NewDecoder(bytes.NewReader(rendered))
	for {
		// decode the next YAML document
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

		// skip non-ApplicationSet resources
		kind, _ := doc["kind"].(string)
		if kind != "ApplicationSet" {
			continue
		}

		template := stringField(doc, "spec", "template", "metadata", "name")
		if template == "" {
			continue
		}

		appSets = append(appSets, AppSet{
			AppNameTemplate: template,
			Name:            stringField(doc, "metadata", "name"),
			OverlayPath:     overlayPath,
		})
	}

	return appSets, nil
}

// stringField walks a chain of nested map[string]interface{} keys and
// returns the string value at the end, or "" if any key along the path is
// missing or not a map/string.
func stringField(doc map[string]interface{}, keys ...string) string {
	var current interface{} = doc
	for _, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = m[key]
	}
	s, _ := current.(string)
	return s
}

// FindCollisions groups appSets by AppNameTemplate, preserving first-seen order,
// and returns any groups with more than one appSet.
func findCollisions(appSets []AppSet) []Collision {
	byTemplate := make(map[string][]AppSet)
	var orderedSet []string

	// Only add app name templates to the ordered set if they haven't been seen yet
	for _, appSet := range appSets {
		if _, seen := byTemplate[appSet.AppNameTemplate]; !seen {
			orderedSet = append(orderedSet, appSet.AppNameTemplate)
		}
		byTemplate[appSet.AppNameTemplate] = append(byTemplate[appSet.AppNameTemplate], appSet)
	}

	// If an app name template has more than one associated app set, there's a collision
	var collisions []Collision
	for _, tmpl := range orderedSet {
		if appSets := byTemplate[tmpl]; len(appSets) > 1 {
			collisions = append(collisions, Collision{AppNameTemplate: tmpl, AppSets: appSets})
		}
	}
	return collisions
}
