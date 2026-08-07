// TODO: エラーメッセージに日本語と英語が混在しています。
// 既存テストへの影響を避けるため、現時点では変更しません。

package renderer

import (
	"strings"

	"github.com/ukaji3/mdview/internal/terminal"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// ANSI escape code constants for text decoration.
const (
	Reset         = "\033[0m"
	Bold          = "\033[1m"
	Italic        = "\033[3m"
	Underline     = "\033[4m"
	Strikethrough = "\033[9m"

	// Specific disable codes (avoid full Reset to preserve nested styles).
	BoldOff          = "\033[22m"
	ItalicOff        = "\033[23m"
	UnderlineOff     = "\033[24m"
	StrikethroughOff = "\033[29m"
	FgDefault        = "\033[39m" // restore default foreground color
	BgDefault        = "\033[49m" // restore default background color
)

// RenderContext holds rendering state and terminal capabilities.
type RenderContext struct {
	TermWidth     int
	CellHeight    int // terminal cell height in pixels (for image row calculation)
	CellWidth     int // terminal cell width in pixels (for image sizing)
	ColorMode     terminal.ColorMode
	ImageProtocol terminal.ImageProtocol
	Theme         *terminal.Theme
	IsTTY         bool
	MermaidTheme  string // Mermaid theme (default, dark, forest, neutral)
	NoMermaid     bool   // Disable Mermaid diagram rendering
}

// Render walks the AST and produces an ANSI-decorated string for terminal output.
func Render(node ast.Node, source []byte, ctx *RenderContext) string {
	var buf strings.Builder
	renderInto(&buf, node, source, ctx)

	result := buf.String()

	// When ColorMode is ColorNone (NO_COLOR set or pipe output),
	// strip all ANSI and Sixel escape sequences for plain text output.
	if ctx.ColorMode == terminal.ColorNone {
		result = StripANSI(result)
	}

	return result
}

// renderInto walks the AST rooted at node and writes ANSI-decorated output to
// buf. It is reused recursively by block renderers (blockquote, paragraph)
// that need to render children into a sub-buffer for post-processing.
func renderInto(buf *strings.Builder, node ast.Node, source []byte, ctx *RenderContext) {
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch v := n.(type) {
		case *ast.Document:
			// Document node: just walk children, no output
			return ast.WalkContinue, nil

		case *ast.Heading:
			return renderHeading(buf, v, entering, source, ctx)

		case *ast.Paragraph:
			return renderParagraph(buf, v, entering, source, ctx)

		case *ast.Text:
			return renderText(buf, v, entering, source)

		case *ast.String:
			if entering {
				buf.WriteString(sanitizeText(string(v.Value)))
			}
			return ast.WalkContinue, nil

		case *ast.Emphasis:
			return renderEmphasis(buf, v, entering)

		case *ast.CodeSpan:
			return renderCodeSpan(buf, v, entering, source, ctx)

		case *ast.FencedCodeBlock:
			lang := string(v.Language(source))
			if IsMermaid(lang) && !ctx.NoMermaid && entering {
				// Collect code lines
				var codeLines []string
				for i := 0; i < v.Lines().Len(); i++ {
					seg := v.Lines().At(i)
					line := string(seg.Value(source))
					line = strings.TrimRight(line, "\n")
					codeLines = append(codeLines, line)
				}
				code := strings.Join(codeLines, "\n")
				buf.WriteString(RenderMermaid(code, ctx.MermaidTheme, ctx))
				return ast.WalkSkipChildren, nil
			}
			return renderFencedCodeBlock(buf, v, entering, source, ctx)

		case *ast.CodeBlock:
			return renderCodeBlock(buf, v, entering, source, ctx)

		case *ast.List:
			return renderList(buf, v, entering)

		case *ast.ListItem:
			return renderListItem(buf, v, entering, source, ctx)

		case *ast.Blockquote:
			return renderBlockquote(buf, v, entering, source, ctx)

		case *ast.ThematicBreak:
			return renderThematicBreak(buf, v, entering, ctx)

		case *ast.Link:
			return renderLink(buf, v, entering, source, ctx)

		case *ast.Image:
			return renderImage(buf, v, entering, source, ctx)

		case *ast.AutoLink:
			return renderAutoLink(buf, v, entering, source)

		case *ast.RawHTML:
			return renderRawHTML(buf, v, entering, source, ctx)

		case *ast.HTMLBlock:
			return renderHTMLBlock(buf, v, entering, source, ctx)

		case *ast.TextBlock:
			if !entering {
				buf.WriteByte('\n')
			}
			return ast.WalkContinue, nil

		// Extension types
		case *east.Table:
			return renderTable(buf, v, entering, source, ctx)

		case *east.Strikethrough:
			return renderStrikethroughText(buf, v, entering)
		}

		return ast.WalkContinue, nil
	})
}

// renderChildrenInto renders each direct child of node into buf.
func renderChildrenInto(buf *strings.Builder, node ast.Node, source []byte, ctx *RenderContext) {
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		renderInto(buf, c, source, ctx)
	}
}

// --- Paragraph and Text (fully implemented) ---

// renderParagraph renders a paragraph. Top-level paragraphs (direct children
// of Document or Blockquote) are word-wrapped to the terminal width; other
// paragraphs (e.g. inside list items) keep the historical pass-through
// behavior so marker alignment is not disturbed.
func renderParagraph(buf *strings.Builder, n *ast.Paragraph, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	switch n.Parent().(type) {
	case *ast.Document, *ast.Blockquote:
		if !entering {
			return ast.WalkSkipChildren, nil
		}
		var inner strings.Builder
		renderChildrenInto(&inner, n, source, ctx)
		text := inner.String()
		if ctx != nil && ctx.TermWidth > 0 {
			text = wrapANSI(text, ctx.TermWidth)
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
		return ast.WalkSkipChildren, nil
	default:
		if !entering {
			buf.WriteString("\n\n")
		}
		return ast.WalkContinue, nil
	}
}

func renderText(buf *strings.Builder, n *ast.Text, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString(sanitizeText(string(n.Segment.Value(source))))
		if n.HardLineBreak() {
			buf.WriteByte('\n')
		} else if n.SoftLineBreak() {
			buf.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

func renderAutoLink(buf *strings.Builder, n *ast.AutoLink, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString(sanitizeText(string(n.URL(source))))
	}
	return ast.WalkContinue, nil
}

// sanitizeText removes control characters (C0 except \n and \t, plus DEL and
// ESC) from source-derived text so that raw escape sequences embedded in a
// Markdown document cannot inject terminal control sequences (cursor
// movement, title changes, etc.) into the output.
func sanitizeText(s string) string {
	// Fast path: no control characters.
	clean := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 && c != '\n' && c != '\t' || c == 0x7f {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 && r != '\n' && r != '\t' || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
