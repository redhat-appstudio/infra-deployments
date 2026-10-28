package detector //nolint:testpackage // tests unexported prefixDeps and Detector internals

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// A directory reached through a symlink that leaves the checkout is not
// contained in the repository, even though its relative path is local.
func TestRepoRef_ContainedDirExists(t *testing.T) {
	g := NewWithT(t)

	root := t.TempDir()
	outside := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(root, "components", "real", "sub"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(outside, "secret"), 0o755)).To(Succeed())
	g.Expect(os.Symlink(outside, filepath.Join(root, "components", "escape"))).To(Succeed())
	g.Expect(os.Symlink(filepath.Join(root, "components", "real"), filepath.Join(root, "components", "inside"))).To(Succeed())

	repo := NewRepoRef(root)
	g.Expect(repo.ContainedDirExists("components/real")).To(BeTrue())
	g.Expect(repo.ContainedDirExists("components/inside/sub")).To(BeTrue())
	g.Expect(repo.ContainedDirExists("components/missing")).To(BeFalse())

	// The symlink itself, and a directory below it (symlinked ancestor).
	g.Expect(repo.DirExists("components/escape/secret")).To(BeTrue())
	g.Expect(repo.ContainedDirExists("components/escape")).To(BeFalse())
	g.Expect(repo.ContainedDirExists("components/escape/secret")).To(BeFalse())
}

// prefixDeps never walks a prefix that escapes the checkout through a
// symlink.
func TestPrefixDeps_RefusesSymlinkEscape(t *testing.T) {
	g := NewWithT(t)

	root := t.TempDir()
	outside := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(root, "components"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(outside, "kustomization.yaml"),
		[]byte("resources:\n- cm.yaml\n"), 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(outside, "cm.yaml"),
		[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n"), 0o600)).To(Succeed())
	g.Expect(os.Symlink(outside, filepath.Join(root, "components", "x"))).To(Succeed())

	d := &Detector{head: NewRepoRef(root)}
	g.Expect(d.prefixDeps("components/x")).To(BeNil())
}
