package wrap

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestWrap_ShortLine(t *testing.T) {
	got := Wrap("hello world", 80)
	if got != "hello world" {
		t.Errorf("expected no wrapping, got %q", got)
	}
}

func TestWrap_WordBoundary(t *testing.T) {
	got := Wrap("hello world foo", 11)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "hello world" {
		t.Errorf("line 0: expected %q, got %q", "hello world", lines[0])
	}
	if lines[1] != "foo" {
		t.Errorf("line 1: expected %q, got %q", "foo", lines[1])
	}
}

func TestWrap_PreservesNewlines(t *testing.T) {
	got := Wrap("line1\nline2\nline3", 80)
	if got != "line1\nline2\nline3" {
		t.Errorf("expected newlines preserved, got %q", got)
	}
}

func TestWrap_CJKCharacterBoundary(t *testing.T) {
	// Each CJK char is 2 columns wide. Width=6 fits 3 CJK chars.
	got := Wrap("あいうえお", 6)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "あいう" {
		t.Errorf("line 0: expected %q, got %q", "あいう", lines[0])
	}
	if lines[1] != "えお" {
		t.Errorf("line 1: expected %q, got %q", "えお", lines[1])
	}
}

func TestWrap_MixedCJKAndLatin(t *testing.T) {
	// "Hello世界" = 5 + 2 + 2 = 9 columns
	got := Wrap("Hello世界test", 9)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "Hello世界" {
		t.Errorf("line 0: expected %q, got %q", "Hello世界", lines[0])
	}
}

func TestWrap_ForcedBreak(t *testing.T) {
	// A single word longer than width must be broken
	got := Wrap("abcdefghij", 4)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "abcd" {
		t.Errorf("line 0: expected %q, got %q", "abcd", lines[0])
	}
	if lines[1] != "efgh" {
		t.Errorf("line 1: expected %q, got %q", "efgh", lines[1])
	}
	if lines[2] != "ij" {
		t.Errorf("line 2: expected %q, got %q", "ij", lines[2])
	}
}

func TestWrap_EmptyString(t *testing.T) {
	got := Wrap("", 80)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestWrap_ZeroWidth(t *testing.T) {
	// Zero width should return text unchanged
	got := Wrap("hello", 0)
	if got != "hello" {
		t.Errorf("expected unchanged text for width=0, got %q", got)
	}
}

func TestWrap_ExactFit(t *testing.T) {
	got := Wrap("hello", 5)
	if got != "hello" {
		t.Errorf("expected no wrapping for exact fit, got %q", got)
	}
}

func TestWrap_NoBreakMidWord(t *testing.T) {
	// "aaa bbb" with width=5: "aaa" fits, " bbb" would make 7, so wrap
	got := Wrap("aaa bbb", 5)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "aaa" {
		t.Errorf("line 0: expected %q, got %q", "aaa", lines[0])
	}
	if lines[1] != "bbb" {
		t.Errorf("line 1: expected %q, got %q", "bbb", lines[1])
	}
}

func TestWrap_MultipleSpaces(t *testing.T) {
	got := Wrap("a b", 5)
	// Should handle multiple spaces gracefully
	if stringWidth(strings.Split(got, "\n")[0]) > 5 {
		t.Errorf("first line exceeds width")
	}
}

func TestWrap_FullwidthForms(t *testing.T) {
	// Fullwidth 'Ａ' (U+FF21) is in the fullwidth range
	got := Wrap("ＡＢＣ", 4)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "ＡＢ" {
		t.Errorf("line 0: expected %q, got %q", "ＡＢ", lines[0])
	}
	if lines[1] != "Ｃ" {
		t.Errorf("line 1: expected %q, got %q", "Ｃ", lines[1])
	}
}

func TestProperty_WrapWordBoundary(t *testing.T) {
	// Feature: markdown-terminal-renderer, Property 14: 単語境界での折り返し
	// Validates: Requirements 10.2
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(40, 200).Draw(t, "width")

		// Generate random ASCII English text: words of lowercase letters separated by spaces.
		// Each word is shorter than the width to ensure word-boundary wrapping is possible.
		maxWordLen := width - 1
		if maxWordLen < 1 {
			maxWordLen = 1
		}
		wordCount := rapid.IntRange(1, 30).Draw(t, "wordCount")
		words := make([]string, wordCount)
		for i := range words {
			wLen := rapid.IntRange(1, maxWordLen).Draw(t, fmt.Sprintf("wordLen_%d", i))
			runes := make([]rune, wLen)
			for j := range runes {
				runes[j] = rune(rapid.IntRange('a', 'z').Draw(t, fmt.Sprintf("char_%d_%d", i, j)))
			}
			words[i] = string(runes)
		}
		text := strings.Join(words, " ")

		result := Wrap(text, width)
		resultLines := strings.Split(result, "\n")

		// Verify each line does not start or end with a space (trimmed properly)
		for i, line := range resultLines {
			if len(line) > 0 {
				if line[0] == ' ' {
					t.Fatalf("line %d starts with a space: %q", i, line)
				}
				if line[len(line)-1] == ' ' {
					t.Fatalf("line %d ends with a space: %q", i, line)
				}
			}
		}

		// Verify each original word appears intact in exactly one line (not split across lines)
		for _, word := range words {
			found := false
			for _, line := range resultLines {
				if containsWord(line, word) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("word %q not found intact in any output line; lines: %v", word, resultLines)
			}
		}
	})
}

