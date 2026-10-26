package main

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFormatUnreferencedChangesSanitizesPaths(t *testing.T) {
	g := NewWithT(t)

	msg := formatUnreferencedChanges([]string{
		"components/x/`\n::error::boom.yaml",
		"components/x/##[warning]fake.yaml",
		"components/x/\u202eevil.yaml",
	})

	g.Expect(msg).To(ContainSubstring("- `components/x/' : :error: :boom.yaml`\n"))
	g.Expect(msg).NotTo(ContainSubstring("::"))
	g.Expect(msg).NotTo(ContainSubstring("##["))
	g.Expect(msg).NotTo(ContainSubstring("\u202e"))
}

func TestSanitizeLineNeutralizesWorkflowCommands(t *testing.T) {
	g := NewWithT(t)
	g.Expect(sanitizeLine("##[warning]fake")).To(Equal("# #[warning]fake"))
	g.Expect(sanitizeLine("::error::fake")).To(Equal(": :error: :fake"))
}
