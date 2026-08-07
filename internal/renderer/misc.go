package renderer

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/yuin/goldmark/ast"
)

// renderThematicBreak renders a horizontal rule using box drawing characters (─)
// spanning the full terminal width, colored with the HRColor from the theme.
func renderThematicBreak(buf *strings.Builder, n *ast.ThematicBreak, entering bool, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		width := ctx.TermWidth
		if width <= 0 {
			width = 80
		}

		if ctx.Theme != nil && ctx.Theme.HRColor != "" {
			buf.WriteString(ctx.Theme.HRColor)
		}
		dashWidth := runewidth.RuneWidth('─')
		dashCount := width / dashWidth
		buf.WriteString(strings.Repeat("─", dashCount))
		if ctx.Theme != nil && ctx.Theme.HRColor != "" {
			buf.WriteString(Reset)
		}
		buf.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

// renderLink renders a link with colored, underlined text followed by the URL in parentheses.
// Format: <LinkColor><Underline>link text<UnderlineOff><FgDefault> (<URL>)
// Individual off-codes are used instead of a full Reset so that outer styles
// (e.g. bold around the link) are preserved.
func renderLink(buf *strings.Builder, n *ast.Link, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		if ctx.Theme != nil && ctx.Theme.LinkColor != "" {
			buf.WriteString(ctx.Theme.LinkColor)
		}
		buf.WriteString(Underline)
	} else {
		buf.WriteString(UnderlineOff)
		buf.WriteString(FgDefault)
		buf.WriteString(" (")
		buf.WriteString(sanitizeText(string(n.Destination)))
		buf.WriteByte(')')
	}
	return ast.WalkContinue, nil
}
