package git //nolint:testpackage // tests unexported splitNullTerminated

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSplitNullTerminated(t *testing.T) {
	g := NewWithT(t)

	g.Expect(splitNullTerminated(nil)).To(BeEmpty())
	g.Expect(splitNullTerminated([]byte("a\x00b\nc\x00"))).To(ConsistOf("a", "b\nc"))
	g.Expect(splitNullTerminated([]byte(" leading-space\x00"))).To(ConsistOf(" leading-space"))
}
