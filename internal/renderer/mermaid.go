package renderer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ukaji3/mdview/internal/sixel"
	"github.com/ukaji3/mdview/internal/terminal"
)

// truncateRunes truncates s to at most n runes (UTF-8 safe).
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// IsMermaid returns true only when the language identifier is exactly "mermaid".
func IsMermaid(lang string) bool {
	return lang == "mermaid"
}

// BuildMmdcArgs constructs the mmdc command arguments for converting a Mermaid
// input file to a PNG output file with the given theme.
func BuildMmdcArgs(inputFile, outputFile, theme string) []string {
	if theme == "" {
		theme = "default"
	}
	return []string{
		"-i", inputFile,
		"-o", outputFile,
		"-b", "white",
		"-t", theme,
	}
}

// CleanupTempFiles removes the given file paths, ignoring errors.
func CleanupTempFiles(paths ...string) {
	for _, p := range paths {
		os.Remove(p)
	}
}

// mmdcTimeout is the maximum time allowed for a single mmdc invocation.
// mmdc launches a headless browser and can hang on pathological input.
const mmdcTimeout = 30 * time.Second

// RenderMermaid processes a Mermaid code block and returns the rendered string.
// It writes the code to a temp file, invokes mmdc to produce a PNG, and then
// either displays the PNG via the detected image protocol or falls back to a
// text representation.
func RenderMermaid(code, theme string, ctx *RenderContext) string {
	var buf strings.Builder

	// Create a dedicated temp directory so that ALL files mmdc generates
	// (including multi-page outputs like output-1.png) are cleaned up by
	// removing the directory.
	// Use a directory under $HOME if available, because snap-confined mmdc
	// cannot access the system /tmp directory.
	workDir, err := os.MkdirTemp(mermaidTempDir(), "mdview-mermaid-*")
	if err != nil {
		errMsg := "[Mermaid一時ファイルエラー]"
		if ctx.Theme != nil {
			buf.WriteString(ctx.Theme.ErrorColor)
		}
		buf.WriteString(errMsg)
		buf.WriteString(Reset)
		buf.WriteByte('\n')
		// Render as normal code block
		lines := strings.Split(code, "\n")
		renderCodeBox(&buf, lines, "mermaid", ctx)
		return buf.String()
	}
	defer os.RemoveAll(workDir)

	inputPath := filepath.Join(workDir, "diagram.mmd")
	if err := os.WriteFile(inputPath, []byte(code), 0600); err != nil {
		errMsg := "[Mermaid一時ファイルエラー]"
		if ctx.Theme != nil {
			buf.WriteString(ctx.Theme.ErrorColor)
		}
		buf.WriteString(errMsg)
		buf.WriteString(Reset)
		buf.WriteByte('\n')
		lines := strings.Split(code, "\n")
		renderCodeBox(&buf, lines, "mermaid", ctx)
		return buf.String()
	}

	// Output PNG path
	outputPath := inputPath + ".png"

	// Check if mmdc is available
	mmdcPath, err := exec.LookPath("mmdc")
	if err != nil {
		// mmdc not found: render as normal code block with special label
		lines := strings.Split(code, "\n")
		renderCodeBox(&buf, lines, "mermaid (図の表示にはmmdcが必要です)", ctx)
		return buf.String()
	}

	// Build and execute mmdc command with a timeout.
	cmdCtx, cancel := context.WithTimeout(context.Background(), mmdcTimeout)
	defer cancel()
	args := BuildMmdcArgs(inputPath, outputPath, theme)
	cmd := exec.CommandContext(cmdCtx, mmdcPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var errMsg string
		if cmdCtx.Err() == context.DeadlineExceeded {
			errMsg = "[Mermaid変換エラー: タイムアウト]"
		} else {
			errMsg = fmt.Sprintf("[Mermaid変換エラー: %s]", truncateRunes(string(output), 100))
		}
		if ctx.Theme != nil {
			buf.WriteString(ctx.Theme.ErrorColor)
		}
		buf.WriteString(errMsg)
		buf.WriteString(Reset)
		buf.WriteByte('\n')
		lines := strings.Split(code, "\n")
		renderCodeBox(&buf, lines, "mermaid", ctx)
		return buf.String()
	}

	// Success: try to display the PNG
	if ctx.ImageProtocol != terminal.ImageNone && ctx.ColorMode != terminal.ColorNone {
		// Load PNG and encode using the detected image protocol
		img, err := sixel.LoadLocalImage(outputPath)
		if err == nil {
			maxWidthPx := imageMaxPixelWidth(ctx)
			img = sixel.ResizeImage(img, maxWidthPx)
			encoded, err := encodeImageByProtocol(img, maxWidthPx, ctx.ImageProtocol)
			if err == nil {
				buf.WriteString(encoded)
				buf.WriteByte('\n')

				// Insert placeholder lines so the pager accounts for the image's visual height.
				imgHeight := img.Bounds().Dy()
				rows := imageRows(imgHeight, ctx.CellHeight)
				for i := 1; i < rows; i++ {
					buf.WriteByte('\n')
				}

				return buf.String()
			}
		}
		// If image encoding fails, fall through to text fallback
	}

	// No image protocol support or encoding failed: text fallback
	// Detect diagram type from first line of code
	diagramType := detectMermaidDiagramType(code)
	infoMsg := fmt.Sprintf("[Mermaid図: %s]", diagramType)
	if ctx.Theme != nil {
		buf.WriteString(ctx.Theme.ImageCaption)
	}
	buf.WriteString(infoMsg)
	buf.WriteString(Reset)
	buf.WriteByte('\n')
	lines := strings.Split(code, "\n")
	renderCodeBox(&buf, lines, "mermaid", ctx)
	return buf.String()
}

// mermaidTempDir returns a temporary directory suitable for mmdc.
// Snap-confined mmdc cannot access /tmp or ~/.cache, so we detect snap
// installations and use ~/snap/mermaid-cli/common/tmp/ instead.
func mermaidTempDir() string {
	// Check if mmdc is a snap binary
	mmdcPath, err := exec.LookPath("mmdc")
	if err == nil && strings.Contains(mmdcPath, "/snap/") {
		home, err := os.UserHomeDir()
		if err == nil {
			dir := filepath.Join(home, "snap", "mermaid-cli", "common", "tmp")
			if err := os.MkdirAll(dir, 0755); err == nil {
				return dir
			}
		}
	}
	return os.TempDir()
}

// detectMermaidDiagramType extracts the diagram type from the first line of
// Mermaid code (e.g., "graph TD", "sequenceDiagram", "classDiagram").
func detectMermaidDiagramType(code string) string {
	firstLine := strings.TrimSpace(code)
	if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
		firstLine = strings.TrimSpace(firstLine[:idx])
	}
	// Common mermaid diagram keywords
	keywords := []string{
		"graph", "flowchart", "sequenceDiagram", "classDiagram",
		"stateDiagram", "erDiagram", "gantt", "pie", "mindmap",
		"journey", "gitgraph", "timeline",
	}
	lower := strings.ToLower(firstLine)
	for _, kw := range keywords {
		if strings.HasPrefix(lower, strings.ToLower(kw)) {
			return kw
		}
	}
	// Return the first word if no keyword matched
	if fields := strings.Fields(firstLine); len(fields) > 0 {
		return fields[0]
	}
	return "unknown"
}
