package sixel

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// genImage generates a random RGBA image with the given width and height.
func genImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 37 + y * 53) % 256),
				G: uint8((x * 59 + y * 71) % 256),
				B: uint8((x * 83 + y * 97) % 256),
				A: 255,
			})
		}
	}
	return img
}

// TestProperty19_SixelEscapeSequenceFormat tests that Sixel encoded output
// starts with \x1bP and ends with \x1b\\ for any valid image.
//
// Feature: markdown-terminal-renderer, Property 19: Sixelエスケープシーケンス形式
// **Validates: Requirements 13.3**
func TestProperty19_SixelEscapeSequenceFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := rapid.IntRange(1, 100).Draw(t, "width")
		h := rapid.IntRange(1, 100).Draw(t, "height")
		maxWidth := rapid.IntRange(1, 200).Draw(t, "maxWidth")

		img := genImage(w, h)
		result, err := EncodeImage(img, maxWidth)
		if err != nil {
			t.Fatalf("EncodeImage failed: %v", err)
		}

		// Must start with DCS introducer \x1bP
		if !strings.HasPrefix(result, "\x1bP") {
			t.Fatalf("Sixel output does not start with \\x1bP, got prefix: %q", result[:min(10, len(result))])
		}

		// Must end with ST (String Terminator) \x1b\\
		if !strings.HasSuffix(result, "\x1b\\") {
			t.Fatalf("Sixel output does not end with \\x1b\\\\, got suffix: %q", result[max(0, len(result)-10):])
		}
	})
}

// TestProperty18_ImageResizeAspectRatio tests that resized images maintain
// aspect ratio and respect the maxWidth constraint.
//
// Feature: markdown-terminal-renderer, Property 18: 画像リサイズとアスペクト比維持
// **Validates: Requirements 13.4, 13.5**
func TestProperty18_ImageResizeAspectRatio(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		origW := rapid.IntRange(1, 500).Draw(t, "origWidth")
		origH := rapid.IntRange(1, 500).Draw(t, "origHeight")
		termWidth := rapid.IntRange(40, 300).Draw(t, "termWidth")

		img := genImage(origW, origH)

		// maxWidth is 80% of terminal width
		maxWidth := int(float64(termWidth) * 0.8)
		if maxWidth < 1 {
			maxWidth = 1
		}

		resized := ResizeImage(img, maxWidth)
		bounds := resized.Bounds()
		newW := bounds.Dx()
		newH := bounds.Dy()

		// Width must not exceed maxWidth
		expectedMaxW := origW
		if expectedMaxW > maxWidth {
			expectedMaxW = maxWidth
		}
		if newW > expectedMaxW {
			t.Fatalf("resized width %d exceeds expected max %d (origW=%d, maxWidth=%d)", newW, expectedMaxW, origW, maxWidth)
		}

		// If original was wider than maxWidth, check aspect ratio is maintained
		if origW > maxWidth {
			origRatio := float64(origW) / float64(origH)
			newRatio := float64(newW) / float64(newH)
			// Allow tolerance proportional to 1/newH for integer rounding
			tolerance := 1.0 / float64(newH)
			if tolerance < 0.05 {
				tolerance = 0.05
			}
			if math.Abs(origRatio-newRatio)/origRatio > tolerance {
				t.Fatalf("aspect ratio not maintained: orig=%f, new=%f (origW=%d, origH=%d, newW=%d, newH=%d)",
					origRatio, newRatio, origW, origH, newW, newH)
			}
		} else {
			// Image should not be resized
			if newW != origW || newH != origH {
				t.Fatalf("image was resized when it shouldn't have been: orig=(%d,%d), new=(%d,%d), maxWidth=%d",
					origW, origH, newW, newH, maxWidth)
			}
		}
	})
}
