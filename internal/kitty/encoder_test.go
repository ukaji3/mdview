package kitty

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func genImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x*37 + y*53) % 256),
				G: uint8((x*59 + y*71) % 256),
				B: uint8((x*83 + y*97) % 256),
				A: 255,
			})
		}
	}
	return img
}

func TestEncodeImage_NilImage(t *testing.T) {
	_, err := EncodeImage(nil, 100)
	if err == nil {
		t.Fatal("expected error for nil image")
	}
}

func TestEncodeImage_SmallImage(t *testing.T) {
	img := genImage(2, 2)
	result, err := EncodeImage(img, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Single chunk: should start with APC and contain f=100,a=T,t=d
	if !strings.HasPrefix(result, "\x1b_Gf=100,a=T,t=d;") {
		t.Fatalf("expected Kitty APC header, got prefix: %q", result[:min(30, len(result))])
	}
	// Should end with ST
	if !strings.HasSuffix(result, "\x1b\\") {
		t.Fatalf("expected ST terminator, got suffix: %q", result[max(0, len(result)-10):])
	}
}

func TestEncodeImage_KittyEscapeFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := rapid.IntRange(1, 50).Draw(t, "width")
		h := rapid.IntRange(1, 50).Draw(t, "height")
		maxWidth := rapid.IntRange(1, 200).Draw(t, "maxWidth")

		img := genImage(w, h)
		result, err := EncodeImage(img, maxWidth)
		if err != nil {
			t.Fatalf("EncodeImage failed: %v", err)
		}

		// Must start with Kitty APC sequence
		if !strings.HasPrefix(result, "\x1b_G") {
			t.Fatalf("Kitty output does not start with \\x1b_G, got prefix: %q", result[:min(10, len(result))])
		}

		// Must end with ST
		if !strings.HasSuffix(result, "\x1b\\") {
			t.Fatalf("Kitty output does not end with \\x1b\\\\, got suffix: %q", result[max(0, len(result)-10):])
		}

		// Must contain f=100 (PNG format)
		if !strings.Contains(result, "f=100") {
			t.Fatalf("Kitty output missing f=100 parameter")
		}
	})
}

func TestSplitChunks(t *testing.T) {
	chunks := splitChunks("abcdefghij", 3)
	expected := []string{"abc", "def", "ghi", "j"}
	if len(chunks) != len(expected) {
		t.Fatalf("expected %d chunks, got %d", len(expected), len(chunks))
	}
	for i, c := range chunks {
		if c != expected[i] {
			t.Errorf("chunk %d: expected %q, got %q", i, expected[i], c)
		}
	}
}

func TestSplitChunks_Empty(t *testing.T) {
	chunks := splitChunks("", 3)
	if len(chunks) != 1 || chunks[0] != "" {
		t.Fatalf("expected single empty chunk, got %v", chunks)
	}
}

func TestSplitChunks_ExactFit(t *testing.T) {
	chunks := splitChunks("abcdef", 3)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != "abc" || chunks[1] != "def" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}
