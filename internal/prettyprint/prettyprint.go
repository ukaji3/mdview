package prettyprint

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// prettyPrintChildren walks all children of a block-level node.
func prettyPrintChildren(buf *strings.Builder, node ast.Node, source []byte, listDepth int) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		prettyPrintNode(buf, child, source, listDepth)
	}
}

// prettyPrintInlineChildren walks all children of an inline-level node.
func prettyPrintInlineChildren(buf *strings.Builder, node ast.Node, source []byte) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		prettyPrintNode(buf, child, source, 0)
	}
}

// isInsideListItem checks whether the given node is a direct child of a ListItem.
func isInsideListItem(node ast.Node) bool {
	if node.Parent() == nil {
		return false
	}
	_, ok := node.Parent().(*ast.ListItem)
	return ok
}

// prettyPrintBlockquote renders a blockquote by prefixing each line with "> ".
func prettyPrintBlockquote(buf *strings.Builder, node *ast.Blockquote, source []byte, listDepth int) {
	// Render blockquote children into a temporary buffer
	var inner strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		prettyPrintNode(&inner, child, source, listDepth)
	}
	// Trim trailing newlines from inner content, then prefix each line with "> "
	content := strings.TrimRight(inner.String(), "\n")
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		buf.WriteString("> ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	// Add blank line after blockquote
	buf.WriteByte('\n')
}

// prettyPrintTable renders a GFM table as Markdown.
func prettyPrintTable(buf *strings.Builder, table *east.Table, source []byte) {
	// Collect all rows (header + body rows)
	var rows [][]string
	var alignments []east.Alignment

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *east.TableHeader:
			alignments = row.Alignments
			var cells []string
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				var cellBuf strings.Builder
				prettyPrintInlineChildren(&cellBuf, cell, source)
				cells = append(cells, cellBuf.String())
			}
			rows = append(rows, cells)
		case *east.TableRow:
			var cells []string
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				var cellBuf strings.Builder
				prettyPrintInlineChildren(&cellBuf, cell, source)
				cells = append(cells, cellBuf.String())
			}
			rows = append(rows, cells)
		}
	}

	if len(rows) == 0 {
		return
	}

	// Determine number of columns
	numCols := 0
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}

	// Pad rows to have equal number of columns
	for i := range rows {
		for len(rows[i]) < numCols {
			rows[i] = append(rows[i], "")
		}
	}

	// Compute column widths
	colWidths := make([]int, numCols)
	for _, row := range rows {
		for j, cell := range row {
			if len(cell) > colWidths[j] {
				colWidths[j] = len(cell)
			}
		}
	}
	// Ensure minimum width of 3 for separator
	for j := range colWidths {
		if colWidths[j] < 3 {
			colWidths[j] = 3
		}
	}

	// Write header row
	buf.WriteByte('|')
	for j, cell := range rows[0] {
		buf.WriteByte(' ')
		buf.WriteString(cell)
		buf.WriteString(strings.Repeat(" ", colWidths[j]-len(cell)))
		buf.WriteString(" |")
	}
	buf.WriteByte('\n')

	// Write separator row with alignment
	buf.WriteByte('|')
	for j := 0; j < numCols; j++ {
		align := east.AlignNone
		if j < len(alignments) {
			align = alignments[j]
		}
		switch align {
		case east.AlignLeft:
			buf.WriteString(" :")
			buf.WriteString(strings.Repeat("-", colWidths[j]-1))
			buf.WriteString(" |")
		case east.AlignRight:
			buf.WriteByte(' ')
			buf.WriteString(strings.Repeat("-", colWidths[j]-1))
			buf.WriteString(": |")
		case east.AlignCenter:
			buf.WriteString(" :")
			buf.WriteString(strings.Repeat("-", colWidths[j]-2))
			buf.WriteString(": |")
		default:
			buf.WriteByte(' ')
			buf.WriteString(strings.Repeat("-", colWidths[j]))
			buf.WriteString(" |")
		}
	}
	buf.WriteByte('\n')

	// Write body rows
	for i := 1; i < len(rows); i++ {
		buf.WriteByte('|')
		for j, cell := range rows[i] {
			buf.WriteByte(' ')
			buf.WriteString(cell)
			buf.WriteString(strings.Repeat(" ", colWidths[j]-len(cell)))
			buf.WriteString(" |")
		}
		buf.WriteByte('\n')
	}
}

// PrettyPrint walks the AST and regenerates valid Markdown text.
// The output is designed to satisfy the round-trip property:
// Parse(PrettyPrint(Parse(text))) yields the same AST as Parse(text).
func PrettyPrint(node ast.Node, source []byte) string {
	var buf strings.Builder
	prettyPrintNode(&buf, node, source, 0)
	return buf.String()
}

