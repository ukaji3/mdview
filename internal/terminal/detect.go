package terminal

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// ColorMode represents the terminal color capability level.
type ColorMode int

const (
	ColorNone ColorMode = iota // NO_COLOR or pipe output
	Color256                   // 256-color mode
	ColorTrue                  // TrueColor (24-bit)
)

// ImageProtocol represents the terminal image display protocol.
type ImageProtocol int

const (
	ImageNone   ImageProtocol = iota // No image support
	ImageSixel                       // Sixel graphics
	ImageKitty                       // Kitty graphics protocol
	ImageITerm2                      // iTerm2 inline images protocol
)

// Capabilities holds detected terminal capabilities.
type Capabilities struct {
	Width         int
	CellHeight    int // terminal cell height in pixels (for image row calculation)
	CellWidth     int // terminal cell width in pixels (for image sizing)
	ColorMode     ColorMode
	ImageProtocol ImageProtocol
	IsTTY         bool
	Theme         *Theme
}

// DetectColorMode determines the color mode from environment variables.
// - If NO_COLOR is set (any value), returns ColorNone.
// - If COLORTERM contains "truecolor" or "24bit", returns ColorTrue.
// - Otherwise returns Color256.
func DetectColorMode() ColorMode {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return ColorNone
	}

	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	if colorterm == "truecolor" || colorterm == "24bit" {
		return ColorTrue
	}

	termEnv := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(termEnv, "truecolor") || strings.Contains(termEnv, "24bit") {
		return ColorTrue
	}

	return Color256
}

// DetectWidth returns the terminal width in columns.
// Falls back to 80 if detection fails, and enforces a minimum of 40.
func DetectWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		w = 80
	}
	if w < 40 {
		w = 40
	}
	return w
}

// DetectSixel checks whether the terminal supports Sixel graphics by sending
// a DA1 (Device Attributes) query and checking the response for attribute 4.
// Only probes when stdin/stdout are terminals.
func DetectSixel() bool {
	return probeSixel()
}

// DetectImageProtocol detects which image protocol the terminal supports.
// It checks environment variables first, then falls back to a DA1 probe for
// Sixel support.
func DetectImageProtocol() ImageProtocol {
	termProgram := os.Getenv("TERM_PROGRAM")

	switch termProgram {
	case "WezTerm", "iTerm.app", "mintty":
		return ImageITerm2
	case "kitty", "ghostty":
		return ImageKitty
	}

	termEnv := os.Getenv("TERM")
	if strings.Contains(termEnv, "kitty") || strings.Contains(termEnv, "ghostty") {
		return ImageKitty
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return ImageKitty
	}

	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		return ImageITerm2
	}

	if DetectSixel() {
		return ImageSixel
	}

	return ImageNone
}

// DetectCellHeight returns the terminal cell height in pixels.
// It uses the TIOCGWINSZ ioctl to get pixel dimensions and divides by rows.
// Falls back to 16 pixels if detection fails or yields a nonsensical value.
func DetectCellHeight() int {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Row == 0 || ws.Ypixel == 0 {
		return 16 // fallback: typical cell height
	}
	h := int(ws.Ypixel) / int(ws.Row)
	if h < 1 {
		return 16
	}
	return h
}

// DetectCellWidth returns the terminal cell width in pixels.
// It uses the TIOCGWINSZ ioctl to get pixel dimensions and divides by columns.
// Falls back to 8 pixels if detection fails or yields a nonsensical value.
func DetectCellWidth() int {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Xpixel == 0 {
		return 8 // fallback: typical cell width
	}
	w := int(ws.Xpixel) / int(ws.Col)
	if w < 1 {
		return 8
	}
	return w
}

// DetectCapabilities detects all terminal capabilities and returns
// a Capabilities struct ready for use by the renderer.
func DetectCapabilities() *Capabilities {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	mode := DetectColorMode()
	if !isTTY {
		mode = ColorNone
	}

	imgProto := ImageNone
	if isTTY {
		imgProto = DetectImageProtocol()
	}

	return &Capabilities{
		Width:         DetectWidth(),
		CellHeight:    DetectCellHeight(),
		CellWidth:     DetectCellWidth(),
		ColorMode:     mode,
		ImageProtocol: imgProto,
		IsTTY:         isTTY,
		Theme:         DefaultTheme(),
	}
}
