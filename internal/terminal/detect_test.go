package terminal

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// helper to set env vars and restore them after the test.
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if existed {
			os.Setenv(key, old)
		}
	})
}

func TestDetectColorMode_NoColor(t *testing.T) {
	setEnv(t, "NO_COLOR", "1")
	if got := DetectColorMode(); got != ColorNone {
		t.Errorf("expected ColorNone when NO_COLOR is set, got %d", got)
	}
}

func TestDetectColorMode_NoColorEmpty(t *testing.T) {
	setEnv(t, "NO_COLOR", "")
	if got := DetectColorMode(); got != ColorNone {
		t.Errorf("expected ColorNone when NO_COLOR is set (even empty), got %d", got)
	}
}

func TestDetectColorMode_TrueColorViaCOLORTERM(t *testing.T) {
	unsetEnv(t, "NO_COLOR")
	setEnv(t, "COLORTERM", "truecolor")
	if got := DetectColorMode(); got != ColorTrue {
		t.Errorf("expected ColorTrue when COLORTERM=truecolor, got %d", got)
	}
}

func TestDetectColorMode_24bitViaCOLORTERM(t *testing.T) {
	unsetEnv(t, "NO_COLOR")
	setEnv(t, "COLORTERM", "24bit")
	if got := DetectColorMode(); got != ColorTrue {
		t.Errorf("expected ColorTrue when COLORTERM=24bit, got %d", got)
	}
}

func TestDetectColorMode_TrueColorViaTERM(t *testing.T) {
	unsetEnv(t, "NO_COLOR")
	unsetEnv(t, "COLORTERM")
	setEnv(t, "TERM", "xterm-truecolor")
	if got := DetectColorMode(); got != ColorTrue {
		t.Errorf("expected ColorTrue when TERM contains truecolor, got %d", got)
	}
}

func TestDetectColorMode_256Color(t *testing.T) {
	unsetEnv(t, "NO_COLOR")
	unsetEnv(t, "COLORTERM")
	setEnv(t, "TERM", "xterm-256color")
	if got := DetectColorMode(); got != Color256 {
		t.Errorf("expected Color256 for xterm-256color, got %d", got)
	}
}

func TestDetectColorMode_Fallback256(t *testing.T) {
	unsetEnv(t, "NO_COLOR")
	unsetEnv(t, "COLORTERM")
	setEnv(t, "TERM", "dumb")
	if got := DetectColorMode(); got != Color256 {
		t.Errorf("expected Color256 as fallback, got %d", got)
	}
}

func TestDetectWidth_MinimumGuarantee(t *testing.T) {
	w := DetectWidth()
	if w < 40 {
		t.Errorf("expected width >= 40, got %d", w)
	}
}

func TestDetectSixel_Stub(t *testing.T) {
	if DetectSixel() {
		t.Error("expected DetectSixel stub to return false")
	}
}

func TestDetectCapabilities_ReturnsNonNil(t *testing.T) {
	caps := DetectCapabilities()
	if caps == nil {
		t.Fatal("expected non-nil Capabilities")
	}
	if caps.Width < 40 {
		t.Errorf("expected width >= 40, got %d", caps.Width)
	}
	if caps.Theme == nil {
		t.Error("expected non-nil Theme")
	}
	if caps.CellHeight < 1 {
		t.Errorf("expected CellHeight >= 1, got %d", caps.CellHeight)
	}
}

func TestDetectCellHeight_Positive(t *testing.T) {
	h := DetectCellHeight()
	if h < 1 {
		t.Errorf("expected DetectCellHeight >= 1, got %d", h)
	}
}

// --- DetectImageProtocol tests ---

func TestDetectImageProtocol_iTerm(t *testing.T) {
	unsetEnv(t, "TERM")
	setEnv(t, "TERM_PROGRAM", "iTerm.app")
	if got := DetectImageProtocol(); got != ImageITerm2 {
		t.Errorf("expected ImageITerm2 for TERM_PROGRAM=iTerm.app, got %d", got)
	}
}

func TestDetectImageProtocol_WezTerm(t *testing.T) {
	unsetEnv(t, "TERM")
	setEnv(t, "TERM_PROGRAM", "WezTerm")
	if got := DetectImageProtocol(); got != ImageITerm2 {
		t.Errorf("expected ImageITerm2 for TERM_PROGRAM=WezTerm, got %d", got)
	}
}