// containsWord checks if the line contains the word as a complete substring
// bounded by start/end of line or spaces.
func containsWord(line, word string) bool {
	idx := 0
	for {
		pos := strings.Index(line[idx:], word)
		if pos < 0 {
			return false
		}
		absPos := idx + pos
		endPos := absPos + len(word)
		// Check left boundary: start of line or preceded by space
		leftOk := absPos == 0 || line[absPos-1] == ' '
		// Check right boundary: end of line or followed by space
		rightOk := endPos == len(line) || line[endPos] == ' '
		if leftOk && rightOk {
			return true
		}
		idx = absPos + 1
		if idx >= len(line) {
			return false
		}
	}
}

func TestProperty_WrapWidthConstraint(t *testing.T) {
	// Feature: markdown-terminal-renderer, Property 13: テキスト折り返しの幅制約
	// Validates: Requirements 10.1
	rapid.Check(t, func(t *rapid.T) {
		// Generate random text: mix of ASCII and CJK characters
		text := rapid.Custom(func(t *rapid.T) string {
			length := rapid.IntRange(0, 200).Draw(t, "length")
			runes := make([]rune, length)
			for i := range runes {
				kind := rapid.IntRange(0, 3).Draw(t, fmt.Sprintf("kind_%d", i))
				switch kind {
				case 0:
					// ASCII letter
					runes[i] = rune(rapid.IntRange(0x41, 0x7A).Draw(t, fmt.Sprintf("ascii_%d", i)))
				case 1:
					// Space
					runes[i] = ' '
				case 2:
					// CJK Unified Ideograph
					runes[i] = rune(rapid.IntRange(0x4E00, 0x4FFF).Draw(t, fmt.Sprintf("cjk_%d", i)))
				case 3:
					// Hiragana
					runes[i] = rune(rapid.IntRange(0x3040, 0x309F).Draw(t, fmt.Sprintf("hira_%d", i)))
				}
			}
			return string(runes)
		}).Draw(t, "text")

		width := rapid.IntRange(40, 200).Draw(t, "width")

		result := Wrap(text, width)
		lines := strings.Split(result, "\n")

		for i, line := range lines {
			w := stringWidth(line)
			if w > width {
				t.Fatalf("line %d has display width %d, exceeds width %d: %q", i, w, width, line)
			}
		}
	})
}
