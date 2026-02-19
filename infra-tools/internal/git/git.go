// Package git provides helpers for interacting with a local git repository.
package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

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
// It uses a two-point diff (content diff between trees) rather than three-dot
// (merge-base diff) to capture ALL differences, including changes that exist
// in baseRef but not yet in HEAD.
func ChangedFiles(ctx context.Context, repoRoot, baseRef string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", baseRef, "HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff %s HEAD: %w", baseRef, err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
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
