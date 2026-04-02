package prettyprint

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ukaji3/mdview/internal/parser"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 1: Pretty Printer ラウンドトリップ
// Validates: Requirements 12.3
func TestProperty1_PrettyPrinterRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid Markdown text
		md := genMarkdown(t)

		source1 := []byte(md)
		ast1 := parser.Parse(source1)

		// PrettyPrint the AST
		pp := PrettyPrint(ast1, source1)

		// Parse the PrettyPrinted output
		source2 := []byte(pp)
		ast2 := parser.Parse(source2)

		// Compare the two ASTs for structural equivalence
		if err := compareAST(ast1, source1, ast2, source2, ""); err != nil {
			t.Fatalf("Round-trip failed.\nOriginal markdown:\n%s\nPrettyPrinted:\n%s\nAST mismatch: %s", md, pp, err)
		}
	})
}

// genMarkdown generates a random valid Markdown document from a set of block elements.
func genMarkdown(t *rapid.T) string {
	numBlocks := rapid.IntRange(1, 5).Draw(t, "numBlocks")
	var blocks []string
	for i := 0; i < numBlocks; i++ {
		blockType := rapid.IntRange(0, 8).Draw(t, fmt.Sprintf("blockType_%d", i))
		switch blockType {
		case 0:
			blocks = append(blocks, genHeading(t, i))
		case 1:
			blocks = append(blocks, genParagraph(t, i))
		case 2:
			blocks = append(blocks, genCodeBlock(t, i))
		case 3:
			blocks = append(blocks, genUnorderedList(t, i))
		case 4:
			blocks = append(blocks, genOrderedList(t, i))
		case 5:
			blocks = append(blocks, genBlockquote(t, i))
		case 6:
			blocks = append(blocks, genThematicBreak())
		case 7:
			blocks = append(blocks, genParagraphWithLink(t, i))
		case 8:
			blocks = append(blocks, genParagraphWithImage(t, i))
		}
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

func genPlainText(t *rapid.T, label string) string {
	return rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,30}`).Draw(t, label)
}

func genWord(t *rapid.T, label string) string {
	return rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{0,10}`).Draw(t, label)
}

func genHeading(t *rapid.T, idx int) string {
	level := rapid.IntRange(1, 6).Draw(t, fmt.Sprintf("headingLevel_%d", idx))
	text := genPlainText(t, fmt.Sprintf("headingText_%d", idx))
	return strings.Repeat("#", level) + " " + text
}

func genParagraph(t *rapid.T, idx int) string {
	text := genPlainText(t, fmt.Sprintf("paraText_%d", idx))
	// Optionally add inline formatting
	fmtType := rapid.IntRange(0, 4).Draw(t, fmt.Sprintf("paraFmt_%d", idx))
	switch fmtType {
	case 1:
		word := genWord(t, fmt.Sprintf("boldWord_%d", idx))
		return text + " **" + word + "**"
	case 2:
		word := genWord(t, fmt.Sprintf("italicWord_%d", idx))
		return text + " *" + word + "*"
	case 3:
		word := genWord(t, fmt.Sprintf("codeWord_%d", idx))
		return text + " `" + word + "`"
	case 4:
		word := genWord(t, fmt.Sprintf("strikeWord_%d", idx))
		return text + " ~~" + word + "~~"
	default:
		return text
	}
}

func genCodeBlock(t *rapid.T, idx int) string {
	langs := []string{"", "go", "python", "js", "rust"}
	langIdx := rapid.IntRange(0, len(langs)-1).Draw(t, fmt.Sprintf("codeLang_%d", idx))
	lang := langs[langIdx]
	numLines := rapid.IntRange(1, 3).Draw(t, fmt.Sprintf("codeLines_%d", idx))
	var lines []string
	for i := 0; i < numLines; i++ {
		line := genWord(t, fmt.Sprintf("codeLine_%d_%d", idx, i))
		lines = append(lines, line)
	}
	return "```" + lang + "\n" + strings.Join(lines, "\n") + "\n```"
}

func genUnorderedList(t *rapid.T, idx int) string {
	numItems := rapid.IntRange(1, 4).Draw(t, fmt.Sprintf("ulItems_%d", idx))
	var items []string
	for i := 0; i < numItems; i++ {
		text := genPlainText(t, fmt.Sprintf("ulItem_%d_%d", idx, i))
		items = append(items, "- "+text)
	}
	return strings.Join(items, "\n")
}

func genOrderedList(t *rapid.T, idx int) string {
	numItems := rapid.IntRange(1, 4).Draw(t, fmt.Sprintf("olItems_%d", idx))
	var items []string
	for i := 0; i < numItems; i++ {
		text := genPlainText(t, fmt.Sprintf("olItem_%d_%d", idx, i))
		items = append(items, fmt.Sprintf("%d. %s", i+1, text))
	}
	return strings.Join(items, "\n")
}

