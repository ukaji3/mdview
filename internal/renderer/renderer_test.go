package renderer

import (
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
)

func defaultCtx() *RenderContext {
	return &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         true,
	}
}

func TestRenderEmptyDocument(t *testing.T) {
	source := []byte("")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if result != "" {
		t.Errorf("expected empty string for empty document, got %q", result)
	}
}

func TestRenderPlainParagraph(t *testing.T) {
	source := []byte("Hello, world!")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "Hello, world!") {
		t.Errorf("expected result to contain 'Hello, world!', got %q", result)
	}
	// Paragraph should end with double newline
	if !strings.HasSuffix(result, "\n\n") {
		t.Errorf("expected paragraph to end with double newline, got %q", result)
	}
}

func TestRenderMultipleParagraphs(t *testing.T) {
	source := []byte("First paragraph.\n\nSecond paragraph.")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "First paragraph.") {
		t.Errorf("expected 'First paragraph.' in result, got %q", result)
	}
	if !strings.Contains(result, "Second paragraph.") {
		t.Errorf("expected 'Second paragraph.' in result, got %q", result)
	}
}

func TestRenderSoftLineBreak(t *testing.T) {
	// In goldmark, a single newline within a paragraph creates a soft line break
	source := []byte("Line one\nLine two")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "Line one") {
		t.Errorf("expected 'Line one' in result, got %q", result)
	}
	if !strings.Contains(result, "Line two") {
		t.Errorf("expected 'Line two' in result, got %q", result)
	}
}

func TestRenderHeadingH1(t *testing.T) {
	source := []byte("# Heading 1")
	node := parser.Parse(source)
	ctx := defaultCtx()
	result := Render(node, source, ctx)
	if !strings.Contains(result, "Heading 1") {
		t.Errorf("expected 'Heading 1' in result, got %q", result)
	}
	// h1: Bold + Underline + Cyan color
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code in h1, got %q", result)
	}
	if !strings.Contains(result, Underline) {
		t.Errorf("expected Underline ANSI code in h1, got %q", result)
	}
	if !strings.Contains(result, ctx.Theme.H1Color) {
		t.Errorf("expected H1Color in h1, got %q", result)
	}
	// h1 prefix "# "
	if !strings.Contains(result, "# Heading 1") {
		t.Errorf("expected '# Heading 1' prefix in h1, got %q", result)
	}
	// Reset after heading
	if !strings.Contains(result, Reset) {
		t.Errorf("expected Reset ANSI code after h1, got %q", result)
	}
}

func TestRenderHeadingH2(t *testing.T) {
	source := []byte("## Heading 2")
	node := parser.Parse(source)
	ctx := defaultCtx()
	result := Render(node, source, ctx)
	if !strings.Contains(result, "Heading 2") {
		t.Errorf("expected 'Heading 2' in result, got %q", result)
	}
	// h2: Bold + Green color, no underline
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code in h2, got %q", result)
	}
	if !strings.Contains(result, ctx.Theme.H2Color) {
		t.Errorf("expected H2Color in h2, got %q", result)
	}
	if strings.Contains(result, Underline) {
		t.Errorf("did not expect Underline ANSI code in h2, got %q", result)
	}
	// h2 prefix "## "
	if !strings.Contains(result, "## Heading 2") {
		t.Errorf("expected '## Heading 2' prefix in h2, got %q", result)
	}
}

func TestRenderHeadingH3ToH6(t *testing.T) {
	tests := []struct {
		level  int
		md     string
		prefix string
		color  func(*terminal.Theme) string
	}{
		{3, "### H3 Title", "### ", func(th *terminal.Theme) string { return th.H3Color }},
		{4, "#### H4 Title", "#### ", func(th *terminal.Theme) string { return th.H4Color }},
		{5, "##### H5 Title", "##### ", func(th *terminal.Theme) string { return th.H5Color }},
		{6, "###### H6 Title", "###### ", func(th *terminal.Theme) string { return th.H6Color }},
	}
	for _, tt := range tests {
		t.Run(tt.md, func(t *testing.T) {
			source := []byte(tt.md)
			node := parser.Parse(source)
			ctx := defaultCtx()
			result := Render(node, source, ctx)
			// Bold for all h3-h6
			if !strings.Contains(result, Bold) {
				t.Errorf("h%d: expected Bold ANSI code, got %q", tt.level, result)
			}
			// Level-specific color
			expectedColor := tt.color(ctx.Theme)
			if !strings.Contains(result, expectedColor) {
				t.Errorf("h%d: expected color %q, got %q", tt.level, expectedColor, result)
			}
			// No underline for h3-h6
			if strings.Contains(result, Underline) {
				t.Errorf("h%d: did not expect Underline, got %q", tt.level, result)
			}
			// Level-based prefix
			if !strings.Contains(result, tt.prefix) {
				t.Errorf("h%d: expected prefix %q, got %q", tt.level, tt.prefix, result)
			}
		})
	}
}

