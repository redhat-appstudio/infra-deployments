package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// fakeBuilder serves fixed YAML per path so runLocal can render one component.
type fakeBuilder struct {
	dirs  map[string]bool
	yamls map[string][]byte
}

func (f fakeBuilder) DirExists(rel string) bool { return f.dirs[rel] }

func (f fakeBuilder) BuildKustomization(rel string) ([]byte, error) {
	yaml, ok := f.yamls[rel]
	if !ok {
		return nil, fmt.Errorf("no kustomization at %s", rel)
	}
	return yaml, nil
}

// runLocal must handle an empty affected set in every output mode without
// touching the (nil) repo builders.
func TestRunLocal_EmptyEngine(t *testing.T) {
	g := NewWithT(t)

	engine := renderdiff.NewEngine(nil, nil, 0)
	affected := map[detector.Environment][]appset.ComponentPath{}

	g.Expect(runLocal(t.Context(), engine, affected, "never", false, t.TempDir())).To(Succeed())
	g.Expect(runLocal(t.Context(), engine, affected, "never", true, "")).To(Succeed())
	g.Expect(runLocal(t.Context(), engine, affected, "never", false, "")).To(Succeed())
}

// The summary must be printed after the progressive printer has drained the
// channel, so the component diff always appears first.
func TestRunLocal_PrintsDiffBeforeSummary(t *testing.T) {
	g := NewWithT(t)

	const path = "components/foo"
	head := fakeBuilder{
		dirs: map[string]bool{path: true},
		yamls: map[string][]byte{path: []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: foo
data:
  key: head
`)},
	}
	base := fakeBuilder{
		dirs: map[string]bool{path: true},
		yamls: map[string][]byte{path: []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: foo
data:
  key: base
`)},
	}
	engine := renderdiff.NewEngine(head, base, 1)
	affected := map[detector.Environment][]appset.ComponentPath{
		detector.Development: {{Path: path}},
	}

	original := os.Stdout
	reader, writer, err := os.Pipe()
	g.Expect(err).NotTo(HaveOccurred())
	os.Stdout = writer

	runErr := runLocal(t.Context(), engine, affected, "never", false, "")

	g.Expect(writer.Close()).To(Succeed())
	os.Stdout = original
	out, readErr := io.ReadAll(reader)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(runErr).NotTo(HaveOccurred())

	text := string(out)
	g.Expect(text).To(ContainSubstring("=== components/foo (development)"))
	g.Expect(text).To(ContainSubstring("--- Summary ---"))
	g.Expect(strings.Index(text, "=== components/foo")).
		To(BeNumerically("<", strings.Index(text, "--- Summary ---")))
}
