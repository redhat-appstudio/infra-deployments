package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

const (
	oldRef = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newRef = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// fakeCommenter records the most recent UpsertCommentByMarker call.
type fakeCommenter struct {
	prNumber int
	body     string
	marker   string
}

func (f *fakeCommenter) UpsertCommentByMarker(_ context.Context, prNumber int, body, marker string) error {
	f.prNumber = prNumber
	f.body = body
	f.marker = marker
	return nil
}

func writeTempKustomization(t *testing.T, ref string) string {
	t.Helper()
	content := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n" +
		"  - https://github.com/konflux-ci/konflux-ci/operator/config/default?ref=" + ref + "\n"
	f, err := os.CreateTemp(t.TempDir(), "kustomization-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

// TestPost_PassesCorrectMarkerAndBody verifies that post wires the permanent
// commentMarker through to UpsertCommentByMarker, not some other value.
func TestPost_PassesCorrectMarkerAndBody(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeCommenter{}
	err := post(context.Background(), fake, 42, "hello")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.prNumber).To(Equal(42))
	g.Expect(fake.body).To(Equal("hello"))
	g.Expect(fake.marker).To(Equal(commentMarker))
}

// TestFormatCompare_ShortRef verifies refs under 12 chars (e.g. branch names)
// are not truncated — the short() helper takes a different branch.
func TestFormatCompare_ShortRef(t *testing.T) {
	g := NewWithT(t)
	body := formatCompare("main", "feature-x")
	g.Expect(body).To(ContainSubstring("main"))
	g.Expect(body).To(ContainSubstring("feature-x"))
}

// TestComputeBody_Unchanged verifies that identical kustomization files produce
// the no-change comment body with the required marker and heading.
func TestComputeBody_Unchanged(t *testing.T) {
	g := NewWithT(t)
	path := writeTempKustomization(t, oldRef)
	body, err := computeBody(path, path)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring(commentMarker))
	g.Expect(body).To(ContainSubstring("### Operator Changelog"))
	g.Expect(body).To(ContainSubstring("No operator ref change"))
}

// TestComputeBody_Changed verifies that different kustomization files produce
// a compare comment containing both refs, short display refs, and a compare URL.
func TestComputeBody_Changed(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(basePath, headPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring(commentMarker))
	g.Expect(body).To(ContainSubstring(oldRef))
	g.Expect(body).To(ContainSubstring(newRef))
	g.Expect(body).To(ContainSubstring(oldRef[:12]))
	g.Expect(body).To(ContainSubstring("konflux-ci/konflux-ci/compare/" + oldRef + "..." + newRef))
	g.Expect(body).NotTo(ContainSubstring("No operator ref change"))
}

// TestComputeBody_Error verifies that an unreadable file returns an error.
func TestComputeBody_Error(t *testing.T) {
	g := NewWithT(t)
	_, err := computeBody("/nonexistent/kustomization.yaml", "/nonexistent/kustomization.yaml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("extracting operator refs"))
}

// TestBuildBody_Integration creates a real git repo with two commits that have
// different operator refs, then calls buildBody to validate the full git path:
// worktree creation, kustomization parsing, and comment body selection.
func TestBuildBody_Integration(t *testing.T) {
	g := NewWithT(t)

	// Set up a bare git repo with minimal config.
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test")

	// Commit 1: kustomization with oldRef (this becomes the base).
	kustPath := filepath.Join(dir, kustomizationPath)
	g.Expect(os.MkdirAll(filepath.Dir(kustPath), 0755)).To(Succeed())
	writeKustomizationFile(t, kustPath, oldRef)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")
	baseCommit := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))

	// Commit 2: kustomization with newRef (this becomes HEAD).
	writeKustomizationFile(t, kustPath, newRef)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "bump operator ref")

	// buildBody should detect the change and return a compare body.
	body, err := buildBody(context.Background(), dir, baseCommit)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring(oldRef))
	g.Expect(body).To(ContainSubstring(newRef))
	g.Expect(body).To(ContainSubstring("compare"))
}

// gitRun runs a git command in dir and fails the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// gitOutput runs a git command and returns its stdout.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// writeKustomizationFile writes a minimal kustomization.yaml with the given ref.
func writeKustomizationFile(t *testing.T, path, ref string) {
	t.Helper()
	content := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n" +
		"  - https://github.com/konflux-ci/konflux-ci/operator/config/default?ref=" + ref + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing kustomization: %v", err)
	}
}
