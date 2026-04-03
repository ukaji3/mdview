package renderer

import (
	"strings"
	"testing"

	"github.com/ukaji3/mdview/internal/parser"
	"github.com/ukaji3/mdview/internal/terminal"
)

func htmlCtx() *RenderContext {
	return &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         true,
	}
}

// --- Block-level HTML tests ---

func TestHTMLBlock_ParagraphContentExtraction(t *testing.T) {
	source := []byte("<p>Hello from HTML paragraph</p>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "Hello from HTML paragraph") {
		t.Errorf("expected paragraph content, got %q", result)
	}
	// Should NOT contain raw <p> tags
	if strings.Contains(result, "<p>") || strings.Contains(result, "</p>") {
		t.Errorf("expected <p> tags to be stripped, got %q", result)
	}
}

func TestHTMLBlock_DivContentExtraction(t *testing.T) {
	source := []byte("<div align=\"center\">Centered content</div>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "Centered content") {
		t.Errorf("expected div content, got %q", result)
	}
	if strings.Contains(result, "<div") || strings.Contains(result, "</div>") {
		t.Errorf("expected <div> tags to be stripped, got %q", result)
	}
}

func TestHTMLBlock_BrLineBreak(t *testing.T) {
	tests := []string{
		"<p>Line one<br>Line two</p>\n",
		"<p>Line one<br/>Line two</p>\n",
		"<p>Line one<br />Line two</p>\n",
	}
	for _, src := range tests {
		source := []byte(src)
		node := parser.Parse(source)
		result := Render(node, source, htmlCtx())
		if !strings.Contains(result, "Line one") || !strings.Contains(result, "Line two") {
			t.Errorf("expected both lines in output for %q, got %q", src, result)
		}
		if strings.Contains(result, "<br") {
			t.Errorf("expected <br> to be converted, got %q", result)
		}
	}
}

func TestHTMLBlock_HrRendering(t *testing.T) {
	source := []byte("<hr>\n")
	node := parser.Parse(source)
	ctx := htmlCtx()
	ctx.TermWidth = 40
	result := Render(node, source, ctx)
	if !strings.Contains(result, "─") {
		t.Errorf("expected horizontal rule characters, got %q", result)
	}
}

func TestHTMLBlock_ImgFallback(t *testing.T) {
	source := []byte("<img src=\"photo.png\" alt=\"My Photo\">\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "[画像: My Photo]") {
		t.Errorf("expected image fallback text, got %q", result)
	}
}

func TestHTMLBlock_ImgNoAlt(t *testing.T) {
	source := []byte("<img src=\"photo.png\">\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	// Should fallback to src as alt text
	if !strings.Contains(result, "photo.png") {
		t.Errorf("expected src as fallback alt text, got %q", result)
	}
}

func TestHTMLBlock_LinkRendering(t *testing.T) {
	source := []byte("<a href=\"https://example.com\">Example</a>\n")
	node := parser.Parse(source)
	ctx := htmlCtx()
	result := Render(node, source, ctx)
	if !strings.Contains(result, "Example") {
		t.Errorf("expected link text, got %q", result)
	}
	// Should have link styling (underline)
	if !strings.Contains(result, Underline) {
		t.Errorf("expected Underline ANSI code for link, got %q", result)
	}
}

func TestHTMLBlock_EmphasisItalic(t *testing.T) {
	source := []byte("<em>italic text</em>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "italic text") {
		t.Errorf("expected italic text, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code, got %q", result)
	}
	if !strings.Contains(result, ItalicOff) {
		t.Errorf("expected ItalicOff ANSI code, got %q", result)
	}
}

func TestHTMLBlock_ItalicTag(t *testing.T) {
	source := []byte("<i>italic text</i>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "italic text") {
		t.Errorf("expected italic text, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code, got %q", result)
	}
}

func TestHTMLBlock_StrongBold(t *testing.T) {
	source := []byte("<strong>bold text</strong>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "bold text") {
		t.Errorf("expected bold text, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code, got %q", result)
	}
}

func TestHTMLBlock_BTag(t *testing.T) {
	source := []byte("<b>bold text</b>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "bold text") {
		t.Errorf("expected bold text, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code, got %q", result)
	}
}

func TestHTMLBlock_CodeInline(t *testing.T) {
	source := []byte("<code>fmt.Println</code>\n")
	node := parser.Parse(source)
	ctx := htmlCtx()
	result := Render(node, source, ctx)
	if !strings.Contains(result, "fmt.Println") {
		t.Errorf("expected code text, got %q", result)
	}
	if !strings.Contains(result, ctx.Theme.CodeBg) {
		t.Errorf("expected CodeBg ANSI code, got %q", result)
	}
}

func TestHTMLBlock_KbdRendering(t *testing.T) {
	source := []byte("<kbd>Ctrl</kbd>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "[Ctrl]") {
		t.Errorf("expected [Ctrl] bracket rendering, got %q", result)
	}
}

func TestHTMLBlock_StrikethroughDel(t *testing.T) {
	source := []byte("<del>deleted</del>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "deleted") {
		t.Errorf("expected deleted text, got %q", result)
	}
	if !strings.Contains(result, Strikethrough) {
		t.Errorf("expected Strikethrough ANSI code, got %q", result)
	}
}

func TestHTMLBlock_StrikethroughS(t *testing.T) {
	source := []byte("<s>struck</s>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "struck") {
		t.Errorf("expected struck text, got %q", result)
	}
	if !strings.Contains(result, Strikethrough) {
		t.Errorf("expected Strikethrough ANSI code, got %q", result)
	}
}

func TestHTMLBlock_UnderlineTag(t *testing.T) {
	source := []byte("<u>underlined</u>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "underlined") {
		t.Errorf("expected underlined text, got %q", result)
	}
	if !strings.Contains(result, Underline) {
		t.Errorf("expected Underline ANSI code, got %q", result)
	}
}

func TestHTMLBlock_SubSupRenderAsText(t *testing.T) {
	source := []byte("<p>H<sub>2</sub>O and x<sup>2</sup></p>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "H") || !strings.Contains(result, "2") || !strings.Contains(result, "O") {
		t.Errorf("expected sub/sup content as plain text, got %q", result)
	}
}

func TestHTMLBlock_DetailsSummary(t *testing.T) {
	source := []byte("<details>\n<summary>Click to expand</summary>\nHidden content here\n</details>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "▶ Click to expand") {
		t.Errorf("expected summary with ▶ prefix, got %q", result)
	}
	if !strings.Contains(result, "Hidden content here") {
		t.Errorf("expected details content, got %q", result)
	}
}

func TestHTMLBlock_UnknownTagsStripped(t *testing.T) {
	source := []byte("<span>some text</span>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "some text") {
		t.Errorf("expected inner text preserved, got %q", result)
	}
	if strings.Contains(result, "<span>") || strings.Contains(result, "</span>") {
		t.Errorf("expected unknown tags to be stripped, got %q", result)
	}
}

// --- Inline HTML tests ---

func TestRawHTML_BrInline(t *testing.T) {
	// Inline <br> within a paragraph
	source := []byte("Hello<br>World")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("expected both parts around <br>, got %q", result)
	}
}

