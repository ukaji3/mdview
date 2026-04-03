package renderer

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
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
	// Use actual display width of box drawing characters
	borderCharWidth := runewidth.RuneWidth('│')
	overhead := borderCharWidth + 1 + lineNumWidth + 1 + borderCharWidth + 1 + 1 + borderCharWidth
	codeAreaWidth := width - overhead
	if codeAreaWidth < 1 {
		codeAreaWidth = 1
	}

	// --- Top border ---
	// Use display width for border calculations
	dashWidth := runewidth.RuneWidth('─')
	topLeftWidth := runewidth.StringWidth("┌─")
	topRightWidth := runewidth.RuneWidth('┐')
	buf.WriteString(borderColor)
	buf.WriteString("┌─")
	if lang != "" {
		buf.WriteString(" ")
		buf.WriteString(lang)
		buf.WriteString(" ")
		langDisplayWidth := runewidth.StringWidth(lang)
		remainingWidth := width - topLeftWidth - 1 - langDisplayWidth - 1 - topRightWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		dashCount := remainingWidth / dashWidth
		buf.WriteString(strings.Repeat("─", dashCount))
	} else {
		remainingWidth := width - topLeftWidth - topRightWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		dashCount := remainingWidth / dashWidth
		buf.WriteString(strings.Repeat("─", dashCount))
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
	bottomLeftWidth := runewidth.RuneWidth('└')
	bottomRightWidth := runewidth.RuneWidth('┘')
	remainingWidth := width - bottomLeftWidth - bottomRightWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	dashCount := remainingWidth / dashWidth
	buf.WriteString(strings.Repeat("─", dashCount))
	buf.WriteString("┘")
	buf.WriteString(Reset)
	buf.WriteByte('\n')
}

// displayWidth returns the visible character width of a string,
// ignoring ANSI escape sequences (CSI, OSC, DCS, APC).
// Uses go-runewidth for accurate East Asian Width handling.
func displayWidth(s string) int {
	w := 0
	i := 0
	runes := []rune(s)
	for i < len(runes) {
		r := runes[i]
		if r == '\033' && i+1 < len(runes) {
			next := runes[i+1]
			switch next {
			case '[': // CSI
				i += 2
				for i < len(runes) && !((runes[i] >= 'a' && runes[i] <= 'z') || (runes[i] >= 'A' && runes[i] <= 'Z')) {
					i++
				}
				if i < len(runes) {
					i++
				}
				continue
			case ']': // OSC - terminated by BEL or ST
				i += 2
				for i < len(runes) {
					if runes[i] == '\x07' {
						i++
						break
					}
					if runes[i] == '\033' && i+1 < len(runes) && runes[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
				continue
			case 'P', '_': // DCS or APC - terminated by ST
				i += 2
				for i < len(runes) {
					if runes[i] == '\033' && i+1 < len(runes) && runes[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
				continue
			default:
				i += 2
				continue
			}
		}
		w += runewidth.RuneWidth(r)
		i++
	}
	return w
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
		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth {
			if maxWidth >= 1 {
				cutW := 0
				cutIdx := 0
				for j, cr := range runes[:i] {
					crw := runewidth.RuneWidth(cr)
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
