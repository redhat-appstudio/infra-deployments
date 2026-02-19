package deptree

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestResolve_SimpleKustomization(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	// Create component/base/kustomization.yaml
	baseDir := filepath.Join(tmpDir, "component", "base")
	g.Expect(os.MkdirAll(baseDir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(baseDir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml
  - service.yaml
`)
	writeFile(t, filepath.Join(baseDir, "deployment.yaml"), "kind: Deployment")
	writeFile(t, filepath.Join(baseDir, "service.yaml"), "kind: Service")

	// Create component/production/kustomization.yaml
	prodDir := filepath.Join(tmpDir, "component", "production")
	g.Expect(os.MkdirAll(prodDir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(prodDir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../base
patches:
  - path: prod-patch.yaml
    target:
      kind: Deployment
`)
	writeFile(t, filepath.Join(prodDir, "prod-patch.yaml"), "kind: Deployment")

	deps, err := Resolve(tmpDir, "component/production")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(deps).To(HaveKey("component/production/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("component/production/prod-patch.yaml"))
	g.Expect(deps).To(HaveKey("component/base/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("component/base/deployment.yaml"))
	g.Expect(deps).To(HaveKey("component/base/service.yaml"))
}

func TestResolve_SkipsRemoteURLs(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "component")
	g.Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(dir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - https://github.com/example/repo/config/default?ref=abc123
  - local-file.yaml
`)
	writeFile(t, filepath.Join(dir, "local-file.yaml"), "kind: ConfigMap")

	deps, err := Resolve(tmpDir, "component")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(deps).To(HaveKey("component/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("component/local-file.yaml"))

	// Remote URLs should not appear
	for dep := range deps {
		g.Expect(dep).NotTo(Equal("https://github.com/example/repo/config/default?ref=abc123"))
	}
}

func TestResolve_Components(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	// Create a kustomize component
	compDir := filepath.Join(tmpDir, "my-component", "k-components", "extra")
	g.Expect(os.MkdirAll(compDir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(compDir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
resources:
  - extra-resource.yaml
`)
	writeFile(t, filepath.Join(compDir, "extra-resource.yaml"), "kind: ConfigMap")

	// Create main kustomization that uses the component
	mainDir := filepath.Join(tmpDir, "my-component", "production")
	g.Expect(os.MkdirAll(mainDir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(mainDir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
components:
  - ../k-components/extra
`)

	deps, err := Resolve(tmpDir, "my-component/production")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(deps).To(HaveKey("my-component/production/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("my-component/k-components/extra/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("my-component/k-components/extra/extra-resource.yaml"))
}

func TestResolve_PatchesStrategicMerge(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "component")
	g.Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(dir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patchesStrategicMerge:
  - delete-stuff.yaml
`)
	writeFile(t, filepath.Join(dir, "delete-stuff.yaml"), "kind: Deployment")

	deps, err := Resolve(tmpDir, "component")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(deps).To(HaveKey("component/delete-stuff.yaml"))
}

func TestResolve_ConfigMapGenerator(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, "component")
	g.Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
	writeFile(t, filepath.Join(dir, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
  - name: my-config
    files:
      - config.json
      - key=data.txt
    envs:
      - env.properties
`)
	writeFile(t, filepath.Join(dir, "config.json"), "{}")
	writeFile(t, filepath.Join(dir, "data.txt"), "data")
	writeFile(t, filepath.Join(dir, "env.properties"), "KEY=val")

	deps, err := Resolve(tmpDir, "component")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(deps).To(HaveKey("component/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("component/config.json"))
	g.Expect(deps).To(HaveKey("component/data.txt"))
	g.Expect(deps).To(HaveKey("component/env.properties"))
}

func TestResolve_CircularReference(t *testing.T) {
	g := NewWithT(t)
	tmpDir := t.TempDir()

	// Create two directories that reference each other
	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")
	g.Expect(os.MkdirAll(dirA, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(dirB, 0o755)).To(Succeed())

	writeFile(t, filepath.Join(dirA, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../b
`)
	writeFile(t, filepath.Join(dirB, "kustomization.yaml"), `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../a
`)

	// Should not infinite loop
	deps, err := Resolve(tmpDir, "a")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(deps).To(HaveKey("a/kustomization.yaml"))
	g.Expect(deps).To(HaveKey("b/kustomization.yaml"))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	NewWithT(t).Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
}
