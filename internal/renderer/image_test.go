package renderer

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// createTestPNG creates a small PNG file at the given path and returns a cleanup function.
func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
}

// TestProperty20_SixelImageCaption tests that when Sixel support is enabled,
// the rendered output contains the alt text as a caption after the Sixel data.
//
// Feature: markdown-terminal-renderer, Property 20: Sixel画像のキャプション表示
// **Validates: Requirements 13.6**
func TestProperty20_SixelImageCaption(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random alt text (ASCII letters and spaces, non-empty)
		altText := rapid.StringMatching(`[a-zA-Z][a-zA-Z ]{0,20}`).Draw(t, "altText")

		// Create a temporary PNG file
		tmpDir, err := os.MkdirTemp("", "mermaid-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)
		imgPath := filepath.Join(tmpDir, "test.png")
		createTestPNGRapid(imgPath, 10, 10)

		// Build markdown with image reference
		md := fmt.Sprintf("![%s](%s)", altText, imgPath)
		source := []byte(md)
		node := parser.Parse(source)

		ctx := &RenderContext{
			TermWidth:     80,
			ColorMode:     terminal.ColorTrue,
			ImageProtocol: terminal.ImageSixel,
			Theme:         terminal.DefaultTheme(),
			IsTTY:         true,
		}

		result := Render(node, source, ctx)

		// The result must contain the Sixel DCS sequence
		if !strings.Contains(result, "\x1bP") {
			t.Fatalf("Sixel output missing DCS start for alt=%q", altText)
		}

		// The result must contain the alt text as caption after the Sixel data
		sixelEnd := strings.LastIndex(result, "\x1b\\")
		if sixelEnd < 0 {
			t.Fatalf("Sixel output missing ST terminator for alt=%q", altText)
		}

		afterSixel := result[sixelEnd+2:]
		if !strings.Contains(afterSixel, altText) {
			t.Fatalf("caption not found after Sixel data for alt=%q, afterSixel=%q", altText, afterSixel)
		}
	})
}

// createTestPNGRapid creates a small PNG file (non-test-helper version for rapid).
func createTestPNGRapid(path string, w, h int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		panic(fmt.Sprintf("failed to create test PNG: %v", err))
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(fmt.Sprintf("failed to encode test PNG: %v", err))
	}
}

// TestProperty24_ImageFallbackNoSixel tests that when Sixel is not supported,
// images are rendered as "[画像: <altテキスト>]" format.
//
// Feature: markdown-terminal-renderer, Property 24: 画像フォールバック（Sixel非サポート時）
// **Validates: Requirements 9.4**
func TestProperty24_ImageFallbackNoSixel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random alt text
		altText := rapid.StringMatching(`[a-zA-Z][a-zA-Z ]{0,20}`).Draw(t, "altText")

		// Build markdown with image reference (path doesn't matter for fallback)
		md := fmt.Sprintf("![%s](some-image.png)", altText)
		source := []byte(md)
		node := parser.Parse(source)

		ctx := &RenderContext{
			TermWidth:     80,
			ColorMode:     terminal.ColorTrue,
			ImageProtocol: terminal.ImageNone, // No image support
			Theme:         terminal.DefaultTheme(),
			IsTTY:         true,
		}

		result := Render(node, source, ctx)

		// Must contain the fallback format
		expected := fmt.Sprintf("[画像: %s]", altText)
		if !strings.Contains(result, expected) {
			t.Fatalf("fallback text not found: expected %q in result %q", expected, result)
		}

		// Must NOT contain Sixel DCS sequence
		if strings.Contains(result, "\x1bP") {
			t.Fatalf("Sixel data found in non-Sixel mode for alt=%q", altText)
		}
	})
}

// TestImageErrorFileNotFound tests that a missing image file produces the correct error message.
func TestImageErrorFileNotFound(t *testing.T) {
	md := "![test image](/nonexistent/path/image.png)"
	source := []byte(md)
	node := parser.Parse(source)

	ctx := &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageSixel,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         true,
	}

	result := Render(node, source, ctx)
	if !strings.Contains(result, "[画像読み込みエラー: test image]") {
		t.Fatalf("expected file not found error, got: %q", result)
	}
}

// TestImageFallbackPlainText tests that in NO_COLOR mode, images show as plain text.
func TestImageFallbackPlainText(t *testing.T) {
	md := "![my alt text](some-image.png)"
	source := []byte(md)
	node := parser.Parse(source)

	ctx := &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorNone,
		ImageProtocol: terminal.ImageNone,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         false,
	}

	result := Render(node, source, ctx)
	if !strings.Contains(result, "[画像: my alt text]") {
		t.Fatalf("expected plain text fallback, got: %q", result)
	}
	// Should not contain ANSI codes
	if strings.Contains(result, "\033[") {
		t.Fatalf("ANSI codes found in NO_COLOR mode: %q", result)
	}
}

// TestImageRows tests the imageRows helper function.
func TestImageRows(t *testing.T) {
	tests := []struct {
		imgHeight  int
		cellHeight int
		expected   int
	}{
		{160, 16, 10},  // exact division
		{161, 16, 11},  // rounds up
		{1, 16, 1},     // minimum 1 row
		{0, 16, 1},     // zero height -> 1 row minimum (via ceil)
		{100, 20, 5},   // exact division
		{101, 20, 6},   // rounds up
		{16, 0, 1},     // zero cell height -> fallback to 16, ceil(16/16)=1
		{32, -1, 2},    // negative cell height -> fallback to 16, ceil(32/16)=2
	}
	for _, tt := range tests {
		got := imageRows(tt.imgHeight, tt.cellHeight)
		if got != tt.expected {
			t.Errorf("imageRows(%d, %d) = %d, want %d", tt.imgHeight, tt.cellHeight, got, tt.expected)
		}
	}
}

// TestImagePlaceholderLines tests that rendered images include placeholder lines
// to account for the image's visual height in the pager.
func TestImagePlaceholderLines(t *testing.T) {
	// Create a tall test image (10x160 pixels)
	tmpDir, err := os.MkdirTemp("", "img-placeholder-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	imgPath := filepath.Join(tmpDir, "tall.png")
	createTestPNG(t, imgPath, 10, 160)

	md := fmt.Sprintf("![tall image](%s)", imgPath)
	source := []byte(md)
	node := parser.Parse(source)

	// Use CellHeight=16 so 160px image = 10 rows, meaning 9 placeholder lines
	ctx := &RenderContext{
		TermWidth:     80,
		CellHeight:    16,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageSixel,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         true,
	}

	result := Render(node, source, ctx)

	// The Sixel escape sequence ends with \x1b\\ (ST).
	// After the ST + newline, there should be placeholder empty lines before the caption.
	sixelEnd := strings.LastIndex(result, "\x1b\\")
	if sixelEnd < 0 {
		t.Fatalf("Sixel output missing ST terminator")
	}

	afterSixel := result[sixelEnd+2:]
	// Count leading newlines (first newline is the line after the escape sequence,
	// then rows-1 placeholder newlines)
	newlineCount := 0
	for _, ch := range afterSixel {
		if ch == '\n' {
			newlineCount++
		} else {
			break
		}
	}

	// 160px / 16px = 10 rows. The escape sequence itself is 1 line,
	// so we need 9 placeholder lines = 9 additional newlines.
	// Plus the first newline after the escape sequence = 10 newlines total.
	expectedNewlines := 10 // 1 (after escape) + 9 (placeholders)
	if newlineCount != expectedNewlines {
		t.Errorf("expected %d newlines after Sixel data, got %d", expectedNewlines, newlineCount)
	}
}
