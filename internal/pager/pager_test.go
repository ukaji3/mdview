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