func TestRenderHeadingBlankLines(t *testing.T) {
	source := []byte("# Title")
	node := parser.Parse(source)
	ctx := defaultCtx()
	result := Render(node, source, ctx)
	// Should start with a newline (blank line above)
	if !strings.HasPrefix(result, "\n") {
		t.Errorf("expected heading to start with blank line, got %q", result)
	}
	// Should end with double newline (blank line below)
	if !strings.HasSuffix(result, "\n\n") {
		t.Errorf("expected heading to end with double newline, got %q", result)
	}
}

func TestRenderEmphasis(t *testing.T) {
	source := []byte("This is **bold** and *italic* text.")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "bold") {
		t.Errorf("expected 'bold' in result, got %q", result)
	}
	if !strings.Contains(result, "italic") {
		t.Errorf("expected 'italic' in result, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code, got %q", result)
	}
}

func TestRenderCodeSpan(t *testing.T) {
	source := []byte("Use `fmt.Println` to print.")
	node := parser.Parse(source)
	ctx := defaultCtx()
	result := Render(node, source, ctx)
	// Inline code should contain the text
	if !strings.Contains(result, "fmt.Println") {
		t.Errorf("expected 'fmt.Println' in result, got %q", result)
	}
	// Should apply CodeBg from theme
	if !strings.Contains(result, ctx.Theme.CodeBg) {
		t.Errorf("expected CodeBg ANSI code %q in result, got %q", ctx.Theme.CodeBg, result)
	}
	// Should reset after inline code
	if !strings.Contains(result, Reset) {
		t.Errorf("expected Reset ANSI code after inline code, got %q", result)
	}
	// Should NOT contain raw backticks wrapping the text
	if strings.Contains(result, "`fmt.Println`") {
		t.Errorf("expected no raw backticks around inline code, got %q", result)
	}
}

func TestRenderFencedCodeBlock(t *testing.T) {
	source := []byte("```go\nfmt.Println(\"hello\")\nfmt.Println(\"world\")\n```")
	node := parser.Parse(source)
	ctx := defaultCtx()
	result := Render(node, source, ctx)

	// Should contain box drawing characters
	if !strings.Contains(result, "┌") {
		t.Errorf("expected top-left box corner '┌' in result, got %q", result)
	}
	if !strings.Contains(result, "┐") {
		t.Errorf("expected top-right box corner '┐' in result, got %q", result)
	}
	if !strings.Contains(result, "└") {
		t.Errorf("expected bottom-left box corner '└' in result, got %q", result)
	}
	if !strings.Contains(result, "┘") {
		t.Errorf("expected bottom-right box corner '┘' in result, got %q", result)
	}
	if !strings.Contains(result, "│") {
		t.Errorf("expected vertical bar '│' in result, got %q", result)
	}
	if !strings.Contains(result, "─") {
		t.Errorf("expected horizontal bar '─' in result, got %q", result)
	}

	// Should contain language label on top border
	if !strings.Contains(result, "go") {
		t.Errorf("expected language label 'go' in result, got %q", result)
	}

	// Should contain code content
	if !strings.Contains(result, "fmt.Println") {
		t.Errorf("expected code content in result, got %q", result)
	}

	// Should contain line numbers
	if !strings.Contains(result, "1") {
		t.Errorf("expected line number 1 in result, got %q", result)
	}
	if !strings.Contains(result, "2") {
		t.Errorf("expected line number 2 in result, got %q", result)
	}

	// Should contain background color from theme
	if !strings.Contains(result, ctx.Theme.CodeBg) {
		t.Errorf("expected CodeBg ANSI code in result, got %q", result)
	}

	// Should contain border color from theme
	if !strings.Contains(result, ctx.Theme.CodeBorder) {
		t.Errorf("expected CodeBorder ANSI code in result, got %q", result)
	}

	// Should contain Reset
	if !strings.Contains(result, Reset) {
		t.Errorf("expected Reset ANSI code in result, got %q", result)
	}
}

