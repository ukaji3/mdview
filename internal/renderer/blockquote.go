package renderer

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// blockquoteNestLevel counts the number of Blockquote ancestors to determine nesting depth.
// The node itself is not counted; only its parents are examined.
func blockquoteNestLevel(n ast.Node) int {
	level := 0
	for p := n.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.Blockquote); ok {
			level++
		}
	}
	return level
}

// renderBlockquote renders a blockquote with colored vertical bars and italic text.
// Each nesting level adds another "│ " prefix with the BlockquoteBar color from the theme.
// Inner Markdown elements are rendered normally by the AST walker.
//
// TODO: 複数行のブロック引用では、│プレフィックスが最初の行にのみ付与されます。
// 正しく修正するには、子要素をバッファにレンダリングしてから各行にプレフィックスを
// 付与する必要がありますが、AST walkerアプローチの大幅なリファクタリングが必要です。
func renderBlockquote(buf *strings.Builder, n *ast.Blockquote, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if entering {
		// Determine the total nesting level (including this blockquote)
		nestLevel := blockquoteNestLevel(n) + 1

		// Write colored vertical bars for each nesting level
		for i := 0; i < nestLevel; i++ {
			if ctx != nil && ctx.Theme != nil && ctx.Theme.BlockquoteBar != "" {
				buf.WriteString(ctx.Theme.BlockquoteBar)
			}
			buf.WriteString("│ ")
			buf.WriteString(Reset)
		}

		// Apply italic for blockquote text
		buf.WriteString(Italic)
	} else {
		// Reset italic and add trailing newline
		buf.WriteString(Reset)
		buf.WriteByte('\n')
	}
	return ast.WalkContinue, nil
}
