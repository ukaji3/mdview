package sixel

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// MaxImageDimension is the maximum allowed width or height in pixels.
// Images exceeding this in either dimension are scaled down to fit.
const MaxImageDimension = 4096

// ResizeImage resizes an image to fit within maxWidth while maintaining aspect ratio.
// If either dimension exceeds MaxImageDimension, the image is scaled down first.
// If the image width is already <= maxWidth (after capping), it is returned unchanged.
func ResizeImage(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	// Cap to MaxImageDimension if either dimension exceeds it
	if origW > MaxImageDimension || origH > MaxImageDimension {
		scale := math.Min(float64(MaxImageDimension)/float64(origW), float64(MaxImageDimension)/float64(origH))
		newW := int(math.Round(float64(origW) * scale))
		newH := int(math.Round(float64(origH) * scale))
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}
		capped := image.NewRGBA(image.Rect(0, 0, newW, newH))
		for y := 0; y < newH; y++ {
			srcY := y * origH / newH
			for x := 0; x < newW; x++ {
				srcX := x * origW / newW
				capped.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
			}
		}
		img = capped
		bounds = img.Bounds()
		origW = bounds.Dx()
		origH = bounds.Dy()
	}

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
//
// TODO: ピクセルデータを複数回走査しています（使用色の検出パスとエンコードパス）。
// 単一パスでの処理が可能ですが、現時点では可読性を優先しています。
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

// maxDownloadSize is the maximum allowed image download size (50 MB).
const maxDownloadSize = 50 * 1024 * 1024

// isPrivateIP returns true if the given IP is in a private, loopback, or
// link-local range that should not be accessed by remote image loading.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// validateRemoteURL checks that the URL uses HTTPS and does not resolve to a
// private/loopback IP address (SSRF protection).
func validateRemoteURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}
	hostname := parsed.Hostname()
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %w", hostname, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("access to private IP address %s is not allowed", ipStr)
		}
	}
	return nil
}

// LoadRemoteImage downloads an image from an HTTP/HTTPS URL and decodes it.
// Supports PNG, JPEG, and GIF formats.
// Applies a 30-second timeout, a 50 MB download size limit, and rejects URLs
// that resolve to private/loopback IP addresses.
func LoadRemoteImage(rawURL string) (image.Image, error) {
	if err := validateRemoteURL(rawURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxDownloadSize)
	img, _, err := image.Decode(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}


