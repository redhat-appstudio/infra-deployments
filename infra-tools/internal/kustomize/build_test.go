package kustomize_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/kustomize"
)

// writeOverlay creates a kustomization that uses a deprecated field
// (patchesStrategicMerge) so kustomize writes a warning to stderr.
func writeOverlay(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - configmap.yaml
patchesStrategicMerge:
  - patch.yaml
`,
		"configmap.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`,
		"patch.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: patched
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Builds run concurrently in both tools. Redirecting os.Stderr must be
// reference-counted, otherwise one build can restore stderr while another still
// relies on it being muted, or leave stderr pointing at a closed file.
func TestBuild_Concurrent(t *testing.T) {
	g := NewWithT(t)

	dir := writeOverlay(t)

	const workers = 8
	var wg sync.WaitGroup
	results := make([][]byte, workers)
	errs := make([]error, workers)

	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = kustomize.Build(dir)
		}()
	}

	wg.Wait()

	original := os.Stderr
	for i := range workers {
		g.Expect(errs[i]).NotTo(HaveOccurred())
		g.Expect(string(results[i])).To(ContainSubstring("key: patched"))
	}
	g.Expect(os.Stderr).To(Equal(original), "stderr must be restored after concurrent builds")

	// A build after the concurrent burst must still work with stderr restored.
	out, err := kustomize.Build(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.Count(string(out), "kind: ConfigMap")).To(Equal(1))
}

// A missing kustomization must return an error mentioning the directory, and
// must not leave stderr muted.
func TestBuild_MissingKustomization(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	original := os.Stderr

	_, err := kustomize.Build(dir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("kustomize build"))
	g.Expect(os.Stderr).To(Equal(original))
}
