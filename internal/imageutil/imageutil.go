// Package imageutil provides protocol-neutral image helpers shared by the
// Sixel, Kitty, and iTerm2 encoders.
package imageutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math"
)

// MaxImageDimension is the maximum allowed width or height in pixels.
// Images exceeding this in either dimension are scaled down to fit.
const MaxImageDimension = 4096

// ResizeImage resizes an image to fit within maxWidth while maintaining aspect
// ratio. If either dimension exceeds MaxImageDimension, the image is scaled
// down to fit that cap as well. Both constraints are applied in a single
// nearest-neighbor pass. If the image already satisfies both constraints, it
// is returned unchanged.
func ResizeImage(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	if origW <= 0 || origH <= 0 {
		return img
	}

	// Compose both constraints into a single scale factor.
	scale := 1.0
	if origW > MaxImageDimension || origH > MaxImageDimension {
		scale = math.Min(float64(MaxImageDimension)/float64(origW), float64(MaxImageDimension)/float64(origH))
	}
	if maxWidth > 0 && float64(origW)*scale > float64(maxWidth) {
		scale = float64(maxWidth) / float64(origW)
	}
	if scale >= 1.0 {
		return img
	}

	newW := int(math.Round(float64(origW) * scale))
	newH := int(math.Round(float64(origH) * scale))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))

	// Nearest-neighbor resize (single pass)
	for y := 0; y < newH; y++ {
		srcY := y * origH / newH
		for x := 0; x < newW; x++ {
			srcX := x * origW / newW
			dst.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	return dst
}

// EncodePNGBase64 resizes img to fit maxWidth, encodes it as PNG, and returns
// the raw PNG bytes together with their base64 encoding. Shared by the Kitty
// and iTerm2 encoders.
func EncodePNGBase64(img image.Image, maxWidth int) (pngBytes []byte, b64 string, err error) {
	if img == nil {
		return nil, "", fmt.Errorf("nil image")
	}
	img = ResizeImage(img, maxWidth)

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, "", fmt.Errorf("failed to encode PNG: %w", err)
	}
	pngBytes = pngBuf.Bytes()
	return pngBytes, base64.StdEncoding.EncodeToString(pngBytes), nil
}