func TestRenderFencedCodeBlockNoLanguage(t *testing.T) {
	source := []byte("```\nhello world\n```")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())

	// Should still have box drawing
	if !strings.Contains(result, "┌") {
		t.Errorf("expected box drawing in result, got %q", result)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("expected code content in result, got %q", result)
	}
}

func TestRenderFencedCodeBlockTruncation(t *testing.T) {
	// Create a very long line that exceeds terminal width
	longLine := strings.Repeat("x", 200)
	source := []byte("```\n" + longLine + "\n```")
	node := parser.Parse(source)
	ctx := defaultCtx()
	ctx.TermWidth = 40
	result := Render(node, source, ctx)

	// Should contain ellipsis for truncated line
	if !strings.Contains(result, "…") {
		t.Errorf("expected ellipsis '…' for truncated line, got %q", result)
	}
}

func TestRenderThematicBreakStub(t *testing.T) {
	source := []byte("Above\n\n---\n\nBelow")
	node := parser.Parse(source)
	ctx := defaultCtx()
	ctx.TermWidth = 40
	result := Render(node, source, ctx)
	hrLine := strings.Repeat("─", 40)
	if !strings.Contains(result, hrLine) {
		t.Errorf("expected horizontal rule of width 40, got %q", result)
	}
}

func TestRenderLinkStub(t *testing.T) {
	source := []byte("[Go](https://golang.org)")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "Go") {
		t.Errorf("expected link text 'Go' in result, got %q", result)
	}
	if !strings.Contains(result, "(https://golang.org)") {
		t.Errorf("expected URL in result, got %q", result)
	}
}

func TestRenderImageStub(t *testing.T) {
	source := []byte("![alt text](image.png)")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "[画像: alt text]") {
		t.Errorf("expected image fallback text, got %q", result)
	}
}

func TestRenderStrikethrough(t *testing.T) {
	source := []byte("This is ~~deleted~~ text.")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "deleted") {
		t.Errorf("expected 'deleted' in result, got %q", result)
	}
	if !strings.Contains(result, Strikethrough) {
		t.Errorf("expected Strikethrough ANSI code, got %q", result)
	}
}

func TestRenderListStub(t *testing.T) {
	source := []byte("- Item 1\n- Item 2\n- Item 3")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "Item 1") {
		t.Errorf("expected 'Item 1' in result, got %q", result)
	}
	if !strings.Contains(result, "•") {
		t.Errorf("expected bullet character in result, got %q", result)
	}
}

func TestRenderBlockquoteStub(t *testing.T) {
	source := []byte("> This is a quote")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "This is a quote") {
		t.Errorf("expected quote text in result, got %q", result)
	}
	if !strings.Contains(result, "│") {
		t.Errorf("expected blockquote bar in result, got %q", result)
	}
}

func TestRenderBoldItalicStacking(t *testing.T) {
	source := []byte("This is ***bold and italic*** text.")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, "bold and italic") {
		t.Errorf("expected 'bold and italic' in result, got %q", result)
	}
	// Both Bold and Italic ANSI codes should be present
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code for bold+italic, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code for bold+italic, got %q", result)
	}
}

func TestRenderStrikethroughWithReset(t *testing.T) {
	source := []byte("This is ~~deleted~~ text.")
	node := parser.Parse(source)
	result := Render(node, source, defaultCtx())
	if !strings.Contains(result, Strikethrough) {
		t.Errorf("expected Strikethrough ANSI code, got %q", result)
	}
	if !strings.Contains(result, Reset) {
		t.Errorf("expected Reset ANSI code after strikethrough, got %q", result)
	}
}

func TestRenderCodeSpanWithNilTheme(t *testing.T) {
	source := []byte("Use `code` here.")
	node := parser.Parse(source)
	ctx := &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:         nil,
		IsTTY:         true,
	}
	result := Render(node, source, ctx)
	if !strings.Contains(result, "code") {
		t.Errorf("expected 'code' in result, got %q", result)
	}
}

func TestRenderContextFields(t *testing.T) {
	ctx := &RenderContext{
		TermWidth:     120,
		ColorMode:     terminal.Color256,
		ImageProtocol: terminal.ImageSixel,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         false,
	}
	if ctx.TermWidth != 120 {
		t.Errorf("expected TermWidth 120, got %d", ctx.TermWidth)
	}
	if ctx.ColorMode != terminal.Color256 {
		t.Errorf("expected Color256, got %d", ctx.ColorMode)
	}
	if ctx.ImageProtocol != terminal.ImageSixel {
		t.Error("expected ImageProtocol ImageSixel")
	}
	if ctx.IsTTY {
		t.Error("expected IsTTY false")
	}
}
