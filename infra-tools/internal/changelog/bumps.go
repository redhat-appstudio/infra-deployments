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
// A single kustomization file can reference multiple upstream repositories;
// all changed pairs within a file are detected, not just the first one.
// Duplicate (owner, repo) pairs across files are deduplicated — only the
// first occurrence is kept.
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
		// Iterate ALL ref changes in the patch — a kustomization file can pin
		// several upstream services; each changed URL base is its own bump.
		for _, change := range extractRefChanges(f.Patch) {
			// Derive owner/repo from the specific URL that changed, not from any
			// github.com URL in the diff — prevents misattribution.
			owner, repo := extractGitHubRepoFromURL(change.base)
			if owner == "" || repo == "" {
				continue
			}
			key := owner + "/" + repo
			if seen[key] {
				continue
			}
			seen[key] = true
			bumps = append(bumps, ServiceBump{Owner: owner, Repo: repo, OldSHA: change.oldSHA, NewSHA: change.newSHA})
		}
	}
	return bumps, hasSkipped
}

// isUpstreamKustomization reports whether path is a kustomization file under
// operator/upstream-kustomizations/ in the operator repo.
func isUpstreamKustomization(filename string) bool {
	return strings.HasPrefix(filename, "operator/upstream-kustomizations/") &&
		(strings.HasSuffix(filename, "/kustomization.yaml") || strings.HasSuffix(filename, "/kustomization.yml"))
}

// refChange holds one matched old→new SHA pair found in a patch, together with
// the URL base (everything before ?ref=) that identifies which resource changed.
type refChange struct {
	base   string
	oldSHA string
	newSHA string
}

// extractRefChanges scans a unified diff and returns ALL removed(-)/added(+)
// ?ref=<40-char-sha> pairs that share the same URL base.
//
// Pairing by URL base is the key correctness invariant: it ensures that when a
// kustomization file lists several upstream services, each service's old→new SHA
// is matched to its own URL rather than to a SHA from a different service.
//
// The `removed` map is consumed as matches are found, so the same base cannot
// produce multiple pairs even if the diff is malformed.
func extractRefChanges(patch string) []refChange {
	lines := strings.Split(patch, "\n")

	// First pass: collect every removed (-) ref, keyed by URL base.
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

	// Second pass: collect every added (+) ref whose URL base was also removed,
	// and whose SHA actually changed. Deleting from `removed` after each match
	// prevents the same base from being emitted more than once.
	var changes []refChange
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
				changes = append(changes, refChange{base: base, oldSHA: old, newSHA: m[1]})
				delete(removed, base)
			}
		}
	}
	return changes
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
