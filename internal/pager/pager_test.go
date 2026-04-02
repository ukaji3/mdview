package pager

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 26: ページャーモード判定
// **Validates: Requirements 15.1, 15.2, 15.16, 15.17**
func TestProperty26_PagerModeDecision(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lineCount := rapid.IntRange(0, 500).Draw(t, "lineCount")
		termHeight := rapid.IntRange(1, 200).Draw(t, "termHeight")
		isTTY := rapid.Bool().Draw(t, "isTTY")
		noPager := rapid.Bool().Draw(t, "noPager")

		result := ShouldPage(lineCount, termHeight, isTTY, noPager)
		expected := lineCount > termHeight && isTTY && !noPager

		if result != expected {
			t.Fatalf("ShouldPage(%d, %d, %v, %v) = %v, want %v",
				lineCount, termHeight, isTTY, noPager, result, expected)
		}
	})
}

// Feature: markdown-terminal-renderer, Property 27: スクロール位置の境界制約
// **Validates: Requirements 15.4, 15.5, 15.6, 15.7, 15.8, 15.9**
func TestProperty27_ScrollPositionBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random content
		numLines := rapid.IntRange(1, 200).Draw(t, "numLines")
		var lines []string
		for i := 0; i < numLines; i++ {
			lines = append(lines, fmt.Sprintf("line %d", i))
		}
		content := strings.Join(lines, "\n")

		termHeight := rapid.IntRange(2, 100).Draw(t, "termHeight")
		termWidth := 80

		p := New(content, termWidth, termHeight)
		h := p.Height()
		n := p.LineCount()

		maxOffset := n - h
		if maxOffset < 0 {
			maxOffset = 0
		}

		// Generate a sequence of scroll operations
		numOps := rapid.IntRange(1, 50).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			op := rapid.IntRange(0, 5).Draw(t, fmt.Sprintf("op_%d", i))
			switch op {
			case 0: // ScrollDown(1) - j key
				p.ScrollDown(1)
			case 1: // ScrollUp(1) - k key
				p.ScrollUp(1)
			case 2: // ScrollDown(height) - space/pgdn
				p.ScrollDown(h)
			case 3: // ScrollUp(height) - b/pgup
				p.ScrollUp(h)
			case 4: // GoToTop - g
				p.GoToTop()
			case 5: // GoToBottom - G
				p.GoToBottom()
			}

			// Verify offset is always within bounds
			offset := p.Offset()
			if offset < 0 {
				t.Fatalf("offset %d < 0 after operation %d", offset, op)
			}
			if offset > maxOffset {
				t.Fatalf("offset %d > maxOffset %d (lines=%d, height=%d) after operation %d",
					offset, maxOffset, n, h, op)
			}
		}
	})
}

// Feature: markdown-terminal-renderer, Property 28: 検索一致箇所の正確性
// **Validates: Requirements 15.10, 15.11, 15.12**
func TestProperty28_SearchMatchAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random content lines
		numLines := rapid.IntRange(1, 100).Draw(t, "numLines")
		var lines []string
		for i := 0; i < numLines; i++ {
			line := rapid.StringMatching(`[a-zA-Z0-9 ]{1,40}`).Draw(t, fmt.Sprintf("line_%d", i))
			lines = append(lines, line)
		}
		content := strings.Join(lines, "\n")

		// Generate a search term (pick a short substring)
		searchTerm := rapid.StringMatching(`[a-zA-Z]{1,3}`).Draw(t, "searchTerm")

		termHeight := rapid.IntRange(5, 50).Draw(t, "termHeight")
		p := New(content, 80, termHeight)

		matches := p.Search(searchTerm)

		// Verify: every match line contains the search term
		for _, idx := range matches {
			if idx < 0 || idx >= len(p.lines) {
				t.Fatalf("match index %d out of range [0, %d)", idx, len(p.lines))
			}
			if !strings.Contains(p.lines[idx], searchTerm) {
				t.Fatalf("line %d (%q) does not contain search term %q",
					idx, p.lines[idx], searchTerm)
			}
		}

		// Verify: lines NOT in matches do NOT contain the search term
		matchSet := make(map[int]bool)
		for _, idx := range matches {
			matchSet[idx] = true
		}
		for i, line := range p.lines {
			if !matchSet[i] && strings.Contains(line, searchTerm) {
				t.Fatalf("line %d (%q) contains search term %q but is not in matches",
					i, line, searchTerm)
			}
		}

		// Verify n/N navigation stays within valid match indices
		if len(matches) > 0 {
			navOps := rapid.IntRange(1, 20).Draw(t, "navOps")
			for i := 0; i < navOps; i++ {
				if rapid.Bool().Draw(t, fmt.Sprintf("nav_dir_%d", i)) {
					p.NextMatch()
				} else {
					p.PrevMatch()
				}
				// matchIdx should be valid
				if p.matchIdx < 0 || p.matchIdx >= len(matches) {
					t.Fatalf("matchIdx %d out of range [0, %d) after navigation",
						p.matchIdx, len(matches))
				}
				// offset should be within bounds
				maxOffset := len(p.lines) - p.height
				if maxOffset < 0 {
					maxOffset = 0
				}
				if p.offset < 0 || p.offset > maxOffset {
					t.Fatalf("offset %d out of range [0, %d] after navigation",
						p.offset, maxOffset)
				}
			}
		}
	})
}

