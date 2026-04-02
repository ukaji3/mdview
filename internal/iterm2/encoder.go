package iterm2

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"

	"github.com/user/mdrender/internal/sixel"
)

// EncodeImage converts an image.Image to an iTerm2 inline image escape sequence.
// The image is resized to fit within maxWidth, encoded as PNG, base64-encoded,
// and wrapped in the OSC 1337 escape sequence.
func EncodeImage(img image.Image, maxWidth int) (string, error) {
	if img == nil {
		return "", fmt.Errorf("nil image")
	}

	img = sixel.ResizeImage(img, maxWidth)

	// Encode as PNG
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	pngBytes := pngBuf.Bytes()
	b64Data := base64.StdEncoding.EncodeToString(pngBytes)

	var buf strings.Builder
	fmt.Fprintf(&buf, "\x1b]1337;File=inline=1;size=%d;width=auto;height=auto;preserveAspectRatio=1:%s\x07",
		len(pngBytes), b64Data)

	return buf.String(), nil
}
