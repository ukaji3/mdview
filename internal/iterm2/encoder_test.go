package iterm2

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
	// Should start with OSC 1337
	if !strings.HasPrefix(result, "\x1b]1337;File=inline=1;size=") {
		t.Fatalf("expected iTerm2 OSC 1337 header, got prefix: %q", result[:min(40, len(result))])
	}
	// Should end with BEL
	if !strings.HasSuffix(result, "\x07") {
		t.Fatalf("expected BEL terminator")
	}
}

func TestEncodeImage_ITerm2EscapeFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := rapid.IntRange(1, 50).Draw(t, "width")
		h := rapid.IntRange(1, 50).Draw(t, "height")
		maxWidth := rapid.IntRange(1, 200).Draw(t, "maxWidth")

		img := genImage(w, h)
		result, err := EncodeImage(img, maxWidth)
		if err != nil {
			t.Fatalf("EncodeImage failed: %v", err)
		}

		// Must start with OSC 1337
		if !strings.HasPrefix(result, "\x1b]1337;") {
			t.Fatalf("iTerm2 output does not start with \\x1b]1337;, got prefix: %q", result[:min(15, len(result))])
		}

		// Must end with BEL
		if !strings.HasSuffix(result, "\x07") {
			t.Fatalf("iTerm2 output does not end with BEL")
		}

		// Must contain inline=1
		if !strings.Contains(result, "inline=1") {
			t.Fatalf("iTerm2 output missing inline=1 parameter")
		}

		// Must contain preserveAspectRatio=1
		if !strings.Contains(result, "preserveAspectRatio=1") {
			t.Fatalf("iTerm2 output missing preserveAspectRatio=1 parameter")
		}

		// Must contain size= parameter
		if !strings.Contains(result, "size=") {
			t.Fatalf("iTerm2 output missing size= parameter")
		}
	})
}