func TestRawHTML_EmInline(t *testing.T) {
	source := []byte("This is <em>important</em> text")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "important") {
		t.Errorf("expected emphasized text, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code for inline <em>, got %q", result)
	}
}

func TestRawHTML_StrongInline(t *testing.T) {
	source := []byte("This is <strong>bold</strong> text")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "bold") {
		t.Errorf("expected bold text, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code for inline <strong>, got %q", result)
	}
}

func TestRawHTML_KbdInline(t *testing.T) {
	source := []byte("Press <kbd>Enter</kbd> to continue")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "[Enter]") {
		t.Errorf("expected [Enter] bracket rendering, got %q", result)
	}
}

func TestRawHTML_CodeInline(t *testing.T) {
	source := []byte("Use <code>go test</code> to run")
	node := parser.Parse(source)
	ctx := htmlCtx()
	result := Render(node, source, ctx)
	if !strings.Contains(result, "go test") {
		t.Errorf("expected code text, got %q", result)
	}
	if !strings.Contains(result, ctx.Theme.CodeBg) {
		t.Errorf("expected CodeBg for inline <code>, got %q", result)
	}
}

// --- Mixed and edge case tests ---

func TestHTMLBlock_MixedHTMLAndText(t *testing.T) {
	source := []byte("<p>Normal <strong>bold</strong> and <em>italic</em> text</p>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "Normal") {
		t.Errorf("expected 'Normal' text, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code, got %q", result)
	}
}

func TestHTMLBlock_NestedTags(t *testing.T) {
	source := []byte("<p><strong><em>bold italic</em></strong></p>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if !strings.Contains(result, "bold italic") {
		t.Errorf("expected nested tag content, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold for nested tags, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic for nested tags, got %q", result)
	}
}

func TestHTMLBlock_SelfClosingTags(t *testing.T) {
	source := []byte("<br/>\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	// Should just produce a newline, no raw tag
	if strings.Contains(result, "<br") {
		t.Errorf("expected <br/> to be converted, got %q", result)
	}
}

func TestHTMLBlock_HTMLComment(t *testing.T) {
	source := []byte("<!-- This is a comment -->\n")
	node := parser.Parse(source)
	result := Render(node, source, htmlCtx())
	if strings.Contains(result, "This is a comment") {
		t.Errorf("expected HTML comment to be stripped, got %q", result)
	}
}

// --- Helper function tests ---

func TestExtractAttr(t *testing.T) {
	tests := []struct {
		attrs    string
		name     string
		expected string
	}{
		{` src="image.png" alt="photo"`, "src", "image.png"},
		{` src="image.png" alt="photo"`, "alt", "photo"},
		{` href='https://example.com'`, "href", "https://example.com"},
		{` class="main"`, "id", ""},
	}
	for _, tt := range tests {
		got := extractAttr(tt.attrs, tt.name)
		if got != tt.expected {
			t.Errorf("extractAttr(%q, %q) = %q, want %q", tt.attrs, tt.name, got, tt.expected)
		}
	}
}

func TestExtractAttrsMap(t *testing.T) {
	tag := `<img src="photo.png" alt="My Photo" width="200">`
	attrs := extractAttrsMap(tag)
	if attrs["src"] != "photo.png" {
		t.Errorf("expected src=photo.png, got %q", attrs["src"])
	}
	if attrs["alt"] != "My Photo" {
		t.Errorf("expected alt=My Photo, got %q", attrs["alt"])
	}
	if attrs["width"] != "200" {
		t.Errorf("expected width=200, got %q", attrs["width"])
	}
}

func TestConvertHTML_EmptyInput(t *testing.T) {
	result := convertHTML("", htmlCtx(), false)
	if result != "" {
		t.Errorf("expected empty output for empty input, got %q", result)
	}
}

func TestConvertHTML_PlainText(t *testing.T) {
	result := convertHTML("Just plain text", htmlCtx(), false)
	if result != "Just plain text" {
		t.Errorf("expected plain text passthrough, got %q", result)
	}
}
