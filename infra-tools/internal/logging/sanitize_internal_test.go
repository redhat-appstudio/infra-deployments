package logging

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSanitize(t *testing.T) {
	g := NewWithT(t)

	g.Expect(Sanitize("plain path/file.yaml")).To(Equal("plain path/file.yaml"))
	// Workflow command prefixes.
	g.Expect(Sanitize("##[error]boom")).To(Equal("# #[error]boom"))
	g.Expect(Sanitize("a::warning::b")).To(Equal("a: :warning: :b"))
	// Control characters and line/paragraph separators would split the line.
	g.Expect(Sanitize("line\nbreak\ttab")).To(Equal("line break tab"))
	g.Expect(Sanitize("a\u2028b\u2029c")).To(Equal("a b c"))
	// Format characters (Cf), such as a zero-width joiner, are removed too.
	g.Expect(Sanitize("a\u200db")).To(Equal("a b"))
}
