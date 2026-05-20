package changelog

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

// buildServicePatch is a representative patch from the real compare data.
const buildServicePatch = `@@ -1,7 +1,7 @@
 apiVersion: kustomize.config.k8s.io/v1beta1
 kind: Kustomization
 resources:
- - https://github.com/konflux-ci/build-service/config/default?ref=211140a26c96e8f028a0595fb779ea13210ed5c8
+ - https://github.com/konflux-ci/build-service/config/default?ref=04a4744321a7fb747f796da783d51fc322aef598
 - build-pipeline-config.yaml
 namespace: build-service
 images:
 - name: quay.io/konflux-ci/build-service
   newName: quay.io/konflux-ci/build-service
- newTag: 211140a26c96e8f028a0595fb779ea13210ed5c8
+ newTag: 04a4744321a7fb747f796da783d51fc322aef598`

const integrationPatch = `@@ -1,8 +1,8 @@
 apiVersion: kustomize.config.k8s.io/v1beta1
 kind: Kustomization
 resources:
-- https://github.com/konflux-ci/integration-service/config/default?ref=ec001afe3986d224f5247fd6e9ad3162e8a52cde
-- https://github.com/konflux-ci/integration-service/config/snapshotgc?ref=ec001afe3986d224f5247fd6e9ad3162e8a52cde
+- https://github.com/konflux-ci/integration-service/config/default?ref=ef2610ecd344292fb85e01321f7b613c7e621ec5
+- https://github.com/konflux-ci/integration-service/config/snapshotgc?ref=ef2610ecd344292fb85e01321f7b613c7e621ec5`

func TestExtractServiceBumps_SingleService(t *testing.T) {
	g := NewWithT(t)

	files := []FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/core/kustomization.yaml",
			Patch:    buildServicePatch,
			Status:   "modified",
		},
	}

	bumps := ExtractServiceBumps(files)
	g.Expect(bumps).To(HaveLen(1))
	g.Expect(bumps[0].Owner).To(Equal("konflux-ci"))
	g.Expect(bumps[0].Repo).To(Equal("build-service"))
	g.Expect(bumps[0].OldSHA).To(Equal("211140a26c96e8f028a0595fb779ea13210ed5c8"))
	g.Expect(bumps[0].NewSHA).To(Equal("04a4744321a7fb747f796da783d51fc322aef598"))
}

func TestExtractServiceBumps_DeduplicatesSameRepo(t *testing.T) {
	g := NewWithT(t)

	// Same repo referenced from two kustomization files (e.g. core/ and another subdir)
	files := []FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/core/kustomization.yaml",
			Patch:    buildServicePatch,
			Status:   "modified",
		},
		{
			Filename: "operator/upstream-kustomizations/build-service/extra/kustomization.yaml",
			Patch:    buildServicePatch,
			Status:   "modified",
		},
	}

	bumps := ExtractServiceBumps(files)
	g.Expect(bumps).To(HaveLen(1))
}

func TestExtractServiceBumps_MultipleServices(t *testing.T) {
	g := NewWithT(t)

	files := []FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/core/kustomization.yaml",
			Patch:    buildServicePatch,
			Status:   "modified",
		},
		{
			Filename: "operator/upstream-kustomizations/integration/core/kustomization.yaml",
			Patch:    integrationPatch,
			Status:   "modified",
		},
	}

	bumps := ExtractServiceBumps(files)
	g.Expect(bumps).To(HaveLen(2))

	repos := []string{bumps[0].Repo, bumps[1].Repo}
	g.Expect(repos).To(ConsistOf("build-service", "integration-service"))
}

func TestExtractServiceBumps_IgnoresNonKustomizationFiles(t *testing.T) {
	g := NewWithT(t)

	files := []FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/core/some-patch.yaml",
			Patch:    buildServicePatch,
			Status:   "modified",
		},
		{
			Filename: ".github/workflows/foo.yaml",
			Patch:    buildServicePatch,
			Status:   "modified",
		},
	}

	bumps := ExtractServiceBumps(files)
	g.Expect(bumps).To(BeEmpty())
}

func TestExtractServiceBumps_IgnoresUnchangedRef(t *testing.T) {
	g := NewWithT(t)

	// Patch where the SHA didn't change (only other fields changed)
	patch := `@@ -5,6 +5,6 @@
 namespace: build-service
-somefield: old
+somefield: new`

	files := []FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/core/kustomization.yaml",
			Patch:    patch,
			Status:   "modified",
		},
	}

	bumps := ExtractServiceBumps(files)
	g.Expect(bumps).To(BeEmpty())
}

func TestExtractServiceBumps_IgnoresEmptyPatch(t *testing.T) {
	g := NewWithT(t)

	files := []FileChange{
		{
			Filename: "operator/upstream-kustomizations/build-service/core/kustomization.yaml",
			Patch:    "",
			Status:   "modified",
		},
	}

	bumps := ExtractServiceBumps(files)
	g.Expect(bumps).To(BeEmpty())
}

// TestExtractServiceBumps_RealData verifies against the actual saved API
// response from our investigation. Expected: 6 service bumps.
func TestExtractServiceBumps_RealData(t *testing.T) {
	fixtureFile := "/home/kellychen/.cursor/projects/home-kellychen-Desktop-infra-deployments/agent-tools/e73643f3-22f0-4c14-bd84-99c3a8e44efb.txt"
	if _, err := os.Stat(fixtureFile); os.IsNotExist(err) {
		t.Skip("fixture file not available")
	}

	g := NewWithT(t)

	data, err := os.ReadFile(fixtureFile)
	g.Expect(err).NotTo(HaveOccurred())

	var response struct {
		Files []struct {
			Filename string `json:"filename"`
			Patch    string `json:"patch"`
			Status   string `json:"status"`
		} `json:"files"`
	}
	g.Expect(json.Unmarshal(data, &response)).To(Succeed())

	files := make([]FileChange, len(response.Files))
	for i, f := range response.Files {
		files[i] = FileChange{Filename: f.Filename, Patch: f.Patch, Status: f.Status}
	}

	bumps := ExtractServiceBumps(files)

	// From our manual investigation we know exactly 6 services bumped
	g.Expect(bumps).To(HaveLen(6))

	// Verify the services we expect are present
	repos := make(map[string]bool)
	for _, b := range bumps {
		repos[b.Owner+"/"+b.Repo] = true
		g.Expect(b.OldSHA).To(MatchRegexp(`^[0-9a-f]{40}$`))
		g.Expect(b.NewSHA).To(MatchRegexp(`^[0-9a-f]{40}$`))
		g.Expect(b.OldSHA).NotTo(Equal(b.NewSHA))
	}

	g.Expect(repos).To(HaveKey("konflux-ci/build-service"))
	g.Expect(repos).To(HaveKey("konflux-ci/image-controller"))
	g.Expect(repos).To(HaveKey("konflux-ci/integration-service"))
	g.Expect(repos).To(HaveKey("konflux-ci/release-service"))
}
