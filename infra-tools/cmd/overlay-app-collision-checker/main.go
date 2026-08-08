// Command overlay-app-collision-checker detects ArgoCD ApplicationSets that
// would template colliding generated Application names when their overlays
// are deployed together to the same cluster / ArgoCD control-plane
// namespace (see internal/collision for background). It fails loudly at
// PR/CI time instead of letting the collision surface later as a flaky
// ArgoCD ownership conflict during e2e or in production.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flag"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/collision"
	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/git"
)

func main() {
	repoRoot := flag.String("repo-root", "", "Path to the repository root (default: auto-detect via git)")
	flag.Parse()

	root := *repoRoot
	if root == "" {
		detected, err := git.TopLevel(context.Background())
		if err != nil {
			fatal("auto-detecting repo root; use --repo-root to specify explicitly: %v", err)
		}
		root = detected
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fatal("resolving repo root: %v", err)
	}

	hadCollision := false

	for _, group := range collision.DefaultGroups {
		fmt.Printf("== Checking co-deployed group: %s ==\n", strings.Join(group.OverlayPaths, " "))

		collisions, err := collision.CheckOverlayGroup(absRoot, group)
		if err != nil {
			fatal("checking group %v: %v", group.OverlayPaths, err)
		}

		if len(collisions) == 0 {
			fmt.Println("  OK: no colliding generated Application names")
			continue
		}

		hadCollision = true
		for _, c := range collisions {
			fmt.Println()
			fmt.Printf("COLLISION: generated Application name %q is templated by multiple ApplicationSets:\n", c.AppNameTemplate)
			for _, e := range c.AppSets {
				fmt.Printf("  - ApplicationSet %q in %s\n", e.Name, e.OverlayPath)
			}
		}
	}

	fmt.Println()
	if hadCollision {
		fmt.Println("One or more ApplicationSets would collide when co-deployed to the same cluster.")
		fmt.Println("Fix one of the following before merging:")
		fmt.Println("  - Remove/disable the superseded ApplicationSet from its overlay (see the")
		fmt.Println("    delete-applications.yaml / delete-legacy-konflux-member-appsets.yaml")
		fmt.Println("    patterns used by argo-cd-apps/overlays/development(-operator)).")
		fmt.Println("  - Rename the colliding Application via spec.template.metadata.name so the")
		fmt.Println("    two ApplicationSets no longer template the same generated Application name.")
		os.Exit(1)
	}

	fmt.Println("No overlay Application-name collisions detected.")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