func genBlockquote(t *rapid.T, idx int) string {
	text := genPlainText(t, fmt.Sprintf("bqText_%d", idx))
	return "> " + text
}

func genThematicBreak() string {
	return "---"
}

func genParagraphWithLink(t *rapid.T, idx int) string {
	text := genWord(t, fmt.Sprintf("linkText_%d", idx))
	url := "https://" + genWord(t, fmt.Sprintf("linkUrl_%d", idx)) + ".com"
	return "[" + text + "](" + url + ")"
}

func genParagraphWithImage(t *rapid.T, idx int) string {
	alt := genWord(t, fmt.Sprintf("imgAlt_%d", idx))
	src := genWord(t, fmt.Sprintf("imgSrc_%d", idx)) + ".png"
	return "![" + alt + "](" + src + ")"
}

// compareAST recursively compares two ASTs for structural equivalence.
// It checks that node kinds match and text content is equivalent.
func compareAST(n1 ast.Node, src1 []byte, n2 ast.Node, src2 []byte, path string) error {
	if n1 == nil && n2 == nil {
		return nil
	}
	if n1 == nil || n2 == nil {
		return fmt.Errorf("at %s: one node is nil (n1=%v, n2=%v)", path, n1, n2)
	}

	// Compare node kinds
	if n1.Kind() != n2.Kind() {
		return fmt.Errorf("at %s: kind mismatch: %s vs %s", path, n1.Kind(), n2.Kind())
	}

	// Compare type-specific attributes
	if err := compareNodeAttributes(n1, src1, n2, src2, path); err != nil {
		return err
	}

	// Compare children
	c1 := n1.FirstChild()
	c2 := n2.FirstChild()
	childIdx := 0
	for c1 != nil || c2 != nil {
		childPath := fmt.Sprintf("%s/%s[%d]", path, nodeKindStr(c1, c2), childIdx)
		if err := compareAST(c1, src1, c2, src2, childPath); err != nil {
			return err
		}
		if c1 != nil {
			c1 = c1.NextSibling()
		}
		if c2 != nil {
			c2 = c2.NextSibling()
		}
		childIdx++
	}

	return nil
}

func nodeKindStr(n1, n2 ast.Node) string {
	if n1 != nil {
		return n1.Kind().String()
	}
	if n2 != nil {
		return n2.Kind().String()
	}
	return "nil"
}

// compareNodeAttributes compares type-specific attributes of two AST nodes.
func compareNodeAttributes(n1 ast.Node, src1 []byte, n2 ast.Node, src2 []byte, path string) error {
	switch v1 := n1.(type) {
	case *ast.Heading:
		v2, ok := n2.(*ast.Heading)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for Heading", path)
		}
		if v1.Level != v2.Level {
			return fmt.Errorf("at %s: heading level mismatch: %d vs %d", path, v1.Level, v2.Level)
		}

	case *ast.Text:
		v2, ok := n2.(*ast.Text)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for Text", path)
		}
		t1 := string(v1.Segment.Value(src1))
		t2 := string(v2.Segment.Value(src2))
		if t1 != t2 {
			return fmt.Errorf("at %s: text content mismatch: %q vs %q", path, t1, t2)
		}

	case *ast.Emphasis:
		v2, ok := n2.(*ast.Emphasis)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for Emphasis", path)
		}
		if v1.Level != v2.Level {
			return fmt.Errorf("at %s: emphasis level mismatch: %d vs %d", path, v1.Level, v2.Level)
		}

	case *ast.FencedCodeBlock:
		v2, ok := n2.(*ast.FencedCodeBlock)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for FencedCodeBlock", path)
		}
		lang1 := string(v1.Language(src1))
		lang2 := string(v2.Language(src2))
		if lang1 != lang2 {
			return fmt.Errorf("at %s: code block language mismatch: %q vs %q", path, lang1, lang2)
		}
		// Compare code content line by line
		content1 := codeBlockContent(v1, src1)
		content2 := codeBlockContent(v2, src2)
		if content1 != content2 {
			return fmt.Errorf("at %s: code block content mismatch:\n%q\nvs\n%q", path, content1, content2)
		}

	case *ast.List:
		v2, ok := n2.(*ast.List)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for List", path)
		}
		if v1.IsOrdered() != v2.IsOrdered() {
			return fmt.Errorf("at %s: list ordered mismatch: %v vs %v", path, v1.IsOrdered(), v2.IsOrdered())
		}
		if v1.IsOrdered() && v1.Start != v2.Start {
			return fmt.Errorf("at %s: ordered list start mismatch: %d vs %d", path, v1.Start, v2.Start)
		}

	case *ast.Link:
		v2, ok := n2.(*ast.Link)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for Link", path)
		}
		if string(v1.Destination) != string(v2.Destination) {
			return fmt.Errorf("at %s: link destination mismatch: %q vs %q", path, v1.Destination, v2.Destination)
		}

	case *ast.Image:
		v2, ok := n2.(*ast.Image)
		if !ok {
			return fmt.Errorf("at %s: type mismatch for Image", path)
		}
		if string(v1.Destination) != string(v2.Destination) {
			return fmt.Errorf("at %s: image destination mismatch: %q vs %q", path, v1.Destination, v2.Destination)
		}

	case *ast.CodeSpan:
		if _, ok := n2.(*ast.CodeSpan); !ok {
			return fmt.Errorf("at %s: type mismatch for CodeSpan", path)
		}
		// Content is compared via children (Text nodes)

	case *east.Strikethrough:
		if _, ok := n2.(*east.Strikethrough); !ok {
			return fmt.Errorf("at %s: type mismatch for Strikethrough", path)
		}

	case *east.Table:
		if _, ok := n2.(*east.Table); !ok {
			return fmt.Errorf("at %s: type mismatch for Table", path)
		}
		// Table structure is compared via children
	}

	return nil
}

