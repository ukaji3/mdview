package sixel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ukaji3/mdview/internal/imageutil"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// MaxImageDimension is the maximum allowed width or height in pixels.
// Images exceeding this in either dimension are scaled down to fit.
const MaxImageDimension = imageutil.MaxImageDimension

// Pre-decode limits to reject decompression bombs before full decoding.
const (
	maxDecodeDimension = 16384      // per-dimension cap checked via DecodeConfig
	maxDecodePixels    = 50_000_000 // total pixel cap checked via DecodeConfig
)

// ResizeImage resizes an image to fit within maxWidth while maintaining
// aspect ratio. Kept as a thin wrapper around imageutil.ResizeImage for
// backward compatibility.
func ResizeImage(img image.Image, maxWidth int) image.Image {
	return imageutil.ResizeImage(img, maxWidth)
}

// paletteEntry holds an RGB color for the Sixel palette.
type paletteEntry struct {
	R, G, B uint8
}

// quantizeColor maps an RGB color to a palette index (0-215).
// Uses a 6x6x6 color cube (216 colors).
func quantizeColor(r, g, b uint8) int {
	// Map to 6 levels each, with rounding centered on each bucket.
	ri := (int(r)*6 - 1) / 256
	gi := (int(g)*6 - 1) / 256
	bi := (int(b)*6 - 1) / 256
	if ri < 0 {
		ri = 0
	}
	if gi < 0 {
		gi = 0
	}
	if bi < 0 {
		bi = 0
	}
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

// alphaThreshold: pixels with alpha below this are treated as transparent
// and are not painted in the Sixel output.
const alphaThreshold = 128

// EncodeImage converts an image.Image to a Sixel escape sequence string.
// If the image width exceeds maxWidth, it is resized maintaining aspect ratio.
// Fully transparent pixels (alpha < 128) are left unpainted.
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

	// Precompute the palette index for every pixel once (-1 = transparent).
	// Direct Pix access avoids per-pixel interface calls.
	idxMap := make([]int16, w*h)
	usedColors := make(map[int]bool)
	pix := rgba.Pix
	for y := 0; y < h; y++ {
		rowOff := y * rgba.Stride
		for x := 0; x < w; x++ {
			o := rowOff + x*4
			if pix[o+3] < alphaThreshold {
				idxMap[y*w+x] = -1
				continue
			}
			idx := quantizeColor(pix[o], pix[o+1], pix[o+2])
			idxMap[y*w+x] = int16(idx)
			usedColors[idx] = true
		}
	}

	var buf strings.Builder

	// DCS introducer + Sixel start
	buf.WriteString("\x1bPq")

	// Write only used color definitions, in deterministic (sorted) order.
	palette := buildPalette()
	sortedColors := make([]int, 0, len(usedColors))
	for idx := range usedColors {
		sortedColors = append(sortedColors, idx)
	}
	sort.Ints(sortedColors)
	for _, idx := range sortedColors {
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

		// Collect colors used in this band (sorted for deterministic output)
		bandColorSet := make(map[int]bool)
		for y := bandTop; y < bandTop+6 && y < h; y++ {
			for x := 0; x < w; x++ {
				if ci := idxMap[y*w+x]; ci >= 0 {
					bandColorSet[int(ci)] = true
				}
			}
		}
		bandColors := make([]int, 0, len(bandColorSet))
		for ci := range bandColorSet {
			bandColors = append(bandColors, ci)
		}
		sort.Ints(bandColors)

		firstColor := true
		for _, colorIdx := range bandColors {
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
					if int(idxMap[y*w+x]) == colorIdx {
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

// Sentinel errors for image loading. Callers can classify failures with
// errors.Is instead of matching error message strings.
var (
	ErrOpenFailed     = errors.New("failed to open image file")
	ErrDownloadFailed = errors.New("failed to download image")
	ErrDecodeFailed   = errors.New("failed to decode image")
)

// checkDecodeSafety verifies image dimensions via DecodeConfig before full
// decoding to reject decompression bombs.
func checkDecodeSafety(r io.Reader) error {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}
	if cfg.Width > maxDecodeDimension || cfg.Height > maxDecodeDimension {
		return fmt.Errorf("%w: dimensions %dx%d exceed limit %d", ErrDecodeFailed, cfg.Width, cfg.Height, maxDecodeDimension)
	}
	if cfg.Width*cfg.Height > maxDecodePixels {
		return fmt.Errorf("%w: %dx%d exceeds pixel limit", ErrDecodeFailed, cfg.Width, cfg.Height)
	}
	return nil
}

// LoadLocalImage reads an image from a local file path.
// Supports PNG, JPEG, and GIF formats.
func LoadLocalImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOpenFailed, err)
	}
	defer f.Close()

	// Check dimensions before full decode (decompression bomb guard).
	if err := checkDecodeSafety(f); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	return img, nil
}

