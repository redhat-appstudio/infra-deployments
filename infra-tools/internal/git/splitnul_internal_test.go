package git

import (
	"testing"

	. "github.com/onsi/gomega"
)

// NUL-separated output must survive the shapes git permits in filenames.
func TestSplitNul(t *testing.T) {
	g := NewWithT(t)

	g.Expect(splitNul(nil)).To(BeEmpty())
	g.Expect(splitNul([]byte("a\x00b\nc\x00"))).To(ConsistOf("a", "b\nc"))
	g.Expect(splitNul([]byte(" leading-space\x00"))).To(ConsistOf(" leading-space"))
}
