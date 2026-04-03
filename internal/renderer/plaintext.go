package renderer

import (
	"regexp"
)

// TODO: 5つの正規表現を順次適用しているため非効率です。
// 単一パスのステートマシンによる実装がより効率的ですが、
// 現時点では正確性を優先しています。

var (
	csiRe       = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)
	oscRe       = regexp.MustCompile(`\033\][^\x07]*(?:\x07|\033\\)`)
	dcsRe       = regexp.MustCompile(`\033P[^\033]*\033\\`)
	apcRe       = regexp.MustCompile(`\033_[^\033]*\033\\`)
	simpleEscRe = regexp.MustCompile(`\033[^\[\]P_]`)
)

// StripANSI removes all ANSI escape sequences from the given string.
// This includes:
//   - CSI sequences (\033[...m, \033[...H, etc.)
//   - OSC sequences (\033]...\x07 or \033]...\033\\)
//   - DCS sequences (\033P...\033\\)
//   - APC sequences (\033_...\033\\)
//   - Simple escape sequences (\033X)
//
// Used when ColorMode is ColorNone (NO_COLOR set or pipe output).
func StripANSI(s string) string {
	s = dcsRe.ReplaceAllString(s, "")
	s = apcRe.ReplaceAllString(s, "")
	s = oscRe.ReplaceAllString(s, "")
	s = csiRe.ReplaceAllString(s, "")
	s = simpleEscRe.ReplaceAllString(s, "")
	return s
}
