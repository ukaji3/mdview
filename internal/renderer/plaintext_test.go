package renderer

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/ukaji3/mdview/internal/parser"
	"github.com/ukaji3/mdview/internal/terminal"
	"pgregory.net/rapid"
)

// --- Unit tests for StripANSI ---

func TestStripANSI_RemovesCSISequences(t *testing.T) {
	input := "\033[1mBold\033[0m"
	result := StripANSI(input)
	if result != "Bold" {
		t.Errorf("expected 'Bold', got %q", result)
	}
}

func TestStripANSI_RemovesSixelDCS(t *testing.T) {
	input := "before\033Pq#0;2;0;0;0\033\\after"
	result := StripANSI(input)
	if result != "beforeafter" {
		t.Errorf("expected 'beforeafter', got %q", result)
	}
}

func TestStripANSI_PreservesPlainText(t *testing.T) {
	input := "Hello, world!"
	result := StripANSI(input)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestStripANSI_RemovesColorCodes(t *testing.T) {
	input := "\033[38;2;0;255;255mCyan\033[0m"
	result := StripANSI(input)
	if result != "Cyan" {
		t.Errorf("expected 'Cyan', got %q", result)
	}
}

func TestStripANSI_EmptyString(t *testing.T) {
	result := StripANSI("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// --- Property 16: NO_COLOR/パイプ出力時のプレーンテキスト化 ---
// Feature: markdown-terminal-renderer, Property 16: NO_COLOR/パイプ出力時のプレーンテキスト化
//
// **Validates: Requirements 11.3, 11.4, 13.11, 13.12, 14.11, 14.12**
//
// For any Markdown text, when NO_COLOR is set or output is piped,
// the rendered result contains no ANSI escape sequences, no Sixel
// escape sequences, and images are displayed as "[画像: altテキスト]".

// ansiCSIPattern matches any ANSI CSI escape sequence.
var ansiCSIPattern = regexp.MustCompile(`\033\[`)

// sixelDCSPattern matches the start of a Sixel DCS sequence.
var sixelDCSPattern = regexp.MustCompile(`\033P`)

// escapePattern matches any ESC character.
var escapePattern = regexp.MustCompile(`\033`)

func plainTextCtx() *RenderContext {
	return &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorNone,
		ImageProtocol: terminal.ImageNone,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         false,
	}
}

// genMarkdownText generates random Markdown text containing various elements.
func genMarkdownText(t *rapid.T) string {
	var parts []string

	// Generate 1-5 random Markdown elements
	numElements := rapid.IntRange(1, 5).Draw(t, "numElements")
	for i := 0; i < numElements; i++ {
		elementType := rapid.IntRange(0, 7).Draw(t, fmt.Sprintf("elementType_%d", i))
		word := rapid.StringMatching(`[A-Za-z]{3,15}`).Draw(t, fmt.Sprintf("word_%d", i))

		switch elementType {
		case 0: // Heading
			level := rapid.IntRange(1, 6).Draw(t, fmt.Sprintf("headingLevel_%d", i))
			prefix := strings.Repeat("#", level)
			parts = append(parts, prefix+" "+word)
		case 1: // Plain paragraph
			parts = append(parts, word)
		case 2: // Bold text
			parts = append(parts, "**"+word+"**")
		case 3: // Italic text
			parts = append(parts, "*"+word+"*")
		case 4: // Code block
			lang := rapid.SampledFrom([]string{"go", "python", "js", ""}).Draw(t, fmt.Sprintf("lang_%d", i))
			parts = append(parts, "```"+lang+"\n"+word+"\n```")
		case 5: // Image
			alt := rapid.StringMatching(`[A-Za-z]{3,10}`).Draw(t, fmt.Sprintf("alt_%d", i))
			parts = append(parts, "!["+alt+"](image.png)")
		case 6: // Link
			url := "https://example.com/" + word
			parts = append(parts, "["+word+"]("+url+")")
		case 7: // Blockquote
			parts = append(parts, "> "+word)
		}
	}

	return strings.Join(parts, "\n\n")
}

func TestProperty16_NoColorPlainText(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		md := genMarkdownText(t)
		source := []byte(md)
		node := parser.Parse(source)
		ctx := plainTextCtx()

		result := Render(node, source, ctx)

		// Property: No ANSI CSI escape sequences
		if ansiCSIPattern.MatchString(result) {
			t.Fatalf("ANSI CSI escape sequence found in plain text output.\nMarkdown: %q\nResult: %q", md, result)
		}

		// Property: No Sixel DCS escape sequences
		if sixelDCSPattern.MatchString(result) {
			t.Fatalf("Sixel DCS escape sequence found in plain text output.\nMarkdown: %q\nResult: %q", md, result)
		}

		// Property: No ESC character at all
		if escapePattern.MatchString(result) {
			t.Fatalf("ESC character (\\033) found in plain text output.\nMarkdown: %q\nResult: %q", md, result)
		}

		// Property: Images are displayed as "[画像: altテキスト]" format
		if strings.Contains(md, "![") {
			// Extract alt text from the markdown
			imgRe := regexp.MustCompile(`!\[([^\]]+)\]`)
			matches := imgRe.FindAllStringSubmatch(md, -1)
			for _, match := range matches {
				altText := match[1]
				expected := "[画像: " + altText + "]"
				if !strings.Contains(result, expected) {
					t.Fatalf("Expected image fallback %q in result.\nMarkdown: %q\nResult: %q", expected, md, result)
				}
			}
		}
	})
}
