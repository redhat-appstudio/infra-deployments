package changelog

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ImageDigestChange describes a single entry in a kustomization images: block
// whose digest changed between two operator versions.
type ImageDigestChange struct {
	ImageName string // e.g. "quay.io/konflux-ci/namespace-lister"
	OldDigest string // "sha256:<64 hex chars>"
	NewDigest string // "sha256:<64 hex chars>"
}

// RegistryInspector is the OCI registry surface used to read image labels.
// Defined as an interface so tests can inject a fake without making real
// network calls.
type RegistryInspector interface {
	InspectLabels(ctx context.Context, imageRef string) (map[string]string, error)
}

var (
	// digestLinePattern matches "digest: sha256:<64-hex>" in a diff line body.
	digestLinePattern = regexp.MustCompile(`digest:\s+(sha256:[0-9a-f]{64})`)

	// imageNameLinePattern matches "name: <value>" but NOT "newName: <value>".
	// \b ensures the word "name" is not preceded by another word character (e.g. "new").
	imageNameLinePattern = regexp.MustCompile(`\bname:\s+(\S+)`)
)

// ExtractImageDigestChanges scans file changes from the operator repo comparison
// for kustomization files under operator/upstream-kustomizations/ that have a
// digest: sha256:... change in their images: block, and returns one
// ImageDigestChange per image whose digest changed.
//
// The diff structure for a digest-pinned image is:
//
//	-- digest: sha256:OLD...    (removed line: diff '-' + YAML list item)
//	+- digest: sha256:NEW...    (added line)
//	   name: quay.io/...        (context line following the digest)
//
// The second return value is true when one or more upstream kustomization files
// had an empty patch — callers should treat this the same as ExtractServiceBumps.
func ExtractImageDigestChanges(files []FileChange) ([]ImageDigestChange, bool) {
	var changes []ImageDigestChange
	hasSkipped := false

	for _, f := range files {
		if !isUpstreamKustomization(f.Filename) {
			continue
		}
		if f.Patch == "" {
			hasSkipped = true
			continue
		}
		changes = append(changes, extractDigestChanges(f.Patch)...)
	}
	return changes, hasSkipped
}

// extractDigestChanges parses a single unified diff patch and returns all
// image entries whose digest changed.
//
// Algorithm (single forward pass):
//  1. A '-' line matching digest: sha256:... sets pendingOld.
//  2. A '+' line matching digest: sha256:... sets pendingNew.
//  3. A context ' ' line matching \bname: (not newName:) closes the pending pair
//     and emits an ImageDigestChange.
//  4. A context ' ' line that starts a new YAML list item ("- ") resets the
//     pending state — we crossed an unchanged image entry.
//
// This ordering is correct for the real kustomization files: digest appears
// before name in every images: block entry.
func extractDigestChanges(patch string) []ImageDigestChange {
	var out []ImageDigestChange
	var pendingOld, pendingNew string

	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 {
			continue
		}
		prefix := line[0]
		rest := line[1:]

		switch prefix {
		case '-':
			if m := digestLinePattern.FindStringSubmatch(rest); m != nil {
				pendingOld = m[1]
			}
		case '+':
			if m := digestLinePattern.FindStringSubmatch(rest); m != nil {
				pendingNew = m[1]
			}
		case ' ':
			// New unchanged YAML list item — reset pending and move on.
			if strings.HasPrefix(strings.TrimSpace(rest), "- ") {
				pendingOld, pendingNew = "", ""
				continue
			}
			// Image name context line — emit pending pair if complete.
			// imageNameLinePattern uses \bname: which does not match "newName:".
			if pendingOld != "" && pendingNew != "" && pendingOld != pendingNew {
				if m := imageNameLinePattern.FindStringSubmatch(rest); m != nil {
					out = append(out, ImageDigestChange{
						ImageName: m[1],
						OldDigest: pendingOld,
						NewDigest: pendingNew,
					})
					pendingOld, pendingNew = "", ""
				}
			}
		}
	}
	return out
}

// NewRegistryInspector returns a RegistryInspector backed by the real OCI
// registry using anonymous (unauthenticated) access — sufficient for public
// Konflux images on quay.io.
func NewRegistryInspector() RegistryInspector {
	return &ocRegistryInspector{}
}

type ocRegistryInspector struct{}

// InspectLabels fetches the config labels for the given image reference
// (e.g. "quay.io/konflux-ci/namespace-lister@sha256:...").
// The call is retried up to three times with exponential backoff.
func (r *ocRegistryInspector) InspectLabels(ctx context.Context, imageRef string) (map[string]string, error) {
	var labels map[string]string
	err := retryDo(ctx, 3, func() error {
		ref, err := name.ParseReference(imageRef)
		if err != nil {
			return fmt.Errorf("parsing image reference %s: %w", imageRef, err)
		}
		img, err := remote.Image(ref, remote.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("fetching image %s: %w", imageRef, err)
		}
		cfg, err := img.ConfigFile()
		if err != nil {
			return fmt.Errorf("reading config for %s: %w", imageRef, err)
		}
		if cfg.Config.Labels != nil {
			labels = cfg.Config.Labels
		} else {
			labels = map[string]string{}
		}
		return nil
	})
	return labels, err
}
