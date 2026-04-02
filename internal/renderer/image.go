package renderer

import (
	"image"
	"strings"

	"github.com/user/mdrender/internal/sixel"
	"github.com/yuin/goldmark/ast"
)

// renderImage renders an image node.
// When SixelSupport is true: encodes the image as Sixel and appends an alt text caption.
// When SixelSupport is false: outputs "[画像: altテキスト]" as fallback.
// Error cases produce specific error messages and continue rendering.
func renderImage(buf *strings.Builder, n *ast.Image, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	altText := string(n.Text(source))
	dest := string(n.Destination)

	// Fallback mode: no Sixel support
	if !ctx.SixelSupport {
		renderImageFallback(buf, altText, ctx)
		return ast.WalkSkipChildren, nil
	}

	// Sixel mode: try to load and encode the image
	img, loadErr := loadImage(dest)
	if loadErr != nil {
		renderImageError(buf, loadErr, altText, dest, ctx)
		return ast.WalkSkipChildren, nil
	}

	// Calculate max width as 80% of terminal width
	maxWidth := int(float64(ctx.TermWidth) * 0.8)
	if maxWidth < 1 {
		maxWidth = 1
	}

	encoded, err := sixel.EncodeImage(img, maxWidth)
	if err != nil {
		renderImageErrorGeneric(buf, altText, ctx)
		return ast.WalkSkipChildren, nil
	}

	// Output Sixel data
	buf.WriteString(encoded)
	buf.WriteByte('\n')

	// Output caption with alt text
	renderImageCaption(buf, altText, ctx)

	return ast.WalkSkipChildren, nil
}

// renderImageFallback outputs the image in plain text fallback format.
func renderImageFallback(buf *strings.Builder, altText string, ctx *RenderContext) {
	if ctx.Theme != nil && ctx.Theme.ImageCaption != "" {
		buf.WriteString(ctx.Theme.ImageCaption)
	}
	buf.WriteString("[画像: ")
	buf.WriteString(altText)
	buf.WriteString("]")
	if ctx.Theme != nil && ctx.Theme.ImageCaption != "" {
		buf.WriteString(Reset)
	}
}

// renderImageCaption outputs the alt text as a caption below the Sixel image.
func renderImageCaption(buf *strings.Builder, altText string, ctx *RenderContext) {
	if altText == "" {
		return
	}
	if ctx.Theme != nil && ctx.Theme.ImageCaption != "" {
		buf.WriteString(ctx.Theme.ImageCaption)
	}
	buf.WriteString(altText)
	if ctx.Theme != nil && ctx.Theme.ImageCaption != "" {
		buf.WriteString(Reset)
	}
	buf.WriteByte('\n')
}

// renderImageError outputs an error message based on the type of load failure.
func renderImageError(buf *strings.Builder, err error, altText string, dest string, ctx *RenderContext) {
	if ctx.Theme != nil && ctx.Theme.ErrorColor != "" {
		buf.WriteString(ctx.Theme.ErrorColor)
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "failed to open image file") {
		buf.WriteString("[画像読み込みエラー: ")
		buf.WriteString(altText)
		buf.WriteString("]")
	} else if strings.Contains(errMsg, "failed to download image") {
		buf.WriteString("[画像取得エラー: ")
		buf.WriteString(altText)
		buf.WriteString("]")
	} else if strings.Contains(errMsg, "failed to decode image") {
		buf.WriteString("[非対応画像形式: ")
		buf.WriteString(altText)
		buf.WriteString("]")
	} else {
		buf.WriteString("[画像読み込みエラー: ")
		buf.WriteString(altText)
		buf.WriteString("]")
	}

	if ctx.Theme != nil && ctx.Theme.ErrorColor != "" {
		buf.WriteString(Reset)
	}
}

// renderImageErrorGeneric outputs a generic image error message.
func renderImageErrorGeneric(buf *strings.Builder, altText string, ctx *RenderContext) {
	if ctx.Theme != nil && ctx.Theme.ErrorColor != "" {
		buf.WriteString(ctx.Theme.ErrorColor)
	}
	buf.WriteString("[画像読み込みエラー: ")
	buf.WriteString(altText)
	buf.WriteString("]")
	if ctx.Theme != nil && ctx.Theme.ErrorColor != "" {
		buf.WriteString(Reset)
	}
}

// loadImage loads an image from a local path or remote URL.
func loadImage(dest string) (image.Image, error) {
	if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") {
		return sixel.LoadRemoteImage(dest)
	}
	return sixel.LoadLocalImage(dest)
}
