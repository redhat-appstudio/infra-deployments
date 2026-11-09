package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
)

// initRepo creates a temporary git repository with one commit containing the
// given files and returns its path.
func initRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "test")
	writeFiles(t, dir, files)
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "base")
	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()

	// Test-only helper: every command comes from the test itself.
	cmd := exec.CommandContext(t.Context(), args[0], args[1:]...) //nolint:gosec // test helper
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// A renamed file must be reported under both paths: the old path belongs to the
// component the file left, and dropping it hides that component from the diff.
func TestChangedFiles_RenameKeepsOldPath(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	dir := initRepo(t, map[string]string{"components/a/deploy.yaml": "a: 1\n"})
	run(t, dir, "mkdir", "-p", "components/b")
	run(t, dir, "git", "mv", "components/a/deploy.yaml", "components/b/deploy.yaml")
	run(t, dir, "git", "commit", "-m", "move")

	files, err := git.ChangedFiles(ctx, dir, "main~1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(files).To(ConsistOf("components/a/deploy.yaml", "components/b/deploy.yaml"))
}

// Paths with spaces and quotes must survive the split: newline-separated output
// would mangle them.
func TestChangedFiles_UnusualPaths(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	dir := initRepo(t, map[string]string{"components/a/base.yaml": "a: 1\n"})
	const weird = "components/a/with space and \"quotes\".yaml"
	writeFiles(t, dir, map[string]string{weird: "b: 2\n"})
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "add weird path")

	files, err := git.ChangedFiles(ctx, dir, "main~1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(files).To(ConsistOf(weird))
}

// Local runs build from the working tree, so uncommitted edits and untracked
// files must be part of the change set.
func TestChangedFilesWorkingTree_IncludesUncommittedWork(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	dir := initRepo(t, map[string]string{"components/a/deploy.yaml": "a: 1\n"})
	writeFiles(t, dir, map[string]string{
		"components/a/deploy.yaml": "a: 2\n",
		"components/a/new.yaml":    "new: true\n",
	})

	files, err := git.ChangedFilesWorkingTree(ctx, dir, "HEAD")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(files).To(ConsistOf("components/a/deploy.yaml", "components/a/new.yaml"))
}

// A path can be both part of the diff against the base ref (still tracked
// there) and untracked in the working tree after `git rm --cached`. It must be
// reported once.
func TestChangedFilesWorkingTree_NoDuplicates(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	dir := initRepo(t, map[string]string{"components/a/deploy.yaml": "a: 1\n"})
	run(t, dir, "git", "rm", "--cached", "components/a/deploy.yaml")
	run(t, dir, "git", "commit", "-m", "stop tracking")

	// The file now exists only as an untracked working-tree file, while the
	// base ref still tracks it, so both commands report it.
	files, err := git.ChangedFilesWorkingTree(ctx, dir, "HEAD~1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(files).To(ConsistOf("components/a/deploy.yaml"))
}

// A clean working tree yields no changes.
func TestChangedFilesWorkingTree_CleanTree(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	dir := initRepo(t, map[string]string{"components/a/deploy.yaml": "a: 1\n"})
	files, err := git.ChangedFilesWorkingTree(ctx, dir, "HEAD")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(files).To(BeEmpty())
}

// An unknown base ref must surface the git error instead of an empty list.
func TestChangedFilesWorkingTree_UnknownRef(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	dir := initRepo(t, map[string]string{"components/a/deploy.yaml": "a: 1\n"})
	_, err := git.ChangedFilesWorkingTree(ctx, dir, "does-not-exist")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("does-not-exist"))
}
