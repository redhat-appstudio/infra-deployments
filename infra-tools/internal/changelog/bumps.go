package changelog

import (
	"regexp"
	"strings"
)

// ServiceBump describes a sub-service whose pinned commit SHA changed between
// the old and new operator versions.
type ServiceBump struct {
	// Owner is the GitHub organisation that owns the service repo (e.g. "konflux-ci").
	Owner string
	// Repo is the GitHub repository name (e.g. "build-service", "integration-service").
	Repo string
	// OldSHA is the previously pinned commit.
	OldSHA string
	// NewSHA is the newly pinned commit.
	NewSHA string
}

// refPattern matches a ?ref=<40-char-sha> in a unified diff line.
var refPattern = regexp.MustCompile(`\?ref=([0-9a-f]{40})`)

// githubRepoPattern extracts the owner and repo from a github.com URL.
// Matches: https://github.com/{owner}/{repo}/...
var githubRepoPattern = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/`)

// ExtractServiceBumps scans a list of FileChanges (from the operator compare)
// for kustomization files under operator/upstream-kustomizations/ that contain
// ?ref= SHA changes, and returns one ServiceBump per service that moved.
//
// A file is considered a bump when a removed diff line (-) and an added diff
// line (+) each contain a ?ref=<sha> and the SHAs differ.
//
// Duplicate (owner, repo) pairs are deduplicated — only the first occurrence
// is kept, since the same service can be referenced from multiple kustomization
// files (e.g. core/ and internal-services/).
func ExtractServiceBumps(files []FileChange) []ServiceBump {
	seen := make(map[string]bool) // key: "owner/repo"
	var bumps []ServiceBump

	for _, f := range files {
		if !isUpstreamKustomization(f.Filename) {
			continue
		}
		if f.Patch == "" {
			continue
		}

		oldSHA, newSHA := extractRefChange(f.Patch)
		if oldSHA == "" || newSHA == "" || oldSHA == newSHA {
			continue
		}

		owner, repo := extractGitHubRepo(f.Patch)
		if owner == "" || repo == "" {
			continue
		}

		key := owner + "/" + repo
		if seen[key] {
			continue
		}
		seen[key] = true

		bumps = append(bumps, ServiceBump{
			Owner:  owner,
			Repo:   repo,
			OldSHA: oldSHA,
			NewSHA: newSHA,
		})
	}

	return bumps
}

// isUpstreamKustomization reports whether the given file path is a
// kustomization file under operator/upstream-kustomizations/.
func isUpstreamKustomization(filename string) bool {
	if !strings.HasPrefix(filename, "operator/upstream-kustomizations/") {
		return false
	}
	return strings.HasSuffix(filename, "/kustomization.yaml") || strings.HasSuffix(filename, "/kustomization.yml")
}

// extractRefChange scans unified diff patch lines for a removed (-) and added
// (+) ?ref=<sha> pair. Returns ("", "") if no such pair is found.
func extractRefChange(patch string) (oldSHA, newSHA string) {
	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 {
			continue
		}
		matches := refPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		sha := matches[1]
		switch line[0] {
		case '-':
			if oldSHA == "" {
				oldSHA = sha
			}
		case '+':
			if newSHA == "" {
				newSHA = sha
			}
		}
	}
	return oldSHA, newSHA
}

// extractGitHubRepo scans the patch for a github.com URL on an added (+) line
// and returns the owner and repo.
func extractGitHubRepo(patch string) (owner, repo string) {
	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 || line[0] != '+' {
			continue
		}
		matches := githubRepoPattern.FindStringSubmatch(line)
		if matches != nil {
			return matches[1], matches[2]
		}
	}
	return "", ""
}
