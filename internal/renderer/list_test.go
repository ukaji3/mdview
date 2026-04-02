package renderer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 6: リストのネストレベルに応じた表示
// Validates: Requirements 6.1, 6.3, 6.4
func TestProperty6_UnorderedListNestLevelDisplay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random nest level (0-3)
		nestLevel := rapid.IntRange(0, 3).Draw(t, "nestLevel")

		// 2. Generate random number of items (1-10)
		numItems := rapid.IntRange(1, 10).Draw(t, "numItems")

		// 3. Generate random item texts with unique prefix to distinguish from parent items
		var items []string
		for i := 0; i < numItems; i++ {
			item := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,20}`).Draw(t, fmt.Sprintf("item%d", i))
			// Add unique prefix so items are distinguishable from parent placeholders
			item = fmt.Sprintf("ITEM%d_%s", i, item)
			items = append(items, item)
		}

		// 4. Construct markdown with appropriate nesting indentation
		// Each nest level requires 2 spaces of indentation in markdown
		var mdBuilder strings.Builder
		// For nestLevel > 0, we need parent list items at each level.
		for level := 0; level < nestLevel; level++ {
			indent := strings.Repeat("  ", level)
			mdBuilder.WriteString(indent + "- PARENT" + fmt.Sprintf("%d", level) + "\n")
		}
		// Now add the actual items at the target nest level
		indent := strings.Repeat("  ", nestLevel)
		for _, item := range items {
			mdBuilder.WriteString(indent + "- " + item + "\n")
		}

		markdown := mdBuilder.String()

		// 5. Parse and render
		source := []byte(markdown)
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    80,
			ColorMode:    terminal.ColorTrue,
			SixelSupport: false,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 6a. Verify correct Unicode bullet for the nest level
		expectedBullet := bulletForLevel(nestLevel)

		// 6b. Verify indent is proportional to level and correct bullet is used
		// Each item line at nestLevel should start with (nestLevel * 2) spaces + bullet
		expectedIndent := strings.Repeat("  ", nestLevel)
		lines := strings.Split(result, "\n")
		foundItemAtLevel := false
		for _, line := range lines {
			// Match only our actual items (not parent placeholders)
			if strings.Contains(line, items[0]) {
				foundItemAtLevel = true
				// Verify the line contains the expected bullet
				if !strings.Contains(line, expectedBullet) {
					t.Fatalf("expected bullet %q for nest level %d in line %q",
						expectedBullet, nestLevel, line)
				}
				// Verify the line starts with the expected indentation
				if nestLevel > 0 && !strings.HasPrefix(line, expectedIndent) {
					t.Fatalf("expected line to start with %d spaces indent for level %d, got %q",
						nestLevel*2, nestLevel, line)
				}
				break
			}
		}
		if !foundItemAtLevel {
			t.Fatalf("could not find item %q at nest level %d in output %q",
				items[0], nestLevel, result)
		}

		// 6c. Verify all items appear in the output
		for _, item := range items {
			if !strings.Contains(result, item) {
				t.Fatalf("expected item text %q in output, got %q", item, result)
			}
		}
	})
}

// Feature: markdown-terminal-renderer, Property 7: 順序付きリストの番号付け
// Validates: Requirements 6.2
func TestProperty7_OrderedListNumbering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random number of items (1-20)
		numItems := rapid.IntRange(1, 20).Draw(t, "numItems")

		// 2. Generate random item texts
		var items []string
		for i := 0; i < numItems; i++ {
			item := rapid.StringMatching(`[A-Za-z0-9]{1,20}`).Draw(t, fmt.Sprintf("item%d", i))
			items = append(items, item)
		}

		// 3. Construct ordered list markdown
		var mdBuilder strings.Builder
		for i, item := range items {
			mdBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}
		markdown := mdBuilder.String()

		// 4. Parse and render
		source := []byte(markdown)
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    80,
			ColorMode:    terminal.ColorTrue,
			SixelSupport: false,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 5a. Verify sequential numbers appear (1, 2, 3, ...)
		for i := 1; i <= numItems; i++ {
			numStr := fmt.Sprintf("%d.", i)
			if !strings.Contains(result, numStr) {
				t.Fatalf("expected number %q in output for %d items, got %q",
					numStr, numItems, result)
			}
		}

		// 5b. Verify numbers are right-aligned (same width)
		// The width of the number field should be consistent.
		// For example, with 11 items: " 1. " and "11. " (2-digit width)
		maxNumWidth := len(fmt.Sprintf("%d", numItems))

		// Check that each number in the output is formatted with the correct width
		lines := strings.Split(result, "\n")
		for i := 1; i <= numItems; i++ {
			// Expected format: right-aligned number with consistent width
			expectedNum := fmt.Sprintf("%*d.", maxNumWidth, i)
			found := false
			for _, line := range lines {
				if strings.Contains(line, expectedNum) && strings.Contains(line, items[i-1]) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected right-aligned number %q with item %q in output, got %q",
					expectedNum, items[i-1], result)
			}
		}

		// 5c. Verify all items appear in the output
		for _, item := range items {
			if !strings.Contains(result, item) {
				t.Fatalf("expected item text %q in output, got %q", item, result)
			}
		}
	})
}
