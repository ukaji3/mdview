package renderer

import (
	"fmt"
	"image"
	"strings"

	"github.com/ukaji3/mdview/internal/iterm2"
	"github.com/ukaji3/mdview/internal/kitty"
	"github.com/ukaji3/mdview/internal/sixel"
	"github.com/ukaji3/mdview/internal/terminal"
	"github.com/yuin/goldmark/ast"
)

// TODO: 画像ロード/リサイズ関数がsixelパッケージに依存しています。
// 共通の画像ユーティリティパッケージに分離するのが望ましいです。

// renderImage renders an image node.
// When ImageProtocol is set: encodes the image using the appropriate protocol and appends an alt text caption.
// When ImageProtocol is ImageNone: outputs "[画像: altテキスト]" as fallback.
// Error cases produce specific error messages and continue rendering.
func renderImage(buf *strings.Builder, n *ast.Image, entering bool, source []byte, ctx *RenderContext) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	altText := string(n.Text(source))
	dest := string(n.Destination)

	// Fallback mode: no image protocol support
	if ctx.ImageProtocol == terminal.ImageNone {
		renderImageFallback(buf, altText, ctx)
		return ast.WalkSkipChildren, nil
	}

	// Image mode: try to load and encode the image
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

	encoded, err := encodeImageByProtocol(img, maxWidth, ctx.ImageProtocol)
	if err != nil {
		renderImageErrorGeneric(buf, altText, ctx)
		return ast.WalkSkipChildren, nil
	}

	// Output encoded image data
	buf.WriteString(encoded)
	buf.WriteByte('\n')

	// Insert placeholder lines so the pager accounts for the image's visual height.
	// The image escape sequence counts as 1 line; we add (rows - 1) empty lines.
	imgHeight := img.Bounds().Dy()
	rows := imageRows(imgHeight, ctx.CellHeight)
	for i := 1; i < rows; i++ {
		buf.WriteByte('\n')
	}

	// Output caption with alt text
	renderImageCaption(buf, altText, ctx)

	return ast.WalkSkipChildren, nil
}

// imageRows calculates how many terminal rows an image of the given pixel height
// will occupy, based on the terminal cell height in pixels.
func imageRows(imgHeight int, cellHeight int) int {
	if cellHeight <= 0 {
		cellHeight = 16 // fallback
	}
	rows := (imgHeight + cellHeight - 1) / cellHeight
	if rows < 1 {
		rows = 1
	}
	return rows
}

// encodeImageByProtocol dispatches image encoding to the appropriate protocol encoder.
func encodeImageByProtocol(img image.Image, maxWidth int, proto terminal.ImageProtocol) (string, error) {
	switch proto {
	case terminal.ImageSixel:
		return sixel.EncodeImage(img, maxWidth)
	case terminal.ImageKitty:
		return kitty.EncodeImage(img, maxWidth)
	case terminal.ImageITerm2:
		return iterm2.EncodeImage(img, maxWidth)
	default:
		return "", fmt.Errorf("unsupported image protocol: %d", proto)
	}
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

// renderImageCaption outputs the alt text as a caption below the image.
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
// NOTE: パス走査の検証は行いません。CLIツールではユーザーが入力を制御するため、
// パストラバーサルは設計上の意図通りです。
func loadImage(dest string) (image.Image, error) {
	if strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://") {
		return sixel.LoadRemoteImage(dest)
	}
	return sixel.LoadLocalImage(dest)
}
