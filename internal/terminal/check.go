package terminal

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
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

// probeKitty sends a Kitty graphics query followed by a DA1 query and checks
// for a Kitty graphics response. The DA1 query acts as a sentinel: virtually
// all terminals answer DA1, so its arrival means the terminal has finished
// responding (whether or not it supports the Kitty protocol).
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

	resp, ok := probeQuery("\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\\x1b[c", hasDA1Response)
	if !ok {
		return false
	}
	return strings.Contains(resp, "\x1b_G")
}

// probeSixel sends a DA1 (Device Attributes) query and checks the attribute
// list for "4" (Sixel graphics). Attributes are compared exactly, not by
// substring, to avoid false positives on attributes like 42 or 44.
func probeSixel() bool {
	resp, ok := probeQuery("\x1b[c", hasDA1Response)
	if !ok {
		return false
	}
	return da1HasAttribute(resp, "4")
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

// hasDA1Response reports whether s contains a complete DA1 response
// (CSI ? ... c).
func hasDA1Response(s string) bool {
	start := strings.Index(s, "\x1b[?")
	if start < 0 {
		return false
	}
	return strings.IndexByte(s[start:], 'c') >= 0
}

// da1HasAttribute parses the DA1 response in s and reports whether the
// attribute list contains exactly attr.
func da1HasAttribute(s, attr string) bool {
	start := strings.Index(s, "\x1b[?")
	if start < 0 {
		return false
	}
	body := s[start+3:]
	end := strings.IndexByte(body, 'c')
	if end < 0 {
		return false
	}
	for _, a := range strings.Split(body[:end], ";") {
		if a == attr {
			return true
		}
	}
	return false
}

// probeQuery sends an escape sequence to the terminal and synchronously reads
// the response using poll with a deadline. No goroutine is spawned, so probes
// never leak readers or steal each other's responses. Any pending input left
// after the probe is drained before returning so it does not leak to the shell.
func probeQuery(query string, isComplete func(string) bool) (string, bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", false
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", false
	}
	defer term.Restore(fd, oldState)
	defer drainInput(fd)

	os.Stdout.WriteString(query)

	deadline := time.Now().Add(500 * time.Millisecond)
	var resp strings.Builder
	buf := make([]byte, 128)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return resp.String(), false
		}
		pfd := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(pfd, int(remaining.Milliseconds())+1)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return resp.String(), false
		}
		if n == 0 {
			// timeout
			return resp.String(), false
		}
		rn, err := unix.Read(fd, buf)
		if rn > 0 {
			resp.Write(buf[:rn])
			if isComplete(resp.String()) {
				return resp.String(), true
			}
		}
		if err != nil || rn <= 0 {
			return resp.String(), false
		}
	}
}

// drainInput consumes any pending bytes on fd without blocking, so that
// unread terminal responses do not spill into the shell after exit.
func drainInput(fd int) {
	buf := make([]byte, 256)
	for {
		pfd := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(pfd, 20)
		if err != nil || n == 0 {
			return
		}
		rn, err := unix.Read(fd, buf)
		if rn <= 0 || err != nil {
			return
		}
	}
}
