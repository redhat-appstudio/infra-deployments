// Package detector — see detector.go for package docs.
package detector

import (
	"os"
	"path/filepath"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/deptree"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/kustomize"
)

// RepoQuerier is the subset of RepoRef that Detector needs.
// Accepting an interface lets tests inject fakes without touching
// the filesystem or running kustomize.
type RepoQuerier interface {
	ListSubDirs(rel string) ([]string, error)
	DirExists(rel string) bool
	FileExists(rel string) bool
	// ContainedDirExists is like DirExists, but also requires the directory
	// to stay inside the repository once symlinks (in rel itself or any of
	// its ancestors) are resolved.
	ContainedDirExists(rel string) bool
	BuildKustomization(rel string) ([]byte, error)
	ResolveDeps(rel string) (map[string]bool, error)
}

// RepoRef provides convenient, path-rooted access to a specific git ref's
// file tree.  Both Head (working copy / PR branch) and Base (target branch
// worktree) are represented as RepoRef values so every detection helper can
// work against either ref without manual filepath.Join / os.Stat boilerplate.
type RepoRef struct {
	root string
}

// NewRepoRef creates a RepoRef rooted at the given absolute path.
func NewRepoRef(root string) *RepoRef {
	return &RepoRef{root: root}
}

// Root returns the absolute root path of this ref.
func (r *RepoRef) Root() string {
	return r.root
}

// AbsPath returns the absolute path for a path relative to the repo root.
func (r *RepoRef) AbsPath(rel string) string {
	return filepath.Join(r.root, rel)
}

// DirExists reports whether rel exists and is a directory.
func (r *RepoRef) DirExists(rel string) bool {
	info, err := os.Stat(r.AbsPath(rel))
	return err == nil && info.IsDir()
}

// FileExists reports whether rel exists as a Git-tracked path. A symlink is a
// file even when it points at a directory or outside the checkout.
func (r *RepoRef) FileExists(rel string) bool {
	_, err := os.Lstat(r.AbsPath(rel))
	return err == nil
}

// ContainedDirExists reports whether rel is a directory whose real path, with
// every symlink resolved, is inside the real repository root. A lexically
// local path can still escape the checkout through a symlink (e.g.
// "components/x -> /"), and the tree is controlled by the PR author.
func (r *RepoRef) ContainedDirExists(rel string) bool {
	if !r.DirExists(rel) {
		return false
	}
	root, err := filepath.EvalSymlinks(r.root)
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(r.AbsPath(rel))
	if err != nil {
		return false
	}
	within, err := filepath.Rel(root, resolved)
	return err == nil && filepath.IsLocal(within)
}

// ReadDir returns the directory entries at rel.
func (r *RepoRef) ReadDir(rel string) ([]os.DirEntry, error) {
	return os.ReadDir(r.AbsPath(rel))
}

// ListSubDirs returns the names of immediate subdirectories under rel.
func (r *RepoRef) ListSubDirs(rel string) ([]string, error) {
	entries, err := os.ReadDir(r.AbsPath(rel))
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}

// BuildKustomization runs kustomize build on the directory at rel and returns
// the rendered YAML.
func (r *RepoRef) BuildKustomization(rel string) ([]byte, error) {
	return kustomize.Build(r.AbsPath(rel))
}

// ResolveDeps walks the kustomization dependency tree starting at rel and
// returns the set of all files (repo-root-relative) it depends on.
func (r *RepoRef) ResolveDeps(rel string) (map[string]bool, error) {
	return deptree.Resolve(r.root, rel)
}
