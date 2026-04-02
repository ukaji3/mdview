package parser

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Parse はMarkdownテキストをgoldmarkのASTに変換する。
// テーブル拡張と取り消し線拡張を有効化したパーサーを使用する。
func Parse(source []byte) ast.Node {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
		),
	)
	reader := text.NewReader(source)
	return md.Parser().Parse(reader)
}
