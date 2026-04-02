package wrap

import (
	"strings"
	"unicode/utf8"
)

// isCJK returns true if the rune is a CJK character that should be wrapped
// at character boundaries and occupies 2 columns.
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0xFF00 && r <= 0xFFEF) // Fullwidth Forms
}

// runeWidth returns the display width of a rune.
// CJK characters are 2 columns wide; all others are 1.
func runeWidth(r rune) int {
	if isCJK(r) {
		return 2
	}
	return 1
}

// stringWidth returns the display width of a string,
// accounting for double-width CJK characters.
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// Wrap wraps text to the given width.
// For ASCII/Latin text, wrapping occurs at word boundaries (spaces).
// For CJK characters, wrapping occurs at character boundaries.
// Existing newlines are preserved.
// If a single word is longer than width, it is forcibly broken.
func Wrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if stringWidth(line) <= width {
			result = append(result, line)
			continue
		}
		wrapped := wrapLine(line, width)
		result = append(result, wrapped...)
	}

	return strings.Join(result, "\n")
}

// wrapLine wraps a single line (no embedded newlines) to the given width.
func wrapLine(line string, width int) []string {
	var result []string
	var current strings.Builder
	currentWidth := 0

	runes := []rune(line)
	i := 0

	for i < len(runes) {
		r := runes[i]

		if isCJK(r) {
			rw := runeWidth(r)
			if currentWidth+rw > width {
				// Emit current line and start new one
				result = append(result, current.String())
				current.Reset()
				currentWidth = 0
			}
			current.WriteRune(r)
			currentWidth += rw
			i++
		} else if r == ' ' {
			// If adding the space would exceed width, break here
			if currentWidth+1 > width {
				result = append(result, current.String())
				current.Reset()
				currentWidth = 0
				i++ // skip the space at line break
				continue
			}
			// Look ahead: peek at the next word to decide whether to add the space
			nextWordStart := i + 1
			if nextWordStart < len(runes) {
				nextWord := collectWord(runes, nextWordStart)
				nextWordWidth := stringWidth(nextWord)
				// If next char is CJK, its width is 2
				if nextWord == "" && nextWordStart < len(runes) && isCJK(runes[nextWordStart]) {
					nextWordWidth = runeWidth(runes[nextWordStart])
				}
				if nextWord != "" && currentWidth+1+nextWordWidth > width {
					// Next word won't fit even with the space; break now, skip the space
					result = append(result, current.String())
					current.Reset()
					currentWidth = 0
					i++ // skip the space
					continue
				}
			}
			current.WriteRune(r)
			currentWidth++
			i++
		} else {
			// Latin/ASCII word: collect the whole word
			word := collectWord(runes, i)
			wordWidth := stringWidth(word)

			if currentWidth == 0 {
				// Start of line
				if wordWidth <= width {
					current.WriteString(word)
					currentWidth += wordWidth
				} else {
					// Word is longer than width: force break
					broken := forceBreak(word, width)
					for j, part := range broken {
						if j < len(broken)-1 {
							result = append(result, part)
						} else {
							current.WriteString(part)
							currentWidth = stringWidth(part)
						}
					}
				}
			} else if currentWidth+wordWidth <= width {
				// Word fits on current line
				current.WriteString(word)
				currentWidth += wordWidth
			} else {
				// Word doesn't fit: wrap
				result = append(result, current.String())
				current.Reset()
				currentWidth = 0

				if wordWidth <= width {
					current.WriteString(word)
					currentWidth = wordWidth
				} else {
					broken := forceBreak(word, width)
					for j, part := range broken {
						if j < len(broken)-1 {
							result = append(result, part)
						} else {
							current.WriteString(part)
							currentWidth = stringWidth(part)
						}
					}
				}
			}
			i += utf8.RuneCountInString(word)
		}
	}

	// Don't forget the last line
	if current.Len() > 0 || len(result) == 0 {
		result = append(result, current.String())
	}

	return result
}

// collectWord collects a contiguous run of non-space, non-CJK runes starting at index i.
func collectWord(runes []rune, i int) string {
	var b strings.Builder
	for i < len(runes) && runes[i] != ' ' && !isCJK(runes[i]) {
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

// forceBreak breaks a string into chunks that each fit within width columns.
func forceBreak(s string, width int) []string {
	var parts []string
	var current strings.Builder
	currentWidth := 0

	for _, r := range s {
		rw := runeWidth(r)
		if currentWidth+rw > width {
			parts = append(parts, current.String())
			current.Reset()
			currentWidth = 0
		}
		current.WriteRune(r)
		currentWidth += rw
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