// Feature: markdown-terminal-renderer, Property 29: ステータスバー形式
// **Validates: Requirements 15.14**
func TestProperty29_StatusBarFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numLines := rapid.IntRange(1, 500).Draw(t, "numLines")
		var lines []string
		for i := 0; i < numLines; i++ {
			lines = append(lines, fmt.Sprintf("line %d", i))
		}
		content := strings.Join(lines, "\n")

		termHeight := rapid.IntRange(2, 100).Draw(t, "termHeight")
		p := New(content, 80, termHeight)

		// Optionally scroll to a random position
		if numLines > 1 {
			scrollOps := rapid.IntRange(0, 10).Draw(t, "scrollOps")
			for i := 0; i < scrollOps; i++ {
				op := rapid.IntRange(0, 3).Draw(t, fmt.Sprintf("scroll_op_%d", i))
				switch op {
				case 0:
					p.ScrollDown(1)
				case 1:
					p.ScrollUp(1)
				case 2:
					p.GoToTop()
				case 3:
					p.GoToBottom()
				}
			}
		}

		statusBar := p.StatusBar()
		expected := fmt.Sprintf("行 %d/%d", p.Offset()+1, p.LineCount())

		if statusBar != expected {
			t.Fatalf("StatusBar() = %q, want %q (offset=%d, lines=%d)",
				statusBar, expected, p.Offset(), p.LineCount())
		}
	})
}

// TestUpdateContent_ScrollPositionClamping verifies that UpdateContent clamps
// the scroll offset to a valid range when content shrinks.
func TestUpdateContent_ScrollPositionClamping(t *testing.T) {
	// Create pager with 100 lines, height 10 (9 visible)
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	content := strings.Join(lines, "\n")
	p := New(content, 80, 10)

	// Scroll to bottom
	p.GoToBottom()
	if p.Offset() == 0 {
		t.Fatal("expected non-zero offset after GoToBottom")
	}
	oldOffset := p.Offset()

	// Now update with much shorter content (5 lines)
	var shortLines []string
	for i := 0; i < 5; i++ {
		shortLines = append(shortLines, fmt.Sprintf("short %d", i))
	}
	shortContent := strings.Join(shortLines, "\n")
	p.UpdateContent(shortContent, 80, 10)

	// Offset should be clamped: maxOffset = 5 - 9 = 0 (clamped to 0)
	if p.Offset() > 0 {
		t.Fatalf("expected offset 0 after UpdateContent with short content, got %d (was %d)", p.Offset(), oldOffset)
	}
	if p.LineCount() != 5 {
		t.Fatalf("expected 5 lines, got %d", p.LineCount())
	}
}

