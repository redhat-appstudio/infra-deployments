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
//
// The second return value is true when one or more upstream kustomization files
// had an empty patch (GitHub omits patch data for very large or renamed files).
// Callers should treat this the same way they treat an API failure — detection
// may be incomplete, so "no bumps found" cannot be stated with confidence.
func ExtractServiceBumps(files []FileChange) ([]ServiceBump, bool) {
	seen := make(map[string]bool)
	var bumps []ServiceBump
	hasSkipped := false

	for _, f := range files {
		if !isUpstreamKustomization(f.Filename) {
			continue
		}
		if f.Patch == "" {
			// Upstream kustomization file present but patch data unavailable —
			// we cannot tell whether its ref changed, so flag incomplete results.
			hasSkipped = true
			continue
		}
		matchedBase, oldSHA, newSHA := extractRefChange(f.Patch)
		if oldSHA == "" || newSHA == "" {
			continue
		}
		// Derive owner/repo from the specific URL that changed, not from any
		// github.com URL in the diff — prevents misattribution when a file
		// references multiple repositories.
		owner, repo := extractGitHubRepoFromURL(matchedBase)
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
	return bumps, hasSkipped
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
// URLs that change independently.
//
// Returns the matched URL base together with the old and new SHAs so callers
// can derive owner/repo from the exact URL that changed. Returns ("","","") if
// no matching pair is found.
func extractRefChange(patch string) (matchedBase, oldSHA, newSHA string) {
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
				return base, old, m[1]
			}
		}
	}
	return "", "", ""
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

// extractGitHubRepoFromURL extracts the owner and repository name from a
// github.com URL string (e.g. "https://github.com/konflux-ci/build-service/config").
// Returns ("","") when the URL is not a github.com URL or does not match the
// expected path structure.
func extractGitHubRepoFromURL(rawURL string) (owner, repo string) {
	if m := githubRepoPattern.FindStringSubmatch(rawURL); m != nil {
		return m[1], m[2]
	}
	return "", ""
}
