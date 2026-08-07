package renderer

import (
	"strings"
)

// wrapANSI wraps ANSI-decorated text to the given display width.
// Escape sequences (CSI/OSC/DCS/APC) are treated as zero-width and are never
// split. Wrapping prefers word boundaries (spaces) for Latin text and breaks
// at character boundaries for wide (CJK) characters. Existing newlines are
// preserved. SGR state persists across terminal lines, so no style
// save/restore is needed at break points.
func wrapANSI(s string, width int) string {
	if width <= 0 || !strings.ContainsAny(s, " ") && stringW(s) <= width {
		// Cheap pre-check is not exact for ANSI text; fall through when unsure.
		if width <= 0 {
			return s
		}
	}

	var out strings.Builder
	out.Grow(len(s) + 16)

	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			out.WriteByte('\n')
		}
		wrapANSILine(&out, line, width)
	}
	return out.String()
}

// wrapANSILine wraps a single line (no embedded newlines) into out.
func wrapANSILine(out *strings.Builder, line string, width int) {
	runes := []rune(line)
	var cur []rune // current output line
	curWidth := 0
	lastSpace := -1     // index in cur of the last breakable space
	lastSpaceWidth := 0 // display width of cur up to (excluding) lastSpace

	flushAt := func(breakIdx int, breakWidth int) {
		// Emit cur[:breakIdx] as a completed line, drop one leading space of
		// the remainder, and continue with the rest.
		out.WriteString(string(cur[:breakIdx]))
		out.WriteByte('\n')
		rest := cur[breakIdx:]
		if len(rest) > 0 && rest[0] == ' ' {
			rest = rest[1:]
		}
		cur = append([]rune(nil), rest...)
		curWidth = curWidth - breakWidth
		if len(rest) < len(cur) { // space removed
			curWidth--
		}
		// Recompute width of remainder precisely (safe and simple).
		curWidth = displayWidthRunes(cur)
		lastSpace = -1
		lastSpaceWidth = 0
	}

	i := 0
	for i < len(runes) {
		r := runes[i]

		// Copy escape sequences verbatim (zero display width).
		if r == '\033' && i+1 < len(runes) {
			j := skipEscape(runes, i)
			cur = append(cur, runes[i:j]...)
			i = j
			continue
		}

		rw := runeW(r)

		if curWidth+rw > width && curWidth > 0 {
			if r == ' ' {
				// Natural break: emit current line, skip the space.
				out.WriteString(string(cur))
				out.WriteByte('\n')
				cur = cur[:0]
				curWidth = 0
				lastSpace = -1
				lastSpaceWidth = 0
				i++
				continue
			}
			if lastSpace >= 0 {
				// Break at the last space.
				flushAt(lastSpace, lastSpaceWidth)
			} else {
				// No breakable space: hard break here.
				out.WriteString(string(cur))
				out.WriteByte('\n')
				cur = cur[:0]
				curWidth = 0
			}
		}

		if r == ' ' {
			lastSpace = len(cur)
			lastSpaceWidth = curWidth
		}
		cur = append(cur, r)
		curWidth += rw
		i++
	}
	out.WriteString(string(cur))
}

// skipEscape returns the index just past the escape sequence starting at i
// (runes[i] must be ESC).
func skipEscape(runes []rune, i int) int {
	j := i + 1
	if j >= len(runes) {
		return j
	}
	switch runes[j] {
	case '[': // CSI: until final byte in @-~
		j++
		for j < len(runes) && !(runes[j] >= '@' && runes[j] <= '~') {
			j++
		}
		if j < len(runes) {
			j++
		}
	case ']': // OSC: until BEL or ST
		j++
		for j < len(runes) {
			if runes[j] == '\x07' {
				j++
				break
			}
			if runes[j] == '\033' && j+1 < len(runes) && runes[j+1] == '\\' {
				j += 2
				break
			}
			j++
		}
	case 'P', '_': // DCS / APC: until ST
		j++
		for j < len(runes) {
			if runes[j] == '\033' && j+1 < len(runes) && runes[j+1] == '\\' {
				j += 2
				break
			}
			j++
		}
	default:
		j++
	}
	return j
}

// displayWidthRunes computes the display width of runes, skipping escapes.
func displayWidthRunes(runes []rune) int {
	w := 0
	i := 0
	for i < len(runes) {
		if runes[i] == '\033' && i+1 < len(runes) {
			i = skipEscape(runes, i)
			continue
		}
		w += runeW(runes[i])
		i++
	}
	return w
}
