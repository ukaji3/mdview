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
)

// RenderContext holds rendering state and terminal capabilities.
type RenderContext struct {
	TermWidth     int
	CellHeight    int // terminal cell height in pixels (for image row calculation)
	ColorMode     terminal.ColorMode
	ImageProtocol terminal.ImageProtocol
	Theme         *terminal.Theme
	IsTTY         bool
	MermaidTheme  string // Mermaid theme (default, dark, forest, neutral)
}

// Render walks the AST and produces an ANSI-decorated string for terminal output.
func Render(node ast.Node, source []byte, ctx *RenderContext) string {
	var buf strings.Builder

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch v := n.(type) {
		case *ast.Document:
			// Document node: just walk children, no output
			return ast.WalkContinue, nil

		case *ast.Heading:
			return renderHeading(&buf, v, entering, source, ctx)

		case *ast.Paragraph:
			return renderParagraph(&buf, v, entering)

		case *ast.Text:
			return renderText(&buf, v, entering, source)

		case *ast.String:
			if entering {
				buf.Write(v.Value)
			}
			return ast.WalkContinue, nil

		case *ast.Emphasis:
			return renderEmphasis(&buf, v, entering)

		case *ast.CodeSpan:
			return renderCodeSpan(&buf, v, entering, source, ctx)

		case *ast.FencedCodeBlock:
			lang := string(v.Language(source))
			if IsMermaid(lang) && entering {
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
			return renderFencedCodeBlock(&buf, v, entering, source, ctx)

		case *ast.CodeBlock:
			return renderCodeBlock(&buf, v, entering, source, ctx)

		case *ast.List:
			return renderList(&buf, v, entering)

		case *ast.ListItem:
			return renderListItem(&buf, v, entering, source, ctx)

		case *ast.Blockquote:
			return renderBlockquote(&buf, v, entering, source, ctx)

		case *ast.ThematicBreak:
			return renderThematicBreak(&buf, v, entering, ctx)

		case *ast.Link:
			return renderLink(&buf, v, entering, source, ctx)

		case *ast.Image:
			return renderImage(&buf, v, entering, source, ctx)

		case *ast.AutoLink:
			return renderAutoLinkStub(&buf, v, entering)

		case *ast.RawHTML:
			return renderRawHTML(&buf, v, entering, source, ctx)

		case *ast.HTMLBlock:
			return renderHTMLBlock(&buf, v, entering, source, ctx)

		case *ast.TextBlock:
			if !entering {
				buf.WriteByte('\n')
			}
			return ast.WalkContinue, nil

		// Extension types
		case *east.Table:
			return renderTable(&buf, v, entering, source, ctx)

		case *east.Strikethrough:
			return renderStrikethroughText(&buf, v, entering)
		}

		return ast.WalkContinue, nil
	})

	result := buf.String()

	// When ColorMode is ColorNone (NO_COLOR set or pipe output),
	// strip all ANSI and Sixel escape sequences for plain text output.
	if ctx.ColorMode == terminal.ColorNone {
		result = StripANSI(result)
	}

	return result
}

// --- Paragraph and Text (fully implemented) ---

func renderParagraph(buf *strings.Builder, n *ast.Paragraph, entering bool) (ast.WalkStatus, error) {
	if !entering {
		buf.WriteString("\n\n")
	}
	return ast.WalkContinue, nil
}

func renderText(buf *strings.Builder, n *ast.Text, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.Write(n.Segment.Value(source))
		if n.HardLineBreak() {
			buf.WriteByte('\n')
		} else if n.SoftLineBreak() {
			buf.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

// --- Stub renderers (to be replaced by heading.go, text.go, code.go, etc.) ---

func renderHeadingStub(buf *strings.Builder, n *ast.Heading, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString(Bold)
	} else {
		buf.WriteString(Reset)
		buf.WriteString("\n\n")
	}
	return ast.WalkContinue, nil
}

func renderEmphasisStub(buf *strings.Builder, n *ast.Emphasis, entering bool) (ast.WalkStatus, error) {
	if n.Level == 2 {
		if entering {
			buf.WriteString(Bold)
		} else {
			buf.WriteString(Reset)
		}
	} else {
		if entering {
			buf.WriteString(Italic)
		} else {
			buf.WriteString(Reset)
		}
	}
	return ast.WalkContinue, nil
}

func renderCodeSpanStub(buf *strings.Builder, n *ast.CodeSpan, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.WriteByte('`')
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value(source))
			}
		}
		buf.WriteByte('`')
	}
	return ast.WalkSkipChildren, nil
}

func renderFencedCodeBlockStub(buf *strings.Builder, n *ast.FencedCodeBlock, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		lang := string(n.Language(source))
		if lang != "" {
			buf.WriteString("[" + lang + "]\n")
		}
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			buf.Write(seg.Value(source))
		}
		buf.WriteByte('\n')
	}
	return ast.WalkSkipChildren, nil
}

func renderCodeBlockStub(buf *strings.Builder, n *ast.CodeBlock, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			buf.Write(seg.Value(source))
		}
		buf.WriteByte('\n')
	}
	return ast.WalkSkipChildren, nil
}

func renderListStub(buf *strings.Builder, n *ast.List, entering bool) (ast.WalkStatus, error) {
	if !entering {
		buf.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func renderListItemStub(buf *strings.Builder, n *ast.ListItem, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString("  • ")
	}
	return ast.WalkContinue, nil
}

func renderBlockquoteStub(buf *strings.Builder, n *ast.Blockquote, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString("│ ")
	}
	return ast.WalkContinue, nil
}

func renderThematicBreakStub(buf *strings.Builder, n *ast.ThematicBreak, entering bool, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		width := ctx.TermWidth
		if width <= 0 {
			width = 80
		}
		buf.WriteString(strings.Repeat("─", width))
		buf.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func renderLinkStub(buf *strings.Builder, n *ast.Link, entering bool, source []byte) (ast.WalkStatus, error) {
	if !entering {
		buf.WriteString(" (")
		buf.Write(n.Destination)
		buf.WriteByte(')')
	}
	return ast.WalkContinue, nil
}

func renderImageStub(buf *strings.Builder, n *ast.Image, entering bool, source []byte) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString("[画像: ")
		buf.Write(n.Text(source))
		buf.WriteString("]")
	}
	return ast.WalkSkipChildren, nil
}

func renderAutoLinkStub(buf *strings.Builder, n *ast.AutoLink, entering bool) (ast.WalkStatus, error) {
	if entering {
		buf.Write(n.URL(n.Label(nil)))
	}
	return ast.WalkContinue, nil
}

// --- Extension stubs ---

func renderTableStub(buf *strings.Builder, n *east.Table, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	// Stub: walk children, table rendering will be in table.go
	if !entering {
		buf.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}

func renderTableCellStub(buf *strings.Builder, n *east.TableCell, entering bool, source []byte) (ast.WalkStatus, error) {
	if !entering {
		buf.WriteString("\t")
	}
	return ast.WalkContinue, nil
}

func renderStrikethroughStub(buf *strings.Builder, n *east.Strikethrough, entering bool) (ast.WalkStatus, error) {
	if entering {
		buf.WriteString(Strikethrough)
	} else {
		buf.WriteString(Reset)
	}
	return ast.WalkContinue, nil
}
