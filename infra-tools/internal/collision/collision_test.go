package collision_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/collision"
)

// -------------------- extractAppSets Tests --------------------
func TestExtractAppSets_ExtractsTemplateAndName(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: my-app
spec:
  template:
    metadata:
      name: my-app-{{nameNormalized}}
    spec:
      source:
        path: components/my-app-rd
`
	appSets, err := collision.ExtractAppSets([]byte(yaml), "argo-cd-apps/overlays/rd-dev")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(appSets).To(HaveLen(1))
	g.Expect(appSets[0].AppNameTemplate).To(Equal("my-app-{{nameNormalized}}"))
	g.Expect(appSets[0].Name).To(Equal("my-app"))
	g.Expect(appSets[0].OverlayPath).To(Equal("argo-cd-apps/overlays/rd-dev"))
}

func TestExtractAppSets_EmptyYAML(t *testing.T) {
	g := NewWithT(t)

	yaml := ``
	appSets, err := collision.ExtractAppSets([]byte(yaml), "overlay")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(appSets).To(BeEmpty())
}

func TestExtractAppSets_NonAppSetResourcesIgnored(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: not-an-appset
data:
  key: value
---
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: my-app
spec:
  template:
    metadata:
      name: my-app-{{nameNormalized}}
`
	appSets, err := collision.ExtractAppSets([]byte(yaml), "overlay")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(appSets).To(HaveLen(1))
}

func TestExtractAppSets_MissingTemplateNameSkipped(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: no-template-name
spec:
  template:
    spec:
      source:
        path: components/foo
`
	appSets, err := collision.ExtractAppSets([]byte(yaml), "overlay")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(appSets).To(BeEmpty())
}

func TestExtractAppSets_DecoderErrorReturned(t *testing.T) {
	g := NewWithT(t)

	yaml := `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
	`
	appSets, err := collision.ExtractAppSets([]byte(yaml), "overlay")
	g.Expect(err).To(HaveOccurred())
	g.Expect(appSets).To(BeEmpty())
	g.Expect(err).To(MatchError(ContainSubstring("decoding YAML document")))
}

// -------------------- findCollisions Tests --------------------
func TestFindCollisions_DetectsDuplicateTemplates(t *testing.T) {
	g := NewWithT(t)

	entries := []collision.AppSet{
		{AppNameTemplate: "my-app-{{nameNormalized}}", Name: "my-app", OverlayPath: "argo-cd-apps/overlays/development"},
		{AppNameTemplate: "my-app-{{nameNormalized}}", Name: "my-app", OverlayPath: "argo-cd-apps/overlays/rd-dev"},
		{AppNameTemplate: "ur-app-{{nameNormalized}}", Name: "ur-app", OverlayPath: "argo-cd-apps/overlays/rd-dev"},
	}

	collisions := collision.FindCollisions(entries)
	g.Expect(collisions).To(HaveLen(1))
	g.Expect(collisions[0].AppNameTemplate).To(Equal("my-app-{{nameNormalized}}"))
	g.Expect(collisions[0].AppSets).To(HaveLen(2))
}

func TestFindCollisions_NoDuplicatesReturnsEmpty(t *testing.T) {
	g := NewWithT(t)

	entries := []collision.AppSet{
		{AppNameTemplate: "my-app-{{nameNormalized}}", Name: "my-app", OverlayPath: "argo-cd-apps/overlays/development"},
		{AppNameTemplate: "ur-app-{{nameNormalized}}", Name: "ur-app", OverlayPath: "argo-cd-apps/overlays/rd-dev"},
	}

	g.Expect(collision.FindCollisions(entries)).To(BeEmpty())
}

// -------------------- stringField Tests --------------------
func TestStringField_ValidNestedPath(t *testing.T) {
	g := NewWithT(t)

	doc := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "my-app-{{nameNormalized}}",
				},
			},
		},
	}

	g.Expect(collision.StringField(doc, "spec", "template", "metadata", "name")).To(Equal("my-app-{{nameNormalized}}"))
	g.Expect(collision.StringField(doc, "spec", "missing", "name")).To(Equal(""))
	g.Expect(collision.StringField(doc, "spec", "template", "metadata", "missing")).To(Equal(""))
}

func TestStringField_InvalidNestedPath(t *testing.T) {
	g := NewWithT(t)

	doc := map[string]interface{}{
		"spec": "string",
	}

	g.Expect(collision.StringField(doc, "spec", "template", "metadata", "name")).To(Equal(""))
}
