package renderer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 4: コードブロックのボックス描画と行番号
// Validates: Requirements 5.1, 5.3, 5.4
func TestProperty4_CodeBlockBoxDrawingAndLineNumbers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random number of code lines (1-20)
		numLines := rapid.IntRange(1, 20).Draw(t, "numLines")

		// 2. Generate random ASCII code lines
		var codeLines []string
		for i := 0; i < numLines; i++ {
			line := rapid.StringMatching(`[A-Za-z0-9 _=\+\-\(\);]{1,60}`).Draw(t, fmt.Sprintf("line%d", i))
			codeLines = append(codeLines, line)
		}
		codeContent := strings.Join(codeLines, "\n")

		// 3. Construct fenced code block markdown
		markdown := "```\n" + codeContent + "\n```"

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

		// 5a. Verify box drawing characters are present
		boxChars := []string{"┌", "┐", "└", "┘", "│", "─"}
		for _, ch := range boxChars {
			if !strings.Contains(result, ch) {
				t.Fatalf("expected box drawing char %q in output, got %q", ch, result)
			}
		}

		// 5b. Verify CodeBg ANSI code is present
		if !strings.Contains(result, theme.CodeBg) {
			t.Fatalf("expected CodeBg ANSI code %q in output, got %q", theme.CodeBg, result)
		}

		// 5c. Verify sequential line numbers (1, 2, 3, ...)
		for i := 1; i <= numLines; i++ {
			numStr := fmt.Sprintf("%d", i)
			if !strings.Contains(result, numStr) {
				t.Fatalf("expected line number %q in output for %d lines, got %q", numStr, numLines, result)
			}
		}

		// 5d. Verify line numbers appear in sequential order in the output
		lastIdx := -1
		for i := 1; i <= numLines; i++ {
			// Line numbers are formatted with padding, search for "│ N │" pattern
			// where N is the line number (possibly padded)
			numStr := fmt.Sprintf("%d", i)
			idx := strings.Index(result[lastIdx+1:], numStr)
			if idx == -1 {
				t.Fatalf("line number %d not found after position %d in output", i, lastIdx+1)
			}
			lastIdx = lastIdx + 1 + idx
		}
	})
}

// Feature: markdown-terminal-renderer, Property 5: コードブロックの言語ラベル表示
// Validates: Requirements 5.2
func TestProperty5_CodeBlockLanguageLabel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random language name (lowercase letters, 1-10 chars)
		lang := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "lang")

		// 2. Generate random code content (1-5 lines)
		numLines := rapid.IntRange(1, 5).Draw(t, "numLines")
		var codeLines []string
		for i := 0; i < numLines; i++ {
			line := rapid.StringMatching(`[A-Za-z0-9 _=\(\);]{1,40}`).Draw(t, fmt.Sprintf("line%d", i))
			codeLines = append(codeLines, line)
		}
		codeContent := strings.Join(codeLines, "\n")

		// 3. Construct fenced code block with language
		markdown := "```" + lang + "\n" + codeContent + "\n```"

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

		// 5. Verify language name appears in the output (in the top border area)
		// The top border format is: ┌─ lang ─...─┐
		// So the language name should appear between the top-left corner and top-right corner
		topBorderEnd := strings.Index(result, "\n")
		if topBorderEnd == -1 {
			t.Fatalf("expected newline in output, got %q", result)
		}
		topBorder := result[:topBorderEnd]

		if !strings.Contains(topBorder, lang) {
			t.Fatalf("expected language label %q in top border %q, full output: %q", lang, topBorder, result)
		}

		// Verify the top border contains box drawing characters around the label
		if !strings.Contains(topBorder, "┌") {
			t.Fatalf("expected ┌ in top border %q", topBorder)
		}
		if !strings.Contains(topBorder, "┐") {
			t.Fatalf("expected ┐ in top border %q", topBorder)
		}
	})
}

// Feature: markdown-terminal-renderer, Property 15: コードブロック行の切り詰め
// Validates: Requirements 10.4
func TestProperty15_CodeBlockLineTruncation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate a terminal width (40-120)
		termWidth := rapid.IntRange(40, 120).Draw(t, "termWidth")

		// 2. Generate a code line that is longer than terminal width
		// Make it at least termWidth+20 chars to guarantee truncation
		lineLen := rapid.IntRange(termWidth+20, termWidth+200).Draw(t, "lineLen")
		longLine := strings.Repeat("x", lineLen)

		// 3. Construct fenced code block
		markdown := "```\n" + longLine + "\n```"

		// 4. Parse and render
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

		// 5a. Verify ellipsis "…" appears (truncation occurred)
		if !strings.Contains(result, "…") {
			t.Fatalf("expected ellipsis '…' for truncated line (termWidth=%d, lineLen=%d), got %q",
				termWidth, lineLen, result)
		}

		// 5b. Verify each output line's visible width does not exceed terminal width
		outputLines := strings.Split(result, "\n")
		for i, line := range outputLines {
			if line == "" {
				continue
			}
			w := displayWidth(line)
			if w > termWidth {
				t.Fatalf("line %d visible width %d exceeds terminal width %d: %q",
					i, w, termWidth, line)
			}
		}
	})
}
