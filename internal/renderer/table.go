package renderer

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// extractCellText extracts the plain text content from a table cell's children
// by reading source bytes from each child node's segments.
func extractCellText(cell *east.TableCell, source []byte) string {
	var buf strings.Builder
	for c := cell.FirstChild(); c != nil; c = c.NextSibling() {
		extractNodeText(&buf, c, source)
	}
	return buf.String()
}

// extractNodeText recursively extracts text from a node and its children.
func extractNodeText(buf *strings.Builder, n ast.Node, source []byte) {
	if t, ok := n.(*ast.Text); ok {
		buf.Write(t.Segment.Value(source))
		return
	}
	if s, ok := n.(*ast.String); ok {
		buf.Write(s.Value)
		return
	}
	if cs, ok := n.(*ast.CodeSpan); ok {
		for c := cs.FirstChild(); c != nil; c = c.NextSibling() {
			extractNodeText(buf, c, source)
		}
		return
	}
	// For other inline nodes (emphasis, strikethrough, etc.), recurse into children
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		extractNodeText(buf, c, source)
	}
}

// renderTable renders a complete table with box drawing characters.
// It collects all cell data first, then renders the table in one pass.
// Returns WalkSkipChildren so the AST walker does not visit table children.
//
// Layout:
//
//	┌──────┬──────┐
//	│ Head │ Head │
//	├──────┼──────┤
//	│ data │ data │
//	│ data │ data │
//	└──────┴──────┘
func renderTable(buf *strings.Builder, n *east.Table, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	// Collect header cells and data rows by walking the AST manually
	var headerCells []cellData
	var dataRows [][]cellData
	alignments := n.Alignments

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *east.TableHeader:
			// TableHeader contains TableCell children directly
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*east.TableCell); ok {
					headerCells = append(headerCells, cellData{
						text:      extractCellText(tc, source),
						alignment: tc.Alignment,
					})
				}
			}
		case *east.TableRow:
			var rowCells []cellData
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*east.TableCell); ok {
					rowCells = append(rowCells, cellData{
						text:      extractCellText(tc, source),
						alignment: tc.Alignment,
					})
				}
			}
			dataRows = append(dataRows, rowCells)
		}
	}

	numCols := len(headerCells)
	if numCols == 0 {
		return ast.WalkSkipChildren, nil
	}

	// Use table-level alignments as the authoritative source
	colAlignments := make([]east.Alignment, numCols)
	for i := 0; i < numCols; i++ {
		if i < len(alignments) {
			colAlignments[i] = alignments[i]
		} else {
			colAlignments[i] = east.AlignNone
		}
	}

	// Calculate column widths (max content width per column)
	colWidths := make([]int, numCols)
	for i, cell := range headerCells {
		w := displayWidth(cell.text)
		if w > colWidths[i] {
			colWidths[i] = w
		}
	}
	for _, row := range dataRows {
		for i, cell := range row {
			if i >= numCols {
				break
			}
			w := displayWidth(cell.text)
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	// Ensure minimum column width of 1
	for i := range colWidths {
		if colWidths[i] < 1 {
			colWidths[i] = 1
		}
	}

	borderColor := ""
	headerColor := ""
	if ctx.Theme != nil {
		borderColor = ctx.Theme.TableBorder
		headerColor = ctx.Theme.TableHeader
	}

	// Draw top border: ┌──────┬──────┐
	drawHorizontalBorder(buf, colWidths, "┌", "┬", "┐", borderColor)

	// Draw header row: │ Head │ Head │
	drawRow(buf, headerCells, colWidths, colAlignments, borderColor, headerColor+Bold, numCols)

	// Draw separator: ├──────┼──────┤
	drawHorizontalBorder(buf, colWidths, "├", "┼", "┤", borderColor)

	// Draw data rows
	for _, row := range dataRows {
		drawRow(buf, row, colWidths, colAlignments, borderColor, "", numCols)
	}

	// Draw bottom border: └──────┴──────┘
	drawHorizontalBorder(buf, colWidths, "└", "┴", "┘", borderColor)

	buf.WriteByte('\n')

	return ast.WalkSkipChildren, nil
}

// cellData holds the text content and alignment for a single table cell.
type cellData struct {
	text      string
	alignment east.Alignment
}

// drawHorizontalBorder draws a horizontal border line like ┌──────┬──────┐
func drawHorizontalBorder(buf *strings.Builder, colWidths []int, left, mid, right, borderColor string) {
	buf.WriteString(borderColor)
	buf.WriteString(left)
	dashWidth := runewidth.RuneWidth('─')
	for i, w := range colWidths {
		// Each cell has 1 space padding on each side, so width + 2
		fillWidth := w + 2
		dashCount := fillWidth / dashWidth
		buf.WriteString(strings.Repeat("─", dashCount))
		if i < len(colWidths)-1 {
			buf.WriteString(mid)
		}
	}
	buf.WriteString(right)
	buf.WriteString(Reset)
	buf.WriteByte('\n')
}

// drawRow draws a single table row like │ data │ data │
func drawRow(buf *strings.Builder, cells []cellData, colWidths []int, alignments []east.Alignment, borderColor, textStyle string, numCols int) {
	for i := 0; i < numCols; i++ {
		buf.WriteString(borderColor)
		buf.WriteString("│")
		buf.WriteString(Reset)

		text := ""
		if i < len(cells) {
			text = cells[i].text
		}

		w := displayWidth(text)
		colW := colWidths[i]
		padTotal := colW - w
		if padTotal < 0 {
			padTotal = 0
		}

		align := east.AlignNone
		if i < len(alignments) {
			align = alignments[i]
		}

		var leftPad, rightPad int
		switch align {
		case east.AlignCenter:
			leftPad = padTotal / 2
			rightPad = padTotal - leftPad
		case east.AlignRight:
			leftPad = padTotal
			rightPad = 0
		default: // AlignLeft, AlignNone
			leftPad = 0
			rightPad = padTotal
		}

		buf.WriteString(" ")
		if textStyle != "" {
			buf.WriteString(textStyle)
		}
		buf.WriteString(strings.Repeat(" ", leftPad))
		buf.WriteString(text)
		buf.WriteString(strings.Repeat(" ", rightPad))
		if textStyle != "" {
			buf.WriteString(Reset)
		}
		buf.WriteString(" ")
	}
	buf.WriteString(borderColor)
	buf.WriteString("│")
	buf.WriteString(Reset)
	buf.WriteByte('\n')
}
