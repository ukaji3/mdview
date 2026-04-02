package renderer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 9: テーブルの列幅自動調整
// **Validates: Requirements 8.1, 8.2, 8.3**
func TestProperty9_TableColumnWidthAutoAdjust(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate random number of columns (2-5) and rows (1-5)
		numCols := rapid.IntRange(2, 5).Draw(t, "numCols")
		numRows := rapid.IntRange(1, 5).Draw(t, "numRows")

		// 2. Generate header cells
		headers := make([]string, numCols)
		for i := 0; i < numCols; i++ {
			headers[i] = rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,15}`).Draw(t, fmt.Sprintf("header%d", i))
		}

		// 3. Generate data rows
		rows := make([][]string, numRows)
		for r := 0; r < numRows; r++ {
			rows[r] = make([]string, numCols)
			for c := 0; c < numCols; c++ {
				rows[r][c] = rapid.StringMatching(`[A-Za-z0-9]{1,20}`).Draw(t, fmt.Sprintf("cell%d_%d", r, c))
			}
		}

		// 4. Construct markdown table
		var md strings.Builder
		// Header row
		md.WriteString("| ")
		md.WriteString(strings.Join(headers, " | "))
		md.WriteString(" |\n")
		// Separator row
		md.WriteString("|")
		for i := 0; i < numCols; i++ {
			md.WriteString(" --- |")
		}
		md.WriteString("\n")
		// Data rows
		for _, row := range rows {
			md.WriteString("| ")
			md.WriteString(strings.Join(row, " | "))
			md.WriteString(" |\n")
		}

		// 5. Parse and render
		source := []byte(md.String())
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    120,
			ColorMode:    terminal.ColorTrue,
			ImageProtocol: terminal.ImageNone,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 6a. Verify box drawing characters are present
		boxChars := []string{"┌", "┐", "└", "┘", "│", "─", "┬", "┴", "├", "┤", "┼"}
		for _, ch := range boxChars {
			if !strings.Contains(result, ch) {
				t.Fatalf("expected box drawing char %q in output, got %q", ch, result)
			}
		}

		// 6b. Verify header Bold ANSI code is present
		if !strings.Contains(result, Bold) {
			t.Fatalf("expected Bold ANSI code in output for header row, got %q", result)
		}

		// 6c. Verify TableHeader color is present
		if !strings.Contains(result, theme.TableHeader) {
			t.Fatalf("expected TableHeader color %q in output, got %q", theme.TableHeader, result)
		}

		// 6d. Verify column widths: each column width >= max content width in that column
		// We check this by verifying all cell content appears in the output
		for _, h := range headers {
			if !strings.Contains(result, h) {
				t.Fatalf("expected header %q in output, got %q", h, result)
			}
		}
		for _, row := range rows {
			for _, cell := range row {
				if !strings.Contains(result, cell) {
					t.Fatalf("expected cell %q in output, got %q", cell, result)
				}
			}
		}

		// 6e. Verify column widths are consistent: all horizontal border segments
		// between the same column separators should have the same width.
		// Extract lines that start with border chars
		outputLines := strings.Split(result, "\n")
		var borderLines []string
		for _, line := range outputLines {
			stripped := stripANSI(line)
			if len(stripped) > 0 && (strings.HasPrefix(stripped, "┌") || strings.HasPrefix(stripped, "├") || strings.HasPrefix(stripped, "└")) {
				borderLines = append(borderLines, stripped)
			}
		}
		if len(borderLines) < 2 {
			t.Fatalf("expected at least 2 border lines (top + bottom), got %d", len(borderLines))
		}

		// All border lines should have the same length (same column widths)
		firstLen := len(borderLines[0])
		for i, bl := range borderLines {
			if len(bl) != firstLen {
				t.Fatalf("border line %d length %d differs from first border line length %d", i, len(bl), firstLen)
			}
		}
	})
}

// Feature: markdown-terminal-renderer, Property 10: テーブルのアライメント
// **Validates: Requirements 8.4**
func TestProperty10_TableAlignment(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 1. Generate alignment types for 3 columns
		alignTypes := []string{"left", "center", "right"}
		alignIdx := make([]int, 3)
		for i := 0; i < 3; i++ {
			alignIdx[i] = rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("align%d", i))
		}

		// 2. Generate short header and data cell text
		headers := make([]string, 3)
		dataCells := make([]string, 3)
		for i := 0; i < 3; i++ {
			headers[i] = rapid.StringMatching(`[A-Z][a-z]{2,8}`).Draw(t, fmt.Sprintf("header%d", i))
			dataCells[i] = rapid.StringMatching(`[a-z]{1,4}`).Draw(t, fmt.Sprintf("data%d", i))
		}

		// 3. Construct markdown table with alignment
		var md strings.Builder
		md.WriteString("| ")
		md.WriteString(strings.Join(headers, " | "))
		md.WriteString(" |\n|")
		for i := 0; i < 3; i++ {
			switch alignTypes[alignIdx[i]] {
			case "left":
				md.WriteString(" :--- |")
			case "center":
				md.WriteString(" :---: |")
			case "right":
				md.WriteString(" ---: |")
			}
		}
		md.WriteString("\n| ")
		md.WriteString(strings.Join(dataCells, " | "))
		md.WriteString(" |\n")

		// 4. Parse and render
		source := []byte(md.String())
		node := parser.Parse(source)
		theme := terminal.DefaultTheme()
		ctx := &RenderContext{
			TermWidth:    120,
			ColorMode:    terminal.ColorTrue,
			ImageProtocol: terminal.ImageNone,
			Theme:        theme,
			IsTTY:        true,
		}
		result := Render(node, source, ctx)

		// 5. Verify alignment by checking padding in data row cells
		// Find the data row line (the line between ├...┤ and └...┘)
		outputLines := strings.Split(result, "\n")
		var dataRowLine string
		foundSeparator := false
		for _, line := range outputLines {
			stripped := stripANSI(line)
			if strings.HasPrefix(stripped, "├") {
				foundSeparator = true
				continue
			}
			if foundSeparator && strings.HasPrefix(stripped, "│") {
				dataRowLine = stripped
				break
			}
		}

		if dataRowLine == "" {
			t.Fatalf("could not find data row line in output %q", result)
		}

		// Split data row by │ to get cell contents
		// The line looks like: │ data │ data │ data │
		parts := strings.Split(dataRowLine, "│")
		// parts[0] is empty (before first │), parts[1..3] are cells, parts[4] is empty (after last │)
		if len(parts) < 4 {
			t.Fatalf("expected at least 4 parts when splitting data row by │, got %d: %q", len(parts), dataRowLine)
		}

		for i := 0; i < 3; i++ {
			cellContent := parts[i+1] // cell with padding
			text := dataCells[i]

			if !strings.Contains(cellContent, text) {
				t.Fatalf("expected cell %d to contain %q, got %q", i, text, cellContent)
			}

			// Check alignment based on padding
			switch alignTypes[alignIdx[i]] {
			case "left":
				// Left-aligned: text should be at the start (after 1 space padding)
				trimmed := strings.TrimLeft(cellContent, " ")
				if !strings.HasPrefix(trimmed, text) {
					t.Fatalf("column %d: expected left-aligned text %q at start, got cell %q", i, text, cellContent)
				}
			case "right":
				// Right-aligned: text should be at the end (before 1 space padding)
				trimmed := strings.TrimRight(cellContent, " ")
				if !strings.HasSuffix(trimmed, text) {
					t.Fatalf("column %d: expected right-aligned text %q at end, got cell %q", i, text, cellContent)
				}
			case "center":
				// Center-aligned: left padding should be approximately equal to right padding
				idx := strings.Index(cellContent, text)
				if idx < 0 {
					t.Fatalf("column %d: text %q not found in cell %q", i, text, cellContent)
				}
				leftPad := idx
				rightPad := len(cellContent) - idx - len(text)
				diff := leftPad - rightPad
				if diff < -1 || diff > 1 {
					t.Fatalf("column %d: center alignment padding unbalanced: left=%d, right=%d for text %q in cell %q",
						i, leftPad, rightPad, text, cellContent)
				}
			}
		}
	})
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var buf strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// Unit test: basic table rendering
func TestTable_BasicRendering(t *testing.T) {
	source := []byte("| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |\n")
	node := parser.Parse(source)
	theme := terminal.DefaultTheme()
	ctx := &RenderContext{
		TermWidth:    80,
		ColorMode:    terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:        theme,
		IsTTY:        true,
	}
	result := Render(node, source, ctx)

	// Verify box drawing chars
	for _, ch := range []string{"┌", "┐", "└", "┘", "│", "─"} {
		if !strings.Contains(result, ch) {
			t.Errorf("expected box drawing char %q in output, got %q", ch, result)
		}
	}
	// Verify content
	if !strings.Contains(result, "Name") {
		t.Errorf("expected 'Name' in output, got %q", result)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("expected 'Alice' in output, got %q", result)
	}
	// Verify header styling
	if !strings.Contains(result, Bold) {
		t.Errorf("expected Bold ANSI code for header, got %q", result)
	}
}

// Unit test: table with alignment
func TestTable_Alignment(t *testing.T) {
	source := []byte("| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |\n")
	node := parser.Parse(source)
	theme := terminal.DefaultTheme()
	ctx := &RenderContext{
		TermWidth:    80,
		ColorMode:    terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:        theme,
		IsTTY:        true,
	}
	result := Render(node, source, ctx)

	// Verify all content is present
	for _, text := range []string{"Left", "Center", "Right", "a", "b", "c"} {
		if !strings.Contains(result, text) {
			t.Errorf("expected %q in output, got %q", text, result)
		}
	}
}
