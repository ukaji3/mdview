package renderer

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// renderBlockquote renders a blockquote with colored vertical bars and italic text.
// The children are rendered into a sub-buffer first, then EVERY line is
// prefixed with a colored "│ " bar. Nested blockquotes compose naturally:
// the inner quote prefixes its own lines, and the outer quote prefixes them
// again, producing "│ │ ...".
func renderBlockquote(buf *strings.Builder, n *ast.Blockquote, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	// Render children into a sub-buffer with a width reduced by the prefix.
	subCtx := ctx
	if ctx != nil && ctx.TermWidth > 2 {
		c := *ctx
		c.TermWidth = ctx.TermWidth - 2
		subCtx = &c
	}
	var inner strings.Builder
	renderChildrenInto(&inner, n, source, subCtx)

	// Trim trailing newlines, then prefix each line.
	content := strings.TrimRight(inner.String(), "\n")

	barPrefix := "│ "
	barColor := ""
	if ctx != nil && ctx.Theme != nil && ctx.Theme.BlockquoteBar != "" {
		barColor = ctx.Theme.BlockquoteBar
	}

	for _, line := range strings.Split(content, "\n") {
		if barColor != "" {
			buf.WriteString(barColor)
		}
		buf.WriteString(barPrefix)
		buf.WriteString(Reset)
		buf.WriteString(Italic)
		buf.WriteString(line)
		buf.WriteString(ItalicOff)
		buf.WriteByte('\n')
	}
	buf.WriteByte('\n')

	return ast.WalkSkipChildren, nil
}
