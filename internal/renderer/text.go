package renderer

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// renderEmphasis renders bold (level 2) or italic (level 1) text with ANSI codes.
// Multiple decorations stack when nested (e.g., bold+italic).
func renderEmphasis(buf *strings.Builder, n *ast.Emphasis, entering bool) (ast.WalkStatus, error) {
	if entering {
		if n.Level == 2 {
			buf.WriteString(Bold)
		} else {
			buf.WriteString(Italic)
		}
	} else {
		buf.WriteString(Reset)
	}
	return ast.WalkContinue, nil
}

// renderCodeSpan renders inline code with background color from the theme.
// It extracts child text, wraps it with the CodeBg ANSI code, and resets.
func renderCodeSpan(buf *strings.Builder, n *ast.CodeSpan, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		// Apply background color from theme
		if ctx != nil && ctx.Theme != nil && ctx.Theme.CodeBg != "" {
			buf.WriteString(ctx.Theme.CodeBg)
		}
		// Extract text from child nodes
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value(source))
			}
		}
		buf.WriteString(Reset)
	}
	return ast.WalkSkipChildren, nil
}

// renderStrikethroughText renders strikethrough text with ANSI strikethrough code.
func renderStrikethroughText(buf *strings.Builder, n *east.Strikethrough, entering bool) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString(Strikethrough)
	} else {
		buf.WriteString(Reset)
	}
	return ast.WalkContinue, nil
}
