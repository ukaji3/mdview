package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ImageProtocolName returns a human-readable name for the protocol.
func ImageProtocolName(p ImageProtocol) string {
	switch p {
	case ImageSixel:
		return "Sixel"
	case ImageKitty:
		return "Kitty Graphics Protocol"
	case ImageITerm2:
		return "iTerm2 Inline Images"
	default:
		return "None"
	}
}

// CheckImageSupport prints detailed diagnostic information about
// terminal image protocol support.
func CheckImageSupport() {
	fmt.Println("=== mdview: Terminal Image Protocol Check ===")
	fmt.Println()

	// Basic terminal info
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	fmt.Printf("TTY: %v\n", isTTY)

	if !isTTY {
		fmt.Println("  Not a TTY — image display is disabled.")
		fmt.Println("  Run mdview directly in a terminal (not piped).")
		return
	}

	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	fmt.Printf("Terminal size: %dx%d\n", w, h)
	fmt.Printf("TERM: %q\n", os.Getenv("TERM"))
	fmt.Printf("TERM_PROGRAM: %q\n", os.Getenv("TERM_PROGRAM"))
	fmt.Printf("TERM_PROGRAM_VERSION: %q\n", os.Getenv("TERM_PROGRAM_VERSION"))
	fmt.Printf("COLORTERM: %q\n", os.Getenv("COLORTERM"))
	fmt.Printf("LC_TERMINAL: %q\n", os.Getenv("LC_TERMINAL"))
	fmt.Printf("KITTY_WINDOW_ID: %q\n", os.Getenv("KITTY_WINDOW_ID"))
	fmt.Println()

	// Environment-based detection
	envProto := DetectImageProtocol()
	fmt.Printf("Environment-based detection: %s\n", ImageProtocolName(envProto))
	fmt.Println()

	// Probe-based detection
	fmt.Println("--- Protocol Probes ---")
	fmt.Println()

	kittyOK := probeKitty()
	fmt.Printf("Kitty Graphics Protocol: %s\n", boolStatus(kittyOK))

	sixelOK := probeSixel()
	fmt.Printf("Sixel Graphics:          %s\n", boolStatus(sixelOK))

	iterm2OK := probeITerm2()
	fmt.Printf("iTerm2 Inline Images:    %s\n", boolStatus(iterm2OK))
	fmt.Println()

	// Determine best protocol
	var bestProto ImageProtocol
	switch {
	case kittyOK:
		bestProto = ImageKitty
	case iterm2OK:
		bestProto = ImageITerm2
	case sixelOK:
		bestProto = ImageSixel
	default:
		bestProto = envProto // fall back to env detection
	}

	if bestProto == ImageNone {
		fmt.Println("Result: No image protocol detected.")
		fmt.Println("  Images will be shown as [画像: alt text] fallback.")
		fmt.Println()
		fmt.Println("Supported terminals for image display:")
		fmt.Println("  - Kitty")
		fmt.Println("  - Ghostty")
		fmt.Println("  - WezTerm")
		fmt.Println("  - iTerm2")
		fmt.Println("  - mintty")
		fmt.Println("  - Sixel-capable terminals (mlterm, foot, etc.)")
	} else {
		fmt.Printf("Result: %s detected — images will be displayed.\n", ImageProtocolName(bestProto))
	}
}

func boolStatus(ok bool) string {
	if ok {
		return "supported ✓"
	}
	return "not detected ✗"
}

// probeKitty sends a Kitty graphics query and checks for a response.
// Query: \x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\
// Expected response contains: \x1b_G
func probeKitty() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	termProg := os.Getenv("TERM_PROGRAM")
	if termProg == "kitty" || termProg == "ghostty" {
		return true
	}
	termEnv := os.Getenv("TERM")
	if strings.Contains(termEnv, "kitty") || strings.Contains(termEnv, "ghostty") {
		return true
	}

	return probeEscape("\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\", "\x1b_G")
}

// probeSixel sends a DA1 (Device Attributes) query and checks for Sixel support.
// Query: \x1b[c
// Sixel support indicated by ";4" in the response.
func probeSixel() bool {
	return probeEscape("\x1b[c", ";4")
}

// probeITerm2 checks for iTerm2-compatible terminals via env vars.
// There's no reliable escape-based probe for iTerm2 inline images.
func probeITerm2() bool {
	termProg := os.Getenv("TERM_PROGRAM")
	switch termProg {
	case "iTerm.app", "WezTerm", "mintty":
		return true
	}
	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		return true
	}
	return false
}

// probeEscape sends an escape sequence to the terminal and reads the response,
// checking if it contains the expected substring.
func probeEscape(query, expect string) bool {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return false
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, oldState)

	// Send query
	os.Stdout.WriteString(query)

	// Read response with timeout
	reader := bufio.NewReader(os.Stdin)
	done := make(chan string, 1)
	go func() {
		var resp strings.Builder
		buf := make([]byte, 128)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				resp.Write(buf[:n])
				// Check if we have enough data
				if strings.Contains(resp.String(), expect) {
					done <- resp.String()
					return
				}
			}
			if err != nil {
				done <- resp.String()
				return
			}
		}
	}()

	select {
	case resp := <-done:
		return strings.Contains(resp, expect)
	case <-time.After(500 * time.Millisecond):
		return false
	}
}
