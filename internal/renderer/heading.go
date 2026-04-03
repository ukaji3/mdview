package renderer

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// headingPrefix returns the level-based prefix for a heading (e.g., "# " for h1, "## " for h2).
func headingPrefix(level int) string {
	return strings.Repeat("#", level) + " "
}

// headingColor returns the ANSI color code for the given heading level from the theme.
func headingColor(level int, ctx *RenderContext) string {
	if ctx.Theme == nil {
		return ""
	}
	switch level {
	case 1:
		return ctx.Theme.H1Color
	case 2:
		return ctx.Theme.H2Color
	case 3:
		return ctx.Theme.H3Color
	case 4:
		return ctx.Theme.H4Color
	case 5:
		return ctx.Theme.H5Color
	case 6:
		return ctx.Theme.H6Color
	default:
		return ctx.Theme.H6Color
	}
}

// renderHeading renders a heading node with level-based color, bold, underline (h1), and prefix.
func renderHeading(buf *strings.Builder, n *ast.Heading, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		// Blank line above
		buf.WriteByte('\n')

		// Apply color from theme
		color := headingColor(n.Level, ctx)
		if color != "" {
			buf.WriteString(color)
		}

		// Apply Bold
		buf.WriteString(Bold)

		// For h1, also apply Underline
		if n.Level == 1 {
			buf.WriteString(Underline)
		}

		// Level-based prefix
		buf.WriteString(headingPrefix(n.Level))
	} else {
		// Reset all attributes
		buf.WriteString(Reset)

		// Blank line below
		buf.WriteString("\n\n")
	}
	return ast.WalkContinue, nil
}
