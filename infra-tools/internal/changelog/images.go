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

	// newNameLinePattern matches "newName: <value>".
	newNameLinePattern = regexp.MustCompile(`newName:\s+(\S+)`)

	// imageNameLinePattern matches "name: <value>" but NOT "newName: <value>".
	// \b ensures the word "name" is not preceded by another word character (e.g. "new").
	imageNameLinePattern = regexp.MustCompile(`\bname:\s+(\S+)`)
)

// ExtractImageDigestChanges scans file changes from the operator repo comparison
// for kustomization files under operator/upstream-kustomizations/ that have a
// digest: sha256:... change in their images: block, and returns one
// ImageDigestChange per image whose digest changed.
//
// Each images: list entry may list fields in any order (e.g. digest-before-name
// or name-before-digest). The parser groups diff lines into per-entry blocks
// and collects name/newName/digest fields without relying on ordering.
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
// Lines are grouped into YAML list-item blocks (one per images: entry). Within
// each block, name/newName/digest fields are collected regardless of order.
// newName is preferred over name for the registry reference.
func extractDigestChanges(patch string) []ImageDigestChange {
	var out []ImageDigestChange
	var block []string

	flush := func() {
		if change, ok := digestChangeFromBlock(block); ok {
			out = append(out, change)
		}
		block = nil
	}

	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 {
			continue
		}
		// Hunk headers must not carry state across unrelated sections.
		if strings.HasPrefix(line, "@@") {
			flush()
			continue
		}
		if line[0] != '-' && line[0] != '+' && line[0] != ' ' {
			continue
		}
		rest := line[1:]
		if len(block) > 0 && isImageListItemStart(line[0], rest) {
			flush()
		}
		block = append(block, line)
	}
	flush()
	return out
}

// isImageListItemStart reports whether a diff line body starts a new images:
// YAML list item (e.g. "- name:", "- digest:", "- newName:").
//
// Added-side digest-first replacements appear as "+- digest:"; those belong to
// the same list item as the preceding "- digest:" removal and must not split
// the block.
func isImageListItemStart(prefix byte, rest string) bool {
	trimmed := strings.TrimSpace(rest)
	if !strings.HasPrefix(trimmed, "- ") {
		return false
	}
	field := strings.TrimSpace(trimmed[2:])
	isDigestField := strings.HasPrefix(field, "digest:")
	if prefix == '+' && isDigestField {
		return false
	}
	return strings.HasPrefix(field, "name:") ||
		strings.HasPrefix(field, "newName:") ||
		isDigestField
}

func digestChangeFromBlock(lines []string) (ImageDigestChange, bool) {
	var oldDigest, newDigest, name, newName string

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		prefix := line[0]
		rest := line[1:]

		if m := newNameLinePattern.FindStringSubmatch(rest); m != nil {
			newName = m[1]
		}
		if m := imageNameLinePattern.FindStringSubmatch(rest); m != nil {
			name = m[1]
		}
		switch prefix {
		case '-':
			if m := digestLinePattern.FindStringSubmatch(rest); m != nil {
				oldDigest = m[1]
			}
		case '+':
			if m := digestLinePattern.FindStringSubmatch(rest); m != nil {
				newDigest = m[1]
			}
		}
	}

	imageName := newName
	if imageName == "" {
		imageName = name
	}
	if imageName == "" || oldDigest == "" || newDigest == "" || oldDigest == newDigest {
		return ImageDigestChange{}, false
	}
	return ImageDigestChange{
		ImageName: imageName,
		OldDigest: oldDigest,
		NewDigest: newDigest,
	}, true
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
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference %s: %w", imageRef, err)
	}

	var labels map[string]string
	err = retryDo(ctx, 3, func() error {
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
