package changelog

import (
	"regexp"
	"strings"
)

// ServiceBump describes an upstream sub-service whose pinned SHA changed
// between the old and new operator versions.
type ServiceBump struct {
	Owner  string // GitHub org, e.g. "konflux-ci"
	Repo   string // repository name, e.g. "build-service"
	OldSHA string // previously pinned 40-char commit SHA
	NewSHA string // newly pinned 40-char commit SHA
}

var (
	// refPattern matches ?ref=<40-char-sha> in a unified diff line.
	refPattern = regexp.MustCompile(`\?ref=([0-9a-f]{40})`)
	// githubRepoPattern extracts owner and repo from a github.com URL.
	githubRepoPattern = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/`)
)

// ExtractServiceBumps scans file changes from the operator repo comparison
// for kustomization files under operator/upstream-kustomizations/ that have
// a ?ref=<sha> change, and returns one ServiceBump per bumped service.
//
// Duplicate (owner, repo) pairs are deduplicated — only the first occurrence
// is kept, since the same service can appear in multiple kustomization files.
func ExtractServiceBumps(files []FileChange) []ServiceBump {
	seen := make(map[string]bool)
	var bumps []ServiceBump

	for _, f := range files {
		if !isUpstreamKustomization(f.Filename) || f.Patch == "" {
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
		bumps = append(bumps, ServiceBump{Owner: owner, Repo: repo, OldSHA: oldSHA, NewSHA: newSHA})
	}
	return bumps
}

// isUpstreamKustomization reports whether path is a kustomization file under
// operator/upstream-kustomizations/ in the operator repo.
func isUpstreamKustomization(filename string) bool {
	return strings.HasPrefix(filename, "operator/upstream-kustomizations/") &&
		(strings.HasSuffix(filename, "/kustomization.yaml") || strings.HasSuffix(filename, "/kustomization.yml"))
}

// extractRefChange scans a unified diff for a removed (-) and added (+)
// ?ref=<40-char-sha> pair that share the same URL base (everything before ?ref=).
// Pairing by URL base avoids false matches when a file contains multiple resource
// URLs that change independently. Returns ("", "") if no matching pair is found.
func extractRefChange(patch string) (oldSHA, newSHA string) {
	lines := strings.Split(patch, "\n")

	// First pass: collect removed refs keyed by URL base.
	removed := make(map[string]string) // urlBase → sha
	for _, line := range lines {
		if len(line) == 0 || line[0] != '-' {
			continue
		}
		m := refPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if base := urlBase(line); base != "" && removed[base] == "" {
			removed[base] = m[1]
		}
	}

	// Second pass: find an added ref whose URL base matches a removed one.
	for _, line := range lines {
		if len(line) == 0 || line[0] != '+' {
			continue
		}
		m := refPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if base := urlBase(line); base != "" {
			if old, ok := removed[base]; ok && old != m[1] {
				return old, m[1]
			}
		}
	}
	return "", ""
}

// urlBase returns the portion of a diff line before the ?ref= query parameter,
// stripping the leading +/- diff marker and surrounding whitespace.
func urlBase(line string) string {
	idx := strings.Index(line, "?ref=")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[1:idx]) // line[0] is +/-
}

// extractGitHubRepo scans added (+) diff lines for a github.com URL and
// returns the owner and repository name.
func extractGitHubRepo(patch string) (owner, repo string) {
	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 || line[0] != '+' {
			continue
		}
		if m := githubRepoPattern.FindStringSubmatch(line); m != nil {
			return m[1], m[2]
		}
	}
	return "", ""
}
