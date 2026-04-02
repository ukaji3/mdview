package renderer

import (
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 11: 水平線のターミナル幅一致
// **Validates: Requirements 9.1**
func TestProperty11_HorizontalRuleTerminalWidth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random terminal width (40-200)
		termWidth := rapid.IntRange(40, 200).Draw(t, "termWidth")

		// 2. Construct markdown with horizontal rule
		markdown := "---"

		// 3. Parse and render
		source := []byte(markdown)
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    termWidth,
			ColorMode:    terminal.ColorTrue,
			ImageProtocol: terminal.ImageNone,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 4. Count the number of "─" characters in the output (ignoring ANSI codes)
		stripped := stripANSI(result)
		dashCount := strings.Count(stripped, "─")

		if dashCount != termWidth {
			t.Fatalf("expected %d horizontal rule chars (─) for terminal width %d, got %d in output %q",
				termWidth, termWidth, dashCount, result)
		}

		// 5. Verify HRColor is present
		if !strings.Contains(result, theme.HRColor) {
			t.Fatalf("expected HRColor %q in output, got %q", theme.HRColor, result)
		}

		// 6. Verify Reset is present
		if !strings.Contains(result, Reset) {
			t.Fatalf("expected Reset ANSI code in output, got %q", result)
		}
	})
}

// Feature: markdown-terminal-renderer, Property 12: リンクのレンダリング形式
// **Validates: Requirements 9.2**
func TestProperty12_LinkRenderingFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random link text and URL
		linkText := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{1,20}`).Draw(t, "linkText")
		urlPath := rapid.StringMatching(`[a-z]{2,10}`).Draw(t, "urlPath")
		url := "https://example.com/" + urlPath

		// 2. Construct markdown with link
		markdown := "[" + linkText + "](" + url + ")"

		// 3. Parse and render
		source := []byte(markdown)
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    80,
			ColorMode:    terminal.ColorTrue,
			ImageProtocol: terminal.ImageNone,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 4a. Verify link text is present
		if !strings.Contains(result, linkText) {
			t.Fatalf("expected link text %q in output, got %q", linkText, result)
		}

		// 4b. Verify URL is present in parentheses
		expectedURLPart := "(" + url + ")"
		if !strings.Contains(result, expectedURLPart) {
			t.Fatalf("expected URL in parentheses %q in output, got %q", expectedURLPart, result)
		}

		// 4c. Verify Underline ANSI code is present
		if !strings.Contains(result, Underline) {
			t.Fatalf("expected Underline ANSI code %q in output, got %q", Underline, result)
		}

		// 4d. Verify LinkColor is present
		if !strings.Contains(result, theme.LinkColor) {
			t.Fatalf("expected LinkColor %q in output, got %q", theme.LinkColor, result)
		}

		// 4e. Verify Reset is present
		if !strings.Contains(result, Reset) {
			t.Fatalf("expected Reset ANSI code in output, got %q", result)
		}
	})
}

// Unit test: basic horizontal rule
func TestHorizontalRule_Basic(t *testing.T) {
	source := []byte("---")
	node := parser.Parse(source)
	theme := terminal.DefaultTheme()
	ctx := &RenderContext{
		TermWidth:    80,
		ColorMode:    terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:        theme,
		IsTTY:        true,
	}
	result := Render(node, source, ctx)

	stripped := stripANSI(result)
	dashCount := strings.Count(stripped, "─")
	if dashCount != 80 {
		t.Errorf("expected 80 horizontal rule chars, got %d in %q", dashCount, result)
	}
}

// Unit test: basic link rendering
func TestLink_Basic(t *testing.T) {
	source := []byte("[Click here](https://example.com)")
	node := parser.Parse(source)
	theme := terminal.DefaultTheme()
	ctx := &RenderContext{
		TermWidth:    80,
		ColorMode:    terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:        theme,
		IsTTY:        true,
	}
	result := Render(node, source, ctx)

	if !strings.Contains(result, "Click here") {
		t.Errorf("expected link text in output, got %q", result)
	}
	if !strings.Contains(result, "(https://example.com)") {
		t.Errorf("expected URL in parentheses in output, got %q", result)
	}
	if !strings.Contains(result, Underline) {
		t.Errorf("expected Underline ANSI code in output, got %q", result)
	}
}
