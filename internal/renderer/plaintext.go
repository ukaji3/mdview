package renderer

import "strings"

// StripANSI removes all ANSI escape sequences from the given string.
// This includes:
//   - CSI sequences (\033[...m, \033[...H, etc.)
//   - OSC sequences (\033]...\x07 or \033]...\033\\)
//   - DCS sequences (\033P...\033\\)
//   - APC sequences (\033_...\033\\)
//   - Simple escape sequences (\033X)
//
// Used when ColorMode is ColorNone (NO_COLOR set or pipe output).
// Implemented as a single-pass state machine (no regexp) for speed.
func StripANSI(s string) string {
	if !strings.ContainsRune(s, '\033') {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	i := 0
	n := len(s)
	for i < n {
		c := s[i]
		if c != '\033' {
			b.WriteByte(c)
			i++
			continue
		}
		// Escape sequence
		if i+1 >= n {
			// Trailing lone ESC: drop it
			break
		}
		switch s[i+1] {
		case '[': // CSI: until final byte in @-~
			i += 2
			for i < n && !(s[i] >= '@' && s[i] <= '~') {
				i++
			}
			if i < n {
				i++
			}
		case ']': // OSC: until BEL or ST
			i += 2
			for i < n {
				if s[i] == '\x07' {
					i++
					break
				}
				if s[i] == '\033' && i+1 < n && s[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		case 'P', '_': // DCS / APC: until ST
			i += 2
			for i < n {
				if s[i] == '\033' && i+1 < n && s[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default: // Simple escape sequence: ESC + one byte
			i += 2
		}
	}
	return b.String()
}
