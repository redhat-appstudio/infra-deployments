package main

import (
	"context"
	"log/slog"
	"strings"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// OutputMode controls how render-diff formats and delivers its output.
type OutputMode string

const (
	OutputModeLocal      OutputMode = "local"
	OutputModeCISummary  OutputMode = "ci-summary"
	OutputModeCIComment  OutputMode = "ci-comment"
	OutputModeCIArtifact OutputMode = "ci-artifact-dir"
)

// runAllOutputModes runs every configured output mode against the given result.
// Returns true if any mode failed.
func runAllOutputModes(ctx context.Context, modes []OutputMode, result *renderdiff.DiffResult, colorMode string, openDiff bool, outputDir, headSHA, baseSHA string) bool {
	var hadError bool
	for _, m := range modes {
		if err := runOutputMode(ctx, m, result, colorMode, openDiff, outputDir, headSHA, baseSHA); err != nil {
			slog.Error("output mode failed", "mode", m, "err", err)
			hadError = true
		}
	}
	return hadError
}

// parseOutputModes splits a comma-separated output-mode string, validates each
// value, and returns the deduplicated list. Returns nil if any value is invalid.
func parseOutputModes(raw string) []OutputMode {
	seen := make(map[OutputMode]bool)
	var modes []OutputMode
	for s := range strings.SplitSeq(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		m := OutputMode(s)
		switch m {
		case OutputModeLocal, OutputModeCISummary, OutputModeCIComment, OutputModeCIArtifact:
			if !seen[m] {
				seen[m] = true
				modes = append(modes, m)
			}
		default:
			return nil
		}
	}
	return modes
}