// maxDownloadSize is the maximum allowed image download size (50 MB).
const maxDownloadSize = 50 * 1024 * 1024

// maxRedirects is the maximum number of HTTP redirects followed.
const maxRedirects = 3

// isDisallowedIP returns true if the given IP must not be accessed by remote
// image loading: loopback, private, link-local, multicast, unspecified,
// CGN (100.64.0.0/10), and other special-purpose ranges.
func isDisallowedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// Additional special-purpose ranges not covered by the net.IP helpers.
	specialRanges := []string{
		"100.64.0.0/10",   // Carrier-grade NAT
		"192.0.0.0/24",    // IETF protocol assignments
		"192.0.2.0/24",    // TEST-NET-1
		"198.18.0.0/15",   // benchmark testing
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
		"240.0.0.0/4",     // reserved
		"255.255.255.255/32",
	}
	for _, cidr := range specialRanges {
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

// validateRemoteURL checks that the URL uses HTTP(S) and does not resolve to a
// private/loopback/special IP address (SSRF protection). Note that the
// authoritative check happens again at connect time in safeDialContext to
// prevent DNS rebinding.
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
		if ip != nil && isDisallowedIP(ip) {
			return fmt.Errorf("access to private IP address %s is not allowed", ipStr)
		}
	}
	return nil
}

// safeDialContext resolves the host itself and connects only to an allowed IP.
// Performing the check on the actually-dialed address (instead of a separate
// earlier lookup) closes the DNS-rebinding TOCTOU window.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %s: %w", host, err)
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ipAddr := range ips {
		if isDisallowedIP(ipAddr.IP) {
			lastErr = fmt.Errorf("access to private IP address %s is not allowed", ipAddr.IP)
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no addresses found for %s", host)
	}
	return nil, lastErr
}

// newSafeHTTPClient builds an HTTP client that validates the connect-time IP
// and every redirect hop.
func newSafeHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: safeDialContext,
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects (max %d)", maxRedirects)
			}
			// Re-validate each hop's URL (scheme + resolved IPs).
			if err := validateRemoteURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}
			return nil
		},
	}
}

// LoadRemoteImage downloads an image from an HTTP/HTTPS URL and decodes it.
// Supports PNG, JPEG, and GIF formats.
// Protections applied:
//   - URL scheme and DNS-resolution validation (validateRemoteURL)
//   - connect-time IP validation to prevent DNS rebinding (safeDialContext)
//   - redirect-hop validation with a hop limit
//   - 30-second timeout
//   - 50 MB download size limit (exceeding it is an error, not a truncation)
//   - image dimension check before full decode (decompression bomb guard)
func LoadRemoteImage(rawURL string) (image.Image, error) {
	if err := validateRemoteURL(rawURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	client := newSafeHTTPClient()
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrDownloadFailed, resp.StatusCode)
	}

	if resp.ContentLength > maxDownloadSize {
		return nil, fmt.Errorf("%w: size %d exceeds limit %d", ErrDownloadFailed, resp.ContentLength, maxDownloadSize)
	}

	// Read at most maxDownloadSize+1 bytes; more than maxDownloadSize is an error.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	if len(data) > maxDownloadSize {
		return nil, fmt.Errorf("%w: exceeds size limit %d", ErrDownloadFailed, maxDownloadSize)
	}

	// Check dimensions before full decode (decompression bomb guard).
	if err := checkDecodeSafety(bytes.NewReader(data)); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	return img, nil
}
