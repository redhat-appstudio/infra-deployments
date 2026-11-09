// Package git provides helpers for interacting with a local git repository.
package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// TopLevel returns the root directory of the git repository that contains
// the current working directory.
func TopLevel(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ResolveRef resolves a git ref (branch, tag, symbolic name, etc.) to its
// short commit SHA.
func ResolveRef(ctx context.Context, repoRoot, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", ref)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --short %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedFiles returns the list of files changed between baseRef and HEAD.
// It uses a two-point diff (baseRef HEAD) which compares the full trees.
// Callers should pass the merge-base commit (see MergeBase) rather than a
// branch name so the diff only contains the current branch's own changes,
// regardless of whether the branch has been rebased.
//
// Rename detection is disabled so that a moved file appears under both its old
// and new path: the old path still belongs to the component it was removed
// from, and dropping it would hide that environment.
func ChangedFiles(ctx context.Context, repoRoot, baseRef string) ([]string, error) {
	return gitDiffNames(ctx, repoRoot, baseRef, "HEAD")
}

// ChangedFilesWorkingTree returns the files changed between baseRef and the
// working tree, including untracked files. Local runs need this: HEAD alone
// ignores edits that have not been committed yet, while the render itself is
// built from the working tree.
func ChangedFilesWorkingTree(ctx context.Context, repoRoot, baseRef string) ([]string, error) {
	tracked, err := gitDiffNames(ctx, repoRoot, baseRef)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard", "-z")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files --others: %w", err)
	}

	seen := make(map[string]bool, len(tracked))
	files := make([]string, 0, len(tracked))
	for _, f := range append(tracked, splitNul(out)...) {
		if !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	sort.Strings(files)
	return files, nil
}

// gitDiffNames runs "git diff --name-only --no-renames -z" between the given
// refs (or between ref and the working tree when head is empty).
func gitDiffNames(ctx context.Context, repoRoot, baseRef string, head ...string) ([]string, error) {
	args := make([]string, 0, 5+len(head))
	args = append(args, "diff", "--name-only", "--no-renames", "-z", baseRef)
	args = append(args, head...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return splitNul(out), nil
}

// splitNul splits NUL-separated git output, dropping empty entries. NUL
// separation is required for paths containing spaces or quotes, which
// newline-separated output would mangle.
func splitNul(out []byte) []string {
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}

// MergeBase returns the merge-base commit between HEAD and the given ref.
func MergeBase(ctx context.Context, repoRoot, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "HEAD", ref)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git merge-base HEAD %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CreateWorktree creates a temporary git worktree checked out at the given ref.
// It returns the worktree path and a cleanup function that removes the worktree.
// The cleanup function uses a background context so it always runs even if the
// parent context has been cancelled.
func CreateWorktree(ctx context.Context, repoRoot, ref string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "env-detector-worktree-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}

	cleanup := func() {
		// Cleanup should not be cancelled — use a fresh background context.
		cmd := exec.Command("git", "worktree", "remove", "--force", tmpDir)
		cmd.Dir = repoRoot
		if err := cmd.Run(); err != nil {
			slog.Warn("failed to remove worktree", "path", tmpDir, "err", err)
		}
		_ = os.RemoveAll(tmpDir)
	}

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", tmpDir, ref)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("creating worktree at %s: %w", ref, err)
	}

	return tmpDir, cleanup, nil
}
