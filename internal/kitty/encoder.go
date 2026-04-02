package kitty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"

	"github.com/user/mdrender/internal/sixel"
)

const chunkSize = 4096

// EncodeImage converts an image.Image to a Kitty graphics protocol escape sequence.
// The image is resized to fit within maxWidth, encoded as PNG, base64-encoded,
// and wrapped in APC escape sequences with chunked transmission.
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

	// Base64 encode
	b64Data := base64.StdEncoding.EncodeToString(pngBuf.Bytes())

	var buf strings.Builder

	chunks := splitChunks(b64Data, chunkSize)

	if len(chunks) == 1 {
		// Single chunk: no m parameter needed (or m=0)
		buf.WriteString("\x1b_Gf=100,a=T,t=d;")
		buf.WriteString(chunks[0])
		buf.WriteString("\x1b\\")
	} else {
		for i, chunk := range chunks {
			if i == 0 {
				// First chunk
				buf.WriteString("\x1b_Gf=100,a=T,t=d,m=1;")
			} else if i == len(chunks)-1 {
				// Last chunk
				buf.WriteString("\x1b_Gm=0;")
			} else {
				// Middle chunk
				buf.WriteString("\x1b_Gm=1;")
			}
			buf.WriteString(chunk)
			buf.WriteString("\x1b\\")
		}
	}

	return buf.String(), nil
}

// splitChunks splits a string into chunks of the given size.
func splitChunks(s string, size int) []string {
	if len(s) == 0 {
		return []string{""}
	}
	var chunks []string
	for len(s) > 0 {
		end := size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[:end])
		s = s[end:]
	}
	return chunks
}
