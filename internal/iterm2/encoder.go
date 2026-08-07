package iterm2

import (
	"fmt"
	"image"
	"strings"

	"github.com/ukaji3/mdview/internal/imageutil"
)

// EncodeImage converts an image.Image to an iTerm2 inline image escape sequence.
// The image is resized to fit within maxWidth, encoded as PNG, base64-encoded,
// and wrapped in the OSC 1337 escape sequence.
func EncodeImage(img image.Image, maxWidth int) (string, error) {
	if img == nil {
		return "", fmt.Errorf("nil image")
	}

	pngBytes, b64Data, err := imageutil.EncodePNGBase64(img, maxWidth)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "\x1b]1337;File=inline=1;size=%d;width=auto;height=auto;preserveAspectRatio=1:%s\x07",
		len(pngBytes), b64Data)

	return buf.String(), nil
}
