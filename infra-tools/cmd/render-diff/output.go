package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/appset"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/detector"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/renderdiff"
)

// runLocal handles the default local output mode with progressive output.
func runLocal(ctx context.Context, engine *renderdiff.Engine, affected map[detector.Environment][]appset.ComponentPath, colorMode string, openDiff bool, outputDir string) error {
	useColor := shouldUseColor(colorMode)

	if outputDir != "" {
		// Write to directory mode
		result, err := engine.Run(ctx, affected)
		if err != nil {
			return fmt.Errorf("running render-diff engine: %w", err)
		}
		if err := writeDiffFiles(result, outputDir); err != nil {
			return fmt.Errorf("writing diff files: %w", err)
		}
		printSummary(result)
		return nil
	}

	if openDiff {
		// Open in external diff tool
		result, err := engine.Run(ctx, affected)
		if err != nil {
			return fmt.Errorf("running render-diff engine: %w", err)
		}
		if err := openInDiffTool(result); err != nil {
			return fmt.Errorf("opening diff tool: %w", err)
		}
		return nil
	}

	// Progressive output to stdout. Wait for the printer before printing the
	// summary, otherwise the summary can overtake the last diffs (or the
	// process can exit before they are written).
	ch := make(chan renderdiff.ComponentDiff, 10)
	printerDone := make(chan struct{})
	go func() {
		defer close(printerDone)
		for cd := range ch {
			printComponentDiff(cd, useColor)
		}
	}()

	result, err := engine.RunProgressive(ctx, affected, ch)
	<-printerDone
	if err != nil {
		return fmt.Errorf("running render-diff engine: %w", err)
	}
	printSummary(result)
	return nil
}

// printComponentDiff prints a single component's diff to stdout.
func printComponentDiff(cd renderdiff.ComponentDiff, useColor bool) {
	if cd.SkipOutput {
		return
	}
	if cd.Error != "" {
		header := fmt.Sprintf("=== %s (%s) === BUILD ERROR", cd.Path, cd.Env)
		if useColor {
			fmt.Printf("\033[1;31m%s\033[0m\n", header)
			fmt.Printf("\033[31m%s\033[0m\n", cd.Error)
		} else {
			fmt.Println(header)
			fmt.Println(cd.Error)
		}
		fmt.Println()
		return
	}
	header := fmt.Sprintf("=== %s (%s) === +%d -%d", cd.Path, cd.Env, cd.Added, cd.Removed)
	if useColor {
		fmt.Printf("\033[1;36m%s\033[0m\n", header)
		colorDiff(cd.Diff)
	} else {
		fmt.Println(header)
		fmt.Print(cd.Diff)
	}
	fmt.Println()
}

// colorDiff prints a unified diff with ANSI colors.
func colorDiff(diff string) {
	for _, line := range strings.Split(diff, "\n") {
		if len(line) == 0 {
			fmt.Println()
			continue
		}
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			fmt.Printf("\033[1m%s\033[0m\n", line)
		case strings.HasPrefix(line, "@@"):
			fmt.Printf("\033[36m%s\033[0m\n", line)
		case line[0] == '+':
			fmt.Printf("\033[32m%s\033[0m\n", line)
		case line[0] == '-':
			fmt.Printf("\033[31m%s\033[0m\n", line)
		default:
			fmt.Println(line)
		}
	}
}

// printSummary prints aggregate statistics.
func printSummary(result *renderdiff.DiffResult) {
	if len(result.Diffs) == 0 {
		fmt.Println("\nNo render differences detected.")
		return
	}

	fmt.Println("\n--- Summary ---")
	sortDiffs(result.Diffs)
	for _, d := range result.Diffs {
		if d.SkipOutput {
			continue
		}
		if d.Error != "" {
			fmt.Printf("  %s (%s): BUILD ERROR\n", d.Path, d.Env)
		} else {
			fmt.Printf("  %s (%s): +%d -%d\n", d.Path, d.Env, d.Added, d.Removed)
		}
	}
	fmt.Printf("\nTotal: %d components, +%d -%d lines\n", len(result.Diffs), result.TotalAdded, result.TotalRemoved)
}

// shouldUseColor determines whether to use ANSI colors based on the --color flag.
func shouldUseColor(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}