func codeBlockContent(n *ast.FencedCodeBlock, source []byte) string {
	var buf strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		buf.Write(seg.Value(source))
	}
	return buf.String()
}

// Unit tests for PrettyPrint

func TestPrettyPrintHeading(t *testing.T) {
	for level := 1; level <= 6; level++ {
		prefix := strings.Repeat("#", level)
		md := prefix + " Hello World"
		source := []byte(md)
		node := parser.Parse(source)
		result := PrettyPrint(node, source)
		expected := prefix + " Hello World\n\n"
		if result != expected {
			t.Errorf("h%d: expected %q, got %q", level, expected, result)
		}
	}
}

func TestPrettyPrintParagraph(t *testing.T) {
	md := "Hello world"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	expected := "Hello world\n\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPrettyPrintEmphasis(t *testing.T) {
	md := "This is **bold** and *italic* text"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "**bold**") {
		t.Errorf("expected **bold** in result, got %q", result)
	}
	if !strings.Contains(result, "*italic*") {
		t.Errorf("expected *italic* in result, got %q", result)
	}
}

func TestPrettyPrintCodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hello\")\n```"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "```go\n") {
		t.Errorf("expected ```go in result, got %q", result)
	}
	if !strings.Contains(result, "fmt.Println") {
		t.Errorf("expected code content in result, got %q", result)
	}
	if !strings.Contains(result, "\n```\n") {
		t.Errorf("expected closing ``` in result, got %q", result)
	}
}

func TestPrettyPrintUnorderedList(t *testing.T) {
	md := "- Item 1\n- Item 2\n- Item 3"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "- Item 1\n") {
		t.Errorf("expected '- Item 1' in result, got %q", result)
	}
	if !strings.Contains(result, "- Item 2\n") {
		t.Errorf("expected '- Item 2' in result, got %q", result)
	}
}

func TestPrettyPrintOrderedList(t *testing.T) {
	md := "1. First\n2. Second\n3. Third"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "1. First\n") {
		t.Errorf("expected '1. First' in result, got %q", result)
	}
	if !strings.Contains(result, "2. Second\n") {
		t.Errorf("expected '2. Second' in result, got %q", result)
	}
}

func TestPrettyPrintBlockquote(t *testing.T) {
	md := "> This is a quote"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "> This is a quote\n") {
		t.Errorf("expected '> This is a quote' in result, got %q", result)
	}
}

func TestPrettyPrintThematicBreak(t *testing.T) {
	md := "Above\n\n---\n\nBelow"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "---\n") {
		t.Errorf("expected '---' in result, got %q", result)
	}
}

func TestPrettyPrintLink(t *testing.T) {
	md := "[Go](https://golang.org)"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "[Go](https://golang.org)") {
		t.Errorf("expected link in result, got %q", result)
	}
}

func TestPrettyPrintImage(t *testing.T) {
	md := "![alt text](image.png)"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "![alt text](image.png)") {
		t.Errorf("expected image in result, got %q", result)
	}
}

func TestPrettyPrintStrikethrough(t *testing.T) {
	md := "This is ~~deleted~~ text"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "~~deleted~~") {
		t.Errorf("expected ~~deleted~~ in result, got %q", result)
	}
}

func TestPrettyPrintCodeSpan(t *testing.T) {
	md := "Use `fmt.Println` here"
	source := []byte(md)
	node := parser.Parse(source)
	result := PrettyPrint(node, source)
	if !strings.Contains(result, "`fmt.Println`") {
		t.Errorf("expected `fmt.Println` in result, got %q", result)
	}
}