// TestUpdateContent_PreservesOffset verifies that UpdateContent preserves
// the scroll offset when the new content is long enough.
func TestUpdateContent_PreservesOffset(t *testing.T) {
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	content := strings.Join(lines, "\n")
	p := New(content, 80, 20)

	// Scroll to offset 30
	p.ScrollDown(30)
	if p.Offset() != 30 {
		t.Fatalf("expected offset 30, got %d", p.Offset())
	}

	// Update with equally long content
	var newLines []string
	for i := 0; i < 100; i++ {
		newLines = append(newLines, fmt.Sprintf("new line %d", i))
	}
	newContent := strings.Join(newLines, "\n")
	p.UpdateContent(newContent, 80, 20)

	// Offset should be preserved
	if p.Offset() != 30 {
		t.Fatalf("expected offset 30 preserved, got %d", p.Offset())
	}
}

// TestUpdateContent_HeightClamp verifies that UpdateContent handles
// very small terminal heights correctly.
func TestUpdateContent_HeightClamp(t *testing.T) {
	content := "line1\nline2\nline3"
	p := New(content, 80, 10)

	// Update with height=1 (should clamp to height=1 after subtracting status bar)
	p.UpdateContent(content, 80, 1)
	if p.Height() != 1 {
		t.Fatalf("expected height 1, got %d", p.Height())
	}

	// Update with height=0 (should clamp to height=1)
	p.UpdateContent(content, 80, 0)
	if p.Height() != 1 {
		t.Fatalf("expected height 1 for zero termHeight, got %d", p.Height())
	}
}

// TestSetRenderFunc verifies that SetRenderFunc stores the callback.
func TestSetRenderFunc(t *testing.T) {
	p := New("hello", 80, 24)
	if p.renderFunc != nil {
		t.Fatal("expected nil renderFunc initially")
	}

	called := false
	p.SetRenderFunc(func(termWidth int) string {
		called = true
		return "re-rendered"
	})

	if p.renderFunc == nil {
		t.Fatal("expected non-nil renderFunc after SetRenderFunc")
	}

	result := p.renderFunc(80)
	if !called {
		t.Fatal("renderFunc was not called")
	}
	if result != "re-rendered" {
		t.Fatalf("expected 're-rendered', got %q", result)
	}
}

// TestSetFilePath verifies that SetFilePath stores the path.
func TestSetFilePath(t *testing.T) {
	p := New("hello", 80, 24)
	if p.filePath != "" {
		t.Fatal("expected empty filePath initially")
	}

	p.SetFilePath("/tmp/test.md")
	if p.filePath != "/tmp/test.md" {
		t.Fatalf("expected '/tmp/test.md', got %q", p.filePath)
	}
}

// TestUpdateContent_PropertyBased uses rapid to verify UpdateContent always
// produces valid state regardless of inputs.
func TestUpdateContent_PropertyBased(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Initial content
		numLines := rapid.IntRange(1, 200).Draw(t, "numLines")
		var lines []string
		for i := 0; i < numLines; i++ {
			lines = append(lines, fmt.Sprintf("line %d", i))
		}
		content := strings.Join(lines, "\n")
		termHeight := rapid.IntRange(2, 100).Draw(t, "termHeight")
		p := New(content, 80, termHeight)

		// Scroll to a random position
		scrollAmount := rapid.IntRange(0, numLines).Draw(t, "scrollAmount")
		p.ScrollDown(scrollAmount)

		// Generate new content with different length
		newNumLines := rapid.IntRange(1, 200).Draw(t, "newNumLines")
		var newLines []string
		for i := 0; i < newNumLines; i++ {
			newLines = append(newLines, fmt.Sprintf("new line %d", i))
		}
		newContent := strings.Join(newLines, "\n")
		newHeight := rapid.IntRange(1, 100).Draw(t, "newHeight")
		newWidth := rapid.IntRange(10, 200).Draw(t, "newWidth")

		p.UpdateContent(newContent, newWidth, newHeight)

		// Verify invariants
		if p.Offset() < 0 {
			t.Fatalf("offset %d < 0", p.Offset())
		}
		maxOffset := p.LineCount() - p.Height()
		if maxOffset < 0 {
			maxOffset = 0
		}
		if p.Offset() > maxOffset {
			t.Fatalf("offset %d > maxOffset %d", p.Offset(), maxOffset)
		}
		if p.Height() < 1 {
			t.Fatalf("height %d < 1", p.Height())
		}
	})
}
