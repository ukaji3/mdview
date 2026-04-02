package renderer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 3: テキスト装飾のANSI属性適用
// Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5
func TestProperty3_TextDecorationANSIAttributes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random text content (ASCII letters, non-empty)
		text := rapid.StringMatching(`[A-Za-z]{1,30}`).Draw(t, "text")

		// Generate random combination of bold/italic/strikethrough (at least one)
		bold := rapid.Bool().Draw(t, "bold")
		italic := rapid.Bool().Draw(t, "italic")
		strikethrough := rapid.Bool().Draw(t, "strikethrough")

		// Ensure at least one decoration is selected
		if !bold && !italic && !strikethrough {
			bold = true
		}

		// Construct Markdown with the selected decorations applied.
		// Nesting order: strikethrough wraps bold/italic wraps text.
		// Bold+Italic = ***text***, Bold = **text**, Italic = *text*
		decorated := text
		if bold && italic {
			decorated = "***" + decorated + "***"
		} else if bold {
			decorated = "**" + decorated + "**"
		} else if italic {
			decorated = "*" + decorated + "*"
		}
		if strikethrough {
			decorated = "~~" + decorated + "~~"
		}

		markdown := fmt.Sprintf("Some %s here.", decorated)

		// Parse and render
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

		// Verify all expected ANSI codes are present
		if bold {
			if !strings.Contains(result, "\033[1m") {
				t.Fatalf("expected Bold ANSI code \\033[1m for bold text, markdown=%q, got %q", markdown, result)
			}
		}
		if italic {
			if !strings.Contains(result, "\033[3m") {
				t.Fatalf("expected Italic ANSI code \\033[3m for italic text, markdown=%q, got %q", markdown, result)
			}
		}
		if strikethrough {
			if !strings.Contains(result, "\033[9m") {
				t.Fatalf("expected Strikethrough ANSI code \\033[9m for strikethrough text, markdown=%q, got %q", markdown, result)
			}
		}

		// Verify text content is present
		if !strings.Contains(result, text) {
			t.Fatalf("expected text %q in output, markdown=%q, got %q", text, markdown, result)
		}
	})
}

// Feature: markdown-terminal-renderer, Property 3: テキスト装飾のANSI属性適用 (インラインコード)
// Validates: Requirements 4.4
func TestProperty3_InlineCodeANSIAttributes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random text content (ASCII letters, non-empty)
		text := rapid.StringMatching(`[A-Za-z]{1,30}`).Draw(t, "text")

		// Inline code: `text`
		markdown := fmt.Sprintf("Use %s here.", "`"+text+"`")

		// Parse and render
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

		// Verify CodeBg ANSI code is present
		expectedCodeBg := "\033[48;5;236m"
		if !strings.Contains(result, expectedCodeBg) {
			t.Fatalf("expected CodeBg ANSI code %q for inline code, markdown=%q, got %q", expectedCodeBg, markdown, result)
		}

		// Verify text content is present
		if !strings.Contains(result, text) {
			t.Fatalf("expected text %q in output, markdown=%q, got %q", text, markdown, result)
		}
	})
}
