package renderer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"github.com/yuin/goldmark/ast"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 8: 引用ブロックのネストレベルに応じた縦線表示
// Validates: Requirements 7.1, 7.2
func TestProperty8_BlockquoteNestLevelVerticalBars(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random nest level (1-4)
		nestLevel := rapid.IntRange(1, 4).Draw(t, "nestLevel")

		// 2. Generate random text content (no trailing spaces to avoid parser trimming)
		text := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{1,30}`).Draw(t, "text")

		// 3. Construct nested blockquote markdown ("> > > text" style)
		prefix := strings.Repeat("> ", nestLevel)
		markdown := prefix + text

		// 4. Parse and render
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

		// 5. Verify: number of "│" chars >= nest level
		barCount := strings.Count(result, "│")
		if barCount < nestLevel {
			t.Fatalf("expected at least %d vertical bars (│) for nest level %d, got %d in output %q",
				nestLevel, nestLevel, barCount, result)
		}

		// 6. Verify: Italic ANSI code is present
		if !strings.Contains(result, Italic) {
			t.Fatalf("expected Italic ANSI code %q in output for blockquote, got %q",
				Italic, result)
		}

		// 7. Verify: BlockquoteBar color is present
		if !strings.Contains(result, theme.BlockquoteBar) {
			t.Fatalf("expected BlockquoteBar color %q in output, got %q",
				theme.BlockquoteBar, result)
		}

		// 8. Verify: the text content is present
		if !strings.Contains(result, text) {
			t.Fatalf("expected text %q in output, got %q", text, result)
		}

		// 9. Verify: Reset code is present (to close italic)
		if !strings.Contains(result, Reset) {
			t.Fatalf("expected Reset ANSI code in output, got %q", result)
		}
	})
}

// Unit test: basic single-level blockquote
func TestBlockquote_SingleLevel(t *testing.T) {
	source := []byte("> Hello world")
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

	if !strings.Contains(result, "│") {
		t.Errorf("expected vertical bar in output, got %q", result)
	}
	if !strings.Contains(result, Italic) {
		t.Errorf("expected Italic ANSI code in output, got %q", result)
	}
	if !strings.Contains(result, "Hello world") {
		t.Errorf("expected text in output, got %q", result)
	}
}

// Unit test: nested blockquote (2 levels)
func TestBlockquote_NestedTwoLevels(t *testing.T) {
	source := []byte("> > Nested quote")
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

	barCount := strings.Count(result, "│")
	if barCount < 2 {
		t.Errorf("expected at least 2 vertical bars for 2-level nesting, got %d in %q", barCount, result)
	}
	if !strings.Contains(result, "Nested quote") {
		t.Errorf("expected text in output, got %q", result)
	}
}

// Unit test: blockquote with inner markdown elements (requirement 7.3)
func TestBlockquote_InnerMarkdownElements(t *testing.T) {
	source := []byte("> **bold** and *italic* text")
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

	if !strings.Contains(result, "│") {
		t.Errorf("expected vertical bar in output, got %q", result)
	}
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code for **bold** inside blockquote, got %q", result)
	}
	if !strings.Contains(result, "bold") {
		t.Errorf("expected 'bold' text in output, got %q", result)
	}
	if !strings.Contains(result, "italic") {
		t.Errorf("expected 'italic' text in output, got %q", result)
	}
}

// Unit test: blockquoteNestLevel helper
func TestBlockquoteNestLevel(t *testing.T) {
	tests := []struct {
		markdown string
		expected int // expected nest level for innermost blockquote
	}{
		{"> text", 0},
		{"> > text", 1},
		{"> > > text", 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("nest_%d", tt.expected), func(t *testing.T) {
			source := []byte(tt.markdown)
			node := parser.Parse(source)

			// Find the innermost blockquote
			var innermost *ast.Blockquote
			ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if bq, ok := n.(*ast.Blockquote); ok && entering {
					innermost = bq
				}
				return ast.WalkContinue, nil
			})

			if innermost == nil {
				t.Fatal("no blockquote found")
			}

			level := blockquoteNestLevel(innermost)
			if level != tt.expected {
				t.Errorf("expected nest level %d, got %d", tt.expected, level)
			}
		})
	}
}
