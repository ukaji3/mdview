package renderer

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/ukaji3/mdview/internal/terminal"
	"github.com/yuin/goldmark/ast"
)

// Regex patterns for HTML tag parsing.
var (
	// htmlTagRe matches opening/closing HTML tags with optional attributes.
	htmlTagRe = regexp.MustCompile(`<(/?)(\w+)([^>]*)>`)
	// attrRe extracts key="value" or key='value' attribute pairs.
	attrRe = regexp.MustCompile(`(\w+)=["']([^"']*)["']`)
	// selfClosingBrRe matches <br>, <br/>, <br />.
	selfClosingBrRe = regexp.MustCompile(`(?i)<br\s*/?>`)
	// selfClosingHrRe matches <hr>, <hr/>, <hr />.
	selfClosingHrRe = regexp.MustCompile(`(?i)<hr\s*/?>`)
	// imgTagRe matches <img ...> tags (self-closing).
	imgTagRe = regexp.MustCompile(`(?i)<img\s+([^>]*)>`)
	// commentRe matches HTML comments <!-- ... -->.
	commentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)
)

// renderHTMLBlock handles ast.HTMLBlock nodes by parsing HTML content
// and converting supported tags to ANSI-decorated terminal output.
func renderHTMLBlock(buf *strings.Builder, n *ast.HTMLBlock, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	// Collect all lines of the HTML block.
	var raw strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		raw.Write(seg.Value(source))
	}
	html := raw.String()

	rendered := convertHTML(html, ctx, false)
	if rendered != "" {
		buf.WriteString(rendered)
	}

	return ast.WalkContinue, nil
}

// renderRawHTML handles ast.RawHTML nodes (inline HTML) by parsing tags
// and converting them to ANSI-decorated terminal output.
func renderRawHTML(buf *strings.Builder, n *ast.RawHTML, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	var raw strings.Builder
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		raw.Write(seg.Value(source))
	}
	html := raw.String()

	rendered := convertHTML(html, ctx, true)
	if rendered != "" {
		buf.WriteString(rendered)
	}

	return ast.WalkContinue, nil
}

// convertHTML parses an HTML string and converts supported tags to ANSI output.
// When inline is true, block-level constructs (like <hr>) produce inline equivalents.
func convertHTML(html string, ctx *RenderContext, inline bool) string {
	// Strip HTML comments.
	html = commentRe.ReplaceAllString(html, "")

	// Handle self-closing <br> tags first.
	html = selfClosingBrRe.ReplaceAllString(html, "\n")

	// Handle self-closing <hr> tags.
	if !inline {
		html = selfClosingHrRe.ReplaceAllStringFunc(html, func(_ string) string {
			return buildThematicBreak(ctx)
		})
	} else {
		html = selfClosingHrRe.ReplaceAllString(html, "\n")
	}

	// Handle <img> tags.
	html = imgTagRe.ReplaceAllStringFunc(html, func(tag string) string {
		return renderImgTag(tag, ctx)
	})

	// Process remaining tags via state machine.
	return processHTMLTags(html, ctx)
}

