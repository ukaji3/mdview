package renderer

import (
	"regexp"
)

// ansiEscapeRe matches ANSI CSI sequences (e.g., \033[0m, \033[38;2;r;g;bm)
// and OSC sequences (e.g., \033]...\033\\).
var ansiEscapeRe = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)

// sixelDCSRe matches Sixel DCS sequences: \033P ... \033\\
// The Sixel data can be very large, so we use a non-greedy match.
var sixelDCSRe = regexp.MustCompile(`\033P[^\033]*\033\\`)

// otherEscapeRe matches any remaining escape sequences that start with \033
// followed by a single character (e.g., \033(B for character set selection).
var otherEscapeRe = regexp.MustCompile(`\033[^\[\]P][^\033]*`)

// StripANSI removes all ANSI escape sequences from the given string.
// This includes:
//   - CSI sequences (\033[...m, \033[...H, etc.)
//   - Sixel DCS sequences (\033P...\033\\)
//   - Other escape sequences
//
// Used when ColorMode is ColorNone (NO_COLOR set or pipe output).
func StripANSI(s string) string {
	// First remove Sixel DCS sequences (they can be very large)
	s = sixelDCSRe.ReplaceAllString(s, "")
	// Then remove ANSI CSI sequences
	s = ansiEscapeRe.ReplaceAllString(s, "")
	// Remove any remaining escape sequences
	s = otherEscapeRe.ReplaceAllString(s, "")
	return s
}
