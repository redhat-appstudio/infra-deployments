package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// writeDiffFiles writes per-component .diff files to a directory.
func writeDiffFiles(result *renderdiff.DiffResult, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	seen := make(map[string]int)
	for _, d := range result.Diffs {
		if d.Error != "" || d.Diff == "" {
			continue
		}
		name := dedupeFileName(diffFileName(d), seen)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(d.Diff), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return nil
}

// diffFileName converts a component diff to a human-readable filename.
// When ClusterDir is set, it takes priority as the most recognizable label
// (e.g., "stone-prod-p02--production.diff"). Otherwise, falls back to the
// full path slug (e.g., "components__foo__staging__staging.diff").
func diffFileName(cd renderdiff.ComponentDiff) string {
	if cd.ClusterDir != "" {
		return fmt.Sprintf("%s--%s.diff", cd.ClusterDir, cd.Env)
	}
	safe := strings.ReplaceAll(cd.Path, "/", "__")
	return fmt.Sprintf("%s__%s.diff", safe, cd.Env)
}

// dedupeFileName returns a unique filename within the seen set. If the base
// name has already been used, it appends a counter suffix (-2, -3, etc.).
func dedupeFileName(base string, seen map[string]int) string {
	seen[base]++
	if seen[base] == 1 {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s-%d%s", stem, seen[base], ext)
}

// openInDiffTool writes all base and head YAML files into two temporary
// directories and opens them in the user's preferred diff tool for a
// side-by-side folder comparison. Files are named after their component
// and environment so they are easy to identify.
func openInDiffTool(result *renderdiff.DiffResult) error {
	if len(result.Diffs) == 0 {
		fmt.Println("No render differences to display.")
		return nil
	}

	// Create temp directories for the base and head YAML files.
	// These are intentionally not cleaned up: GUI diff tools like meld may
	// return immediately while still reading the files, and keeping them
	// lets the user re-inspect after the tool closes. The OS cleans /tmp.
	baseDir, err := os.MkdirTemp("", "render-diff-base-*")
	if err != nil {
		return fmt.Errorf("creating base temp dir: %w", err)
	}

	headDir, err := os.MkdirTemp("", "render-diff-head-*")
	if err != nil {
		return fmt.Errorf("creating head temp dir: %w", err)
	}

	seen := make(map[string]int)
	for _, d := range result.Diffs {
		if d.Error != "" || d.Diff == "" {
			continue
		}
		name := dedupeFileName(diffFileName(d), seen)
		// Replace .diff extension with .yaml for clarity in the diff tool.
		name = strings.TrimSuffix(name, ".diff") + ".yaml"

		if err := os.WriteFile(filepath.Join(baseDir, name), d.BaseYAML, 0o644); err != nil {
			return fmt.Errorf("writing base file for %s: %w", d.Path, err)
		}
		if err := os.WriteFile(filepath.Join(headDir, name), d.HeadYAML, 0o644); err != nil {
			return fmt.Errorf("writing head file for %s: %w", d.Path, err)
		}
	}

	toolName := os.Getenv("DIFFTOOL")
	var cmd *exec.Cmd
	if toolName != "" {
		cmd = exec.Command(toolName, baseDir, headDir)
	} else {
		cmd = exec.Command("git", "difftool", "--no-index", "--dir-diff", baseDir, headDir)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Opening folder diff: %s vs %s\n", baseDir, headDir)
	if err := cmd.Run(); err != nil {
		// diff tools return non-zero when files differ, which is expected
		slog.Debug("diff tool exited", "err", err)
	}
	return nil
}

// sortDiffs sorts diffs by environment then path for consistent output.
func sortDiffs(diffs []renderdiff.ComponentDiff) {
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Env != diffs[j].Env {
			return diffs[i].Env < diffs[j].Env
		}
		return diffs[i].Path < diffs[j].Path
	})
}
