package renderer

import (
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 2: 見出しレベルに応じたANSI装飾
// Validates: Requirements 3.1, 3.2, 3.3, 3.4
func TestProperty2_HeadingLevelANSIDecoration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random heading level (1-6)
		level := rapid.IntRange(1, 6).Draw(t, "level")

		// 2. Generate random text content (ASCII letters, no newlines, non-empty)
		text := rapid.StringMatching(`[A-Za-z]{1,50}`).Draw(t, "text")

		// 3. Construct Markdown like "# text" with appropriate number of #
		prefix := strings.Repeat("#", level)
		markdown := prefix + " " + text

		// 4. Parse and render
		source := []byte(markdown)
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    80,
			ColorMode:    terminal.ColorTrue,
			SixelSupport: false,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 5a. Bold ANSI code is present
		if !strings.Contains(result, Bold) {
			t.Fatalf("h%d: expected Bold ANSI code in output, got %q", level, result)
		}

		// 5b. Level-specific color code from Theme is present
		var expectedColor string
		switch level {
		case 1:
			expectedColor = theme.H1Color
		case 2:
			expectedColor = theme.H2Color
		case 3:
			expectedColor = theme.H3Color
		case 4:
			expectedColor = theme.H4Color
		case 5:
			expectedColor = theme.H5Color
		case 6:
			expectedColor = theme.H6Color
		}
		if !strings.Contains(result, expectedColor) {
			t.Fatalf("h%d: expected color %q in output, got %q", level, expectedColor, result)
		}

		// 5c. For h1: Underline ANSI code is present
		if level == 1 {
			if !strings.Contains(result, Underline) {
				t.Fatalf("h1: expected Underline ANSI code in output, got %q", result)
			}
		}

		// 5d. For h2-h6: Underline is NOT present
		if level >= 2 {
			if strings.Contains(result, Underline) {
				t.Fatalf("h%d: did NOT expect Underline ANSI code in output, got %q", level, result)
			}
		}

		// 5e. Level-based prefix (strings.Repeat("#", level) + " ") is present
		expectedPrefix := strings.Repeat("#", level) + " "
		if !strings.Contains(result, expectedPrefix) {
			t.Fatalf("h%d: expected prefix %q in output, got %q", level, expectedPrefix, result)
		}

		// 5f. Text content is present in output
		if !strings.Contains(result, text) {
			t.Fatalf("h%d: expected text %q in output, got %q", level, text, result)
		}
	})
}
