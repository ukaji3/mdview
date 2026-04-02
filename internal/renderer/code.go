package renderer

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// renderFencedCodeBlock renders a fenced code block with box drawing characters,
// background color, line numbers, and optional language label.
func renderFencedCodeBlock(buf *strings.Builder, n *ast.FencedCodeBlock, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	lang := string(n.Language(source))

	// Collect code lines
	var lines []string
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		line := string(seg.Value(source))
		// Strip trailing newline from each line
		line = strings.TrimRight(line, "\n")
		lines = append(lines, line)
	}

	renderCodeBox(buf, lines, lang, ctx)

	return ast.WalkSkipChildren, nil
}

// renderCodeBlock renders an indented code block (no language label).
func renderCodeBlock(buf *strings.Builder, n *ast.CodeBlock, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	var lines []string
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		line := string(seg.Value(source))
		line = strings.TrimRight(line, "\n")
		lines = append(lines, line)
	}

	renderCodeBox(buf, lines, "", ctx)

	return ast.WalkSkipChildren, nil
}

// renderCodeBox draws a box with line numbers around the given code lines.
// The box uses Unicode box drawing characters and fills to terminal width.
//
// Layout:
//   ┌─ lang ───────────────────────────┐
//   │  1 │ code line 1                 │
//   │  2 │ code line 2                 │
//   └───────────────────────────────────┘
func renderCodeBox(buf *strings.Builder, lines []string, lang string, ctx *RenderContext) {
	width := ctx.TermWidth
	if width <= 0 {
		width = 80
	}

	borderColor := ""
	bgColor := ""
	if ctx.Theme != nil {
		borderColor = ctx.Theme.CodeBorder
		bgColor = ctx.Theme.CodeBg
	}

	numLines := len(lines)
	if numLines == 0 {
		numLines = 1
		lines = []string{""}
	}

	// Line number width: digits needed for the largest line number
	lineNumWidth := len(fmt.Sprintf("%d", numLines))
	if lineNumWidth < 1 {
		lineNumWidth = 1
	}

	// Inner content area:
	// "│" + " " + lineNum + " " + "│" + " " + code + padding + " " + "│"
	// Border chars: 1 (left │) + 1 (space) + lineNumWidth + 1 (space) + 1 (│) + 1 (space) + ... + 1 (space) + 1 (right │)
	// = lineNumWidth + 7
	overhead := lineNumWidth + 7
	codeAreaWidth := width - overhead
	if codeAreaWidth < 1 {
		codeAreaWidth = 1
	}

	// --- Top border ---
	buf.WriteString(borderColor)
	buf.WriteString("┌─")
	if lang != "" {
		buf.WriteString(" ")
		buf.WriteString(lang)
		buf.WriteString(" ")
		remaining := width - 4 - len(lang) - 2 // "┌─" + " lang " + "─...─" + "┐"
		if remaining < 0 {
			remaining = 0
		}
		buf.WriteString(strings.Repeat("─", remaining))
	} else {
		remaining := width - 3 // "┌─" + "─...─" + "┐"
		if remaining < 0 {
			remaining = 0
		}
		buf.WriteString(strings.Repeat("─", remaining))
	}
	buf.WriteString("┐")
	buf.WriteString(Reset)
	buf.WriteByte('\n')

	// --- Code lines ---
	for i, line := range lines {
		lineNum := i + 1
		numStr := fmt.Sprintf("%*d", lineNumWidth, lineNum)

		// Truncate code line if it exceeds the available code area width
		displayLine := truncateLine(line, codeAreaWidth)

		// Pad the display line to fill the code area
		padLen := codeAreaWidth - displayWidth(displayLine)
		if padLen < 0 {
			padLen = 0
		}

		buf.WriteString(borderColor)
		buf.WriteString("│")
		buf.WriteString(Reset)
		buf.WriteString(bgColor)
		buf.WriteString(" ")
		buf.WriteString(numStr)
		buf.WriteString(" ")
		buf.WriteString(borderColor)
		buf.WriteString("│")
		buf.WriteString(Reset)
		buf.WriteString(bgColor)
		buf.WriteString(" ")
		buf.WriteString(displayLine)
		buf.WriteString(strings.Repeat(" ", padLen))
		buf.WriteString(" ")
		buf.WriteString(Reset)
		buf.WriteString(borderColor)
		buf.WriteString("│")
		buf.WriteString(Reset)
		buf.WriteByte('\n')
	}

	// --- Bottom border ---
	buf.WriteString(borderColor)
	buf.WriteString("└")
	remaining := width - 2 // "└" + "─...─" + "┘"
	if remaining < 0 {
		remaining = 0
	}
	buf.WriteString(strings.Repeat("─", remaining))
	buf.WriteString("┘")
	buf.WriteString(Reset)
	buf.WriteByte('\n')
}

// displayWidth returns the visible character width of a string,
// ignoring ANSI escape sequences. CJK characters count as 2.
func displayWidth(s string) int {
	w := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		if isCJK(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// isCJK returns true if the rune is a CJK character (double-width).
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3040 && r <= 0x309F) ||
		(r >= 0x30A0 && r <= 0x30FF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFF00 && r <= 0xFFEF)
}

// truncateLine truncates a line to fit within maxWidth visible columns.
// If truncation occurs, the last visible character is replaced with "…".
func truncateLine(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	w := 0
	runes := []rune(line)
	for i, r := range runes {
		rw := 1
		if isCJK(r) {
			rw = 2
		}
		if w+rw > maxWidth {
			// Need to truncate: replace last char position with ellipsis
			// The ellipsis "…" takes 1 column
			if maxWidth >= 1 {
				// Find the cut point that leaves room for "…"
				cutW := 0
				cutIdx := 0
				for j, cr := range runes[:i] {
					crw := 1
					if isCJK(cr) {
						crw = 2
					}
					if cutW+crw > maxWidth-1 {
						break
					}
					cutW += crw
					cutIdx = j + 1
				}
				return string(runes[:cutIdx]) + "…"
			}
			return "…"
		}
		w += rw
	}
	return line
}
