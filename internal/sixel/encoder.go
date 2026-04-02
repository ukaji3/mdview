package sixel

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/http"
	"os"
	"strings"
)

// ResizeImage resizes an image to fit within maxWidth while maintaining aspect ratio.
// If the image width is already <= maxWidth, it is returned unchanged.
func ResizeImage(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	if origW <= maxWidth {
		return img
	}

	newW := maxWidth
	newH := int(math.Round(float64(origH) * float64(newW) / float64(origW)))
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))

	// Nearest-neighbor resize
	for y := 0; y < newH; y++ {
		srcY := y * origH / newH
		for x := 0; x < newW; x++ {
			srcX := x * origW / newW
			dst.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	return dst
}

// paletteEntry holds an RGB color for the Sixel palette.
type paletteEntry struct {
	R, G, B uint8
}

// quantizeColor maps an RGBA color to a palette index (0-255).
// Uses a simple 6x6x6 color cube (216 colors) plus a grayscale ramp.
func quantizeColor(r, g, b uint8) int {
	// Map to 6 levels each
	ri := int(r) * 5 / 255
	gi := int(g) * 5 / 255
	bi := int(b) * 5 / 255
	return ri*36 + gi*6 + bi
}

// buildPalette generates a 216-color palette (6x6x6 cube).
func buildPalette() []paletteEntry {
	palette := make([]paletteEntry, 216)
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				idx := r*36 + g*6 + b
				palette[idx] = paletteEntry{
					R: uint8(r * 255 / 5),
					G: uint8(g * 255 / 5),
					B: uint8(b * 255 / 5),
				}
			}
		}
	}
	return palette
}

// EncodeImage converts an image.Image to a Sixel escape sequence string.
// If the image width exceeds maxWidth, it is resized maintaining aspect ratio.
// The Sixel format: \x1bPq <color defs> <pixel data> \x1b\\
func EncodeImage(img image.Image, maxWidth int) (string, error) {
	if img == nil {
		return "", fmt.Errorf("nil image")
	}

	// Resize if needed
	img = ResizeImage(img, maxWidth)

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w == 0 || h == 0 {
		return "", fmt.Errorf("image has zero dimensions")
	}

	// Convert to RGBA for uniform pixel access
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)

	var buf strings.Builder

	// DCS introducer + Sixel start
	buf.WriteString("\x1bPq")

	// Write color palette definitions
	palette := buildPalette()
	usedColors := make(map[int]bool)

	// First pass: find which palette colors are actually used
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := rgba.At(x, y).RGBA()
			idx := quantizeColor(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			usedColors[idx] = true
		}
	}

	// Write only used color definitions
	for idx := range usedColors {
		p := palette[idx]
		// Sixel color definition: #idx;2;R%;G%;B% (percentages 0-100)
		rp := int(p.R) * 100 / 255
		gp := int(p.G) * 100 / 255
		bp := int(p.B) * 100 / 255
		fmt.Fprintf(&buf, "#%d;2;%d;%d;%d", idx, rp, gp, bp)
	}

	// Encode pixel data in bands of 6 rows
	for band := 0; band*6 < h; band++ {
		bandTop := band * 6

		// For each color used in this band, emit a row of sixel data
		bandColors := make(map[int]bool)
		for y := bandTop; y < bandTop+6 && y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, _ := rgba.At(x, y).RGBA()
				idx := quantizeColor(uint8(r>>8), uint8(g>>8), uint8(b>>8))
				bandColors[idx] = true
			}
		}

		firstColor := true
		for colorIdx := range bandColors {
			if !firstColor {
				buf.WriteByte('$') // Carriage return (go back to start of band)
			}
			firstColor = false

			// Select color
			fmt.Fprintf(&buf, "#%d", colorIdx)

			// Encode each column for this color in this band
			for x := 0; x < w; x++ {
				sixelByte := byte(0)
				for bit := 0; bit < 6; bit++ {
					y := bandTop + bit
					if y >= h {
						break
					}
					r, g, b, _ := rgba.At(x, y).RGBA()
					idx := quantizeColor(uint8(r>>8), uint8(g>>8), uint8(b>>8))
					if idx == colorIdx {
						sixelByte |= 1 << uint(bit)
					}
				}
				buf.WriteByte(sixelByte + 0x3F)
			}
		}

		buf.WriteByte('-') // New line (next band)
	}

	// ST (String Terminator)
	buf.WriteString("\x1b\\")

	return buf.String(), nil
}

// LoadLocalImage reads an image from a local file path.
// Supports PNG, JPEG, and GIF formats.
func LoadLocalImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// LoadRemoteImage downloads an image from an HTTP/HTTPS URL and decodes it.
// Supports PNG, JPEG, and GIF formats.
func LoadRemoteImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// colorToRGBA extracts uint8 RGBA components from a color.Color.
func colorToRGBA(c color.Color) (r, g, b, a uint8) {
	rr, gg, bb, aa := c.RGBA()
	return uint8(rr >> 8), uint8(gg >> 8), uint8(bb >> 8), uint8(aa >> 8)
}
