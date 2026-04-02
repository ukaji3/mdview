package parser

import (
	"testing"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

func TestParse_ReturnsDocument(t *testing.T) {
	source := []byte("# Hello")
	node := Parse(source)
	if node.Kind() != ast.KindDocument {
		t.Fatalf("expected Document node, got %v", node.Kind())
	}
}

func TestParse_Heading(t *testing.T) {
	source := []byte("# Heading 1\n\n## Heading 2\n")
	node := Parse(source)

	child := node.FirstChild()
	if child == nil || child.Kind() != ast.KindHeading {
		t.Fatal("expected first child to be a Heading")
	}
	h := child.(*ast.Heading)
	if h.Level != 1 {
		t.Fatalf("expected heading level 1, got %d", h.Level)
	}

	child = child.NextSibling()
	if child == nil || child.Kind() != ast.KindHeading {
		t.Fatal("expected second child to be a Heading")
	}
	h = child.(*ast.Heading)
	if h.Level != 2 {
		t.Fatalf("expected heading level 2, got %d", h.Level)
	}
}

func TestParse_Paragraph(t *testing.T) {
	source := []byte("Hello world\n")
	node := Parse(source)

	child := node.FirstChild()
	if child == nil || child.Kind() != ast.KindParagraph {
		t.Fatal("expected first child to be a Paragraph")
	}
}

func TestParse_Table(t *testing.T) {
	source := []byte("| A | B |\n|---|---|\n| 1 | 2 |\n")
	node := Parse(source)

	child := node.FirstChild()
	if child == nil || child.Kind() != east.KindTable {
		t.Fatalf("expected first child to be a Table, got %v", child.Kind())
	}
}

func TestParse_Strikethrough(t *testing.T) {
	source := []byte("~~deleted~~\n")
	node := Parse(source)

	// Document -> Paragraph -> Strikethrough
	para := node.FirstChild()
	if para == nil || para.Kind() != ast.KindParagraph {
		t.Fatal("expected Paragraph")
	}
	child := para.FirstChild()
	if child == nil || child.Kind() != east.KindStrikethrough {
		t.Fatalf("expected Strikethrough, got %v", child.Kind())
	}
}

func TestParse_FencedCodeBlock(t *testing.T) {
	source := []byte("```go\nfmt.Println(\"hello\")\n```\n")
	node := Parse(source)

	child := node.FirstChild()
	if child == nil || child.Kind() != ast.KindFencedCodeBlock {
		t.Fatalf("expected FencedCodeBlock, got %v", child.Kind())
	}
	cb := child.(*ast.FencedCodeBlock)
	lang := cb.Language(source)
	if string(lang) != "go" {
		t.Fatalf("expected language 'go', got '%s'", string(lang))
	}
}

func TestParse_List(t *testing.T) {
	source := []byte("- item 1\n- item 2\n")
	node := Parse(source)

	child := node.FirstChild()
	if child == nil || child.Kind() != ast.KindList {
		t.Fatalf("expected List, got %v", child.Kind())
	}
}

func TestParse_Blockquote(t *testing.T) {
	source := []byte("> quoted text\n")
	node := Parse(source)

	child := node.FirstChild()
	if child == nil || child.Kind() != ast.KindBlockquote {
		t.Fatalf("expected Blockquote, got %v", child.Kind())
	}
}

func TestParse_EmptyInput(t *testing.T) {
	source := []byte("")
	node := Parse(source)
	if node.Kind() != ast.KindDocument {
		t.Fatalf("expected Document node for empty input, got %v", node.Kind())
	}
	if node.HasChildren() {
		t.Fatal("expected no children for empty input")
	}
}