// processHTMLTags walks through HTML content, converting known tags to ANSI codes
// and stripping unknown tags while preserving inner text.
func processHTMLTags(html string, ctx *RenderContext) string {
	var buf strings.Builder
	lastIndex := 0
	inDetails := false
	inSummary := false
	var summaryBuf strings.Builder

	matches := htmlTagRe.FindAllStringSubmatchIndex(html, -1)
	for _, loc := range matches {
		// Write text before this tag.
		if loc[0] > lastIndex {
			text := html[lastIndex:loc[0]]
			if inSummary {
				summaryBuf.WriteString(text)
			} else {
				buf.WriteString(text)
			}
		}

		fullMatch := html[loc[0]:loc[1]]
		isClosing := html[loc[2]:loc[3]] == "/"
		tagName := strings.ToLower(html[loc[4]:loc[5]])
		attrs := ""
		if loc[6] != loc[7] {
			attrs = html[loc[6]:loc[7]]
		}

		lastIndex = loc[1]

		switch tagName {
		case "p", "div":
			// Strip <p> and <div> tags, just render inner content.
			if isClosing {
				buf.WriteByte('\n')
			}

		case "details":
			if !isClosing {
				inDetails = true
			} else {
				inDetails = false
				buf.WriteByte('\n')
			}

		case "summary":
			if !isClosing {
				inSummary = true
				summaryBuf.Reset()
			} else {
				inSummary = false
				buf.WriteString("▶ ")
				buf.WriteString(strings.TrimSpace(summaryBuf.String()))
				buf.WriteByte('\n')
			}

		case "a":
			if !isClosing {
				href := extractAttr(attrs, "href")
				if href != "" {
					buf.WriteString(linkOpen(ctx))
				}
			} else {
				buf.WriteString(Reset)
				// We don't have the href here easily, so just close the styling.
			}

		case "em", "i":
			if !isClosing {
				buf.WriteString(Italic)
			} else {
				buf.WriteString(ItalicOff)
			}

		case "strong", "b":
			if !isClosing {
				buf.WriteString(Bold)
			} else {
				buf.WriteString(BoldOff)
			}

		case "code":
			if !isClosing {
				if ctx != nil && ctx.Theme != nil && ctx.Theme.CodeBg != "" {
					buf.WriteString(ctx.Theme.CodeBg)
				}
			} else {
				buf.WriteString(Reset)
			}

		case "kbd":
			if !isClosing {
				if ctx != nil && ctx.Theme != nil && ctx.Theme.CodeBorder != "" {
					buf.WriteString(ctx.Theme.CodeBorder)
				}
				buf.WriteString("[")
			} else {
				buf.WriteString("]")
				buf.WriteString(Reset)
			}

		case "del", "s":
			if !isClosing {
				buf.WriteString(Strikethrough)
			} else {
				buf.WriteString(StrikethroughOff)
			}

		case "u":
			if !isClosing {
				buf.WriteString(Underline)
			} else {
				buf.WriteString(UnderlineOff)
			}

		case "sub", "sup":
			// Terminals can't do sub/superscript; just render text as-is.

		case "br", "hr":
			// Already handled by regex above, but catch any remaining.
			buf.WriteByte('\n')

		default:
			// Unknown tag: strip it, keep inner text.
			_ = fullMatch
			_ = inDetails
		}
	}

	// Write any remaining text after the last tag.
	if lastIndex < len(html) {
		remaining := html[lastIndex:]
		if inSummary {
			summaryBuf.WriteString(remaining)
		} else {
			buf.WriteString(remaining)
		}
	}

	return buf.String()
}

// renderImgTag converts an <img> tag to terminal output.
// Uses the existing image rendering pipeline if an image protocol is available,
// otherwise falls back to "[画像: alt]".
func renderImgTag(tag string, ctx *RenderContext) string {
	attrs := extractAttrsMap(tag)
	alt := attrs["alt"]
	src := attrs["src"]

	if src == "" && alt == "" {
		return ""
	}

	// Try to render the image using the existing pipeline.
	if src != "" && ctx != nil && ctx.ImageProtocol != terminal.ImageNone {
		img, err := loadImage(src)
		if err == nil {
			maxWidth := int(float64(ctx.TermWidth) * 0.8)
			if maxWidth < 1 {
				maxWidth = 1
			}
			encoded, encErr := encodeImageByProtocol(img, maxWidth, ctx.ImageProtocol)
			if encErr == nil {
				var buf strings.Builder
				buf.WriteString(encoded)
				buf.WriteByte('\n')
				if alt != "" {
					renderImageCaption(&buf, alt, ctx)
				}
				return buf.String()
			}
		}
	}

	// Fallback: render as text.
	if alt == "" {
		alt = src
	}
	var buf strings.Builder
	renderImageFallback(&buf, alt, ctx)
	return buf.String()
}

// buildThematicBreak produces a horizontal rule string matching renderThematicBreak output.
func buildThematicBreak(ctx *RenderContext) string {
	var buf strings.Builder
	width := 80
	if ctx != nil && ctx.TermWidth > 0 {
		width = ctx.TermWidth
	}
	if ctx != nil && ctx.Theme != nil && ctx.Theme.HRColor != "" {
		buf.WriteString(ctx.Theme.HRColor)
	}
	dashWidth := runewidth.RuneWidth('─')
	dashCount := width / dashWidth
	buf.WriteString(strings.Repeat("─", dashCount))
	if ctx != nil && ctx.Theme != nil && ctx.Theme.HRColor != "" {
		buf.WriteString(Reset)
	}
	buf.WriteByte('\n')
	return buf.String()
}

// linkOpen returns the ANSI codes to start a link (colored + underlined).
func linkOpen(ctx *RenderContext) string {
	var buf strings.Builder
	if ctx != nil && ctx.Theme != nil && ctx.Theme.LinkColor != "" {
		buf.WriteString(ctx.Theme.LinkColor)
	}
	buf.WriteString(Underline)
	return buf.String()
}

// extractAttr extracts a single attribute value from an attribute string.
func extractAttr(attrs string, name string) string {
	for _, m := range attrRe.FindAllStringSubmatch(attrs, -1) {
		if strings.ToLower(m[1]) == name {
			return m[2]
		}
	}
	return ""
}

// extractAttrsMap extracts all attributes from an HTML tag string into a map.
func extractAttrsMap(tag string) map[string]string {
	result := make(map[string]string)
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		result[strings.ToLower(m[1])] = m[2]
	}
	return result
}