// prettyPrintNode recursively walks the AST and writes Markdown to buf.
// listDepth tracks the current list nesting level for indentation.
func prettyPrintNode(buf *strings.Builder, node ast.Node, source []byte, listDepth int) {
	switch n := node.(type) {
	case *ast.Document:
		prettyPrintChildren(buf, n, source, listDepth)

	case *ast.Heading:
		buf.WriteString(strings.Repeat("#", n.Level))
		buf.WriteByte(' ')
		prettyPrintInlineChildren(buf, n, source)
		buf.WriteByte('\n')
		// Add blank line after heading
		buf.WriteByte('\n')

	case *ast.Paragraph:
		// Check if inside a blockquote — prefix handled by blockquote
		prettyPrintInlineChildren(buf, n, source)
		buf.WriteByte('\n')
		// Add blank line after paragraph unless inside a list item
		if !isInsideListItem(n) {
			buf.WriteByte('\n')
		}

	case *ast.Text:
		buf.Write(n.Segment.Value(source))
		if n.HardLineBreak() {
			buf.WriteString("  \n")
		} else if n.SoftLineBreak() {
			buf.WriteByte('\n')
		}

	case *ast.String:
		buf.Write(n.Value)

	case *ast.Emphasis:
		marker := "*"
		if n.Level == 2 {
			marker = "**"
		}
		buf.WriteString(marker)
		prettyPrintInlineChildren(buf, n, source)
		buf.WriteString(marker)

	case *ast.CodeSpan:
		buf.WriteByte('`')
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value(source))
			}
		}
		buf.WriteByte('`')

	case *ast.FencedCodeBlock:
		lang := string(n.Language(source))
		buf.WriteString("```")
		buf.WriteString(lang)
		buf.WriteByte('\n')
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			buf.Write(seg.Value(source))
		}
		buf.WriteString("```\n")
		buf.WriteByte('\n')

	case *ast.CodeBlock:
		// Indented code block — output as fenced for round-trip consistency
		buf.WriteString("```\n")
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			buf.Write(seg.Value(source))
		}
		buf.WriteString("```\n")
		buf.WriteByte('\n')

	case *ast.List:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			prettyPrintNode(buf, child, source, listDepth)
			// For loose lists, add a blank line after each item
			if !n.IsTight && child.NextSibling() != nil {
				buf.WriteByte('\n')
			}
		}
		// Add blank line after top-level list
		if listDepth == 0 {
			buf.WriteByte('\n')
		}

	case *ast.ListItem:
		parentList, ok := n.Parent().(*ast.List)
		if !ok {
			prettyPrintChildren(buf, n, source, listDepth)
			return
		}

		indent := strings.Repeat("    ", listDepth)

		var marker string
		if parentList.IsOrdered() {
			idx := 0
			for c := parentList.FirstChild(); c != nil; c = c.NextSibling() {
				if c == n {
					break
				}
				idx++
			}
			num := parentList.Start + idx
			marker = fmt.Sprintf("%d. ", num)
		} else {
			marker = "- "
		}

		buf.WriteString(indent)
		buf.WriteString(marker)

		// Render children of the list item
		first := true
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			switch c := child.(type) {
			case *ast.Paragraph:
				if !first {
					// Continuation paragraph in a list item
					buf.WriteString(indent)
					buf.WriteString(strings.Repeat(" ", len(marker)))
				}
				prettyPrintInlineChildren(buf, c, source)
				buf.WriteByte('\n')
				first = false
			case *ast.List:
				prettyPrintNode(buf, c, source, listDepth+1)
				first = false
			default:
				prettyPrintNode(buf, child, source, listDepth+1)
				first = false
			}
		}

	case *ast.Blockquote:
		prettyPrintBlockquote(buf, n, source, listDepth)

	case *ast.ThematicBreak:
		buf.WriteString("---\n")
		buf.WriteByte('\n')

	case *ast.Link:
		buf.WriteByte('[')
		prettyPrintInlineChildren(buf, n, source)
		buf.WriteString("](")
		buf.Write(n.Destination)
		if len(n.Title) > 0 {
			buf.WriteString(` "`)
			buf.Write(n.Title)
			buf.WriteByte('"')
		}
		buf.WriteByte(')')

	case *ast.Image:
		buf.WriteString("![")
		buf.Write(n.Text(source))
		buf.WriteString("](")
		buf.Write(n.Destination)
		if len(n.Title) > 0 {
			buf.WriteString(` "`)
			buf.Write(n.Title)
			buf.WriteByte('"')
		}
		buf.WriteByte(')')

	case *ast.AutoLink:
		buf.WriteByte('<')
		buf.Write(n.URL(n.Label(nil)))
		buf.WriteByte('>')

	case *ast.RawHTML:
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			buf.Write(seg.Value(source))
		}

	case *ast.HTMLBlock:
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			buf.Write(seg.Value(source))
		}
		buf.WriteByte('\n')

	case *ast.TextBlock:
		prettyPrintInlineChildren(buf, n, source)
		buf.WriteByte('\n')

	// Extension types
	case *east.Table:
		prettyPrintTable(buf, n, source)
		buf.WriteByte('\n')

	case *east.Strikethrough:
		buf.WriteString("~~")
		prettyPrintInlineChildren(buf, n, source)
		buf.WriteString("~~")

	default:
		// For unknown node types, try to walk children
		prettyPrintChildren(buf, node, source, listDepth)
	}
}