func TestDetectImageProtocol_Mintty(t *testing.T) {
	unsetEnv(t, "TERM")
	setEnv(t, "TERM_PROGRAM", "mintty")
	if got := DetectImageProtocol(); got != ImageITerm2 {
		t.Errorf("expected ImageITerm2 for TERM_PROGRAM=mintty, got %d", got)
	}
}

func TestDetectImageProtocol_KittyTermProgram(t *testing.T) {
	unsetEnv(t, "TERM")
	setEnv(t, "TERM_PROGRAM", "kitty")
	if got := DetectImageProtocol(); got != ImageKitty {
		t.Errorf("expected ImageKitty for TERM_PROGRAM=kitty, got %d", got)
	}
}

func TestDetectImageProtocol_KittyTerm(t *testing.T) {
	unsetEnv(t, "TERM_PROGRAM")
	setEnv(t, "TERM", "xterm-kitty")
	if got := DetectImageProtocol(); got != ImageKitty {
		t.Errorf("expected ImageKitty for TERM=xterm-kitty, got %d", got)
	}
}

func TestDetectImageProtocol_GhosttyTermProgram(t *testing.T) {
	unsetEnv(t, "TERM")
	setEnv(t, "TERM_PROGRAM", "ghostty")
	if got := DetectImageProtocol(); got != ImageKitty {
		t.Errorf("expected ImageKitty for TERM_PROGRAM=ghostty, got %d", got)
	}
}

func TestDetectImageProtocol_GhosttyTerm(t *testing.T) {
	unsetEnv(t, "TERM_PROGRAM")
	setEnv(t, "TERM", "xterm-ghostty")
	if got := DetectImageProtocol(); got != ImageKitty {
		t.Errorf("expected ImageKitty for TERM=xterm-ghostty, got %d", got)
	}
}

func TestDetectImageProtocol_None(t *testing.T) {
	unsetEnv(t, "TERM_PROGRAM")
	setEnv(t, "TERM", "xterm-256color")
	if got := DetectImageProtocol(); got != ImageNone {
		t.Errorf("expected ImageNone for generic terminal, got %d", got)
	}
}

func TestDefaultTheme_NonEmpty(t *testing.T) {
	th := DefaultTheme()
	if th.H1Color == "" {
		t.Error("expected non-empty H1Color")
	}
	if th.ErrorColor == "" {
		t.Error("expected non-empty ErrorColor")
	}
}

// Feature: markdown-terminal-renderer, Property 17: カラーモード検出
// Validates: Requirements 11.2
// 任意のTERM/COLORTERM環境変数の組み合わせに対して、TrueColor対応値の場合はTrueColorモード、
// それ以外は256色モードが選択されることを検証
func TestProperty17_ColorModeDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		termVal := rapid.StringMatching(`[a-zA-Z0-9\-]{0,30}`).Draw(t, "TERM")
		colortermVal := rapid.StringMatching(`[a-zA-Z0-9\-]{0,20}`).Draw(t, "COLORTERM")

		// Save and restore environment
		oldNoColor, hadNoColor := os.LookupEnv("NO_COLOR")
		oldTerm, hadTerm := os.LookupEnv("TERM")
		oldColorterm, hadColorterm := os.LookupEnv("COLORTERM")
		defer func() {
			if hadNoColor {
				os.Setenv("NO_COLOR", oldNoColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}
			if hadTerm {
				os.Setenv("TERM", oldTerm)
			} else {
				os.Unsetenv("TERM")
			}
			if hadColorterm {
				os.Setenv("COLORTERM", oldColorterm)
			} else {
				os.Unsetenv("COLORTERM")
			}
		}()

		// Unset NO_COLOR so we only test TERM/COLORTERM logic
		os.Unsetenv("NO_COLOR")
		os.Setenv("TERM", termVal)
		os.Setenv("COLORTERM", colortermVal)

		got := DetectColorMode()

		// Determine expected result
		colortermLower := strings.ToLower(colortermVal)
		termLower := strings.ToLower(termVal)

		isTrueColor := colortermLower == "truecolor" || colortermLower == "24bit" ||
			strings.Contains(termLower, "truecolor") || strings.Contains(termLower, "24bit")

		if isTrueColor {
			if got != ColorTrue {
				t.Fatalf("expected ColorTrue for TERM=%q COLORTERM=%q, got %d", termVal, colortermVal, got)
			}
		} else {
			if got != Color256 {
				t.Fatalf("expected Color256 for TERM=%q COLORTERM=%q, got %d", termVal, colortermVal, got)
			}
		}
	})
}
