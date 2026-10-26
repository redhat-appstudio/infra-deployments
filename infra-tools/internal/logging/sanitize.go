package logging

import (
	"strings"
	"unicode"
)

// Sanitize removes characters that could break out of a log line or a GitHub
// Actions workflow command. Use it for PR-controlled values before logging.
func Sanitize(s string) string {
	// Both workflow-command syntaxes are matched anywhere in a line.
	s = strings.ReplaceAll(s, "##[", "# #[")
	s = strings.ReplaceAll(s, "::", ": :")
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) ||
			r == '\u2028' || r == '\u2029' {
			return ' '
		}
		return r
	}, s)
}
