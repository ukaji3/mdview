package renderer

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Unordered list bullets by nest level.
var unorderedBullets = []string{"•", "◦", "▪"}

// bulletForLevel returns the Unicode bullet character for the given nesting level.
// Level 0 = •, Level 1 = ◦, Level 2+ = ▪
func bulletForLevel(level int) string {
	if level < 0 {
		level = 0
	}
	if level >= len(unorderedBullets) {
		return unorderedBullets[len(unorderedBullets)-1]
	}
	return unorderedBullets[level]
}

// listNestLevel counts the number of ancestor List nodes to determine nesting depth.
// The node itself is not counted; only its parents are examined.
func listNestLevel(n ast.Node) int {
	level := 0
	for p := n.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.List); ok {
			level++
		}
	}
	return level
}

// indentForLevel returns the indentation string for a given nesting level.
// Each level adds 2 spaces of indentation.
func indentForLevel(level int) string {
	return strings.Repeat("  ", level)
}

// renderList handles entering/leaving a List node.
// On leaving, it appends a newline to separate the list from following content.
func renderList(buf *strings.Builder, n *ast.List, entering bool) (ast.WalkStatus, error) {
	if !entering {
		// Only add trailing newline for top-level lists
		level := listNestLevel(n)
		if level == 0 {
			buf.WriteByte('\n')
		}
	}
	return ast.WalkContinue, nil
}

// renderListItem renders a single list item with appropriate bullet/number and indentation.
// It collects the text content of the item's inline children and writes it with proper formatting.
func renderListItem(buf *strings.Builder, n *ast.ListItem, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	parentList, ok := n.Parent().(*ast.List)
	if !ok {
		return ast.WalkContinue, nil
	}

	// The nest level is determined by counting List ancestors of the ListItem.
	// Since the ListItem's direct parent is a List, we count List ancestors of that List.
	level := listNestLevel(n)
	// listNestLevel counts List ancestors of n. Since n is a ListItem, its parent is a List.
	// We want the nesting depth of the List itself, which is level - 1 for the ListItem
	// (because the immediate parent List is counted).
	// Actually, listNestLevel counts List nodes among ancestors of n.
	// For a top-level ListItem: parent=List, grandparent=Document -> level=1
	// For a nested ListItem: parent=List, grandparent=ListItem, great-grandparent=List, ... -> level=2
	// So the visual nesting level is level - 1.
	nestLevel := level - 1
	if nestLevel < 0 {
		nestLevel = 0
	}

	indent := indentForLevel(nestLevel)

	// Build the marker (bullet or number)
	var marker string
	if parentList.IsOrdered() {
		// Calculate the item index within the parent list
		idx := 0
		for c := parentList.FirstChild(); c != nil; c = c.NextSibling() {
			if c == n {
				break
			}
			idx++
		}
		start := parentList.Start
		num := start + idx

		// Count total items for digit alignment
		totalItems := 0
		for c := parentList.FirstChild(); c != nil; c = c.NextSibling() {
			totalItems++
		}
		lastNum := start + totalItems - 1
		width := len(fmt.Sprintf("%d", lastNum))

		marker = fmt.Sprintf("%*d. ", width, num)
	} else {
		marker = bulletForLevel(nestLevel) + " "
	}

	buf.WriteString(indent)
	buf.WriteString(marker)

	// Render children inline. We walk the children manually to handle
	// nested lists vs inline content properly.
	// We let the AST walker handle children normally by returning WalkContinue.
	// The paragraph inside a ListItem will handle text output.
	return ast.WalkContinue, nil
}
