package renderer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/user/mdrender/internal/sixel"
	"github.com/user/mdrender/internal/terminal"
)

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

// RenderMermaid processes a Mermaid code block and returns the rendered string.
// It writes the code to a temp file, invokes mmdc to produce a PNG, and then
// either displays the PNG via Sixel or falls back to a text representation.
func RenderMermaid(code, theme string, ctx *RenderContext) string {
	var buf strings.Builder

	// Create temp input file.
	// Use a directory under $HOME if available, because snap-confined mmdc
	// cannot access the system /tmp directory.
	tmpDir := mermaidTempDir()
	inputFile, err := os.CreateTemp(tmpDir, "mermaid-*.mmd")
	if err != nil {
		// Temp file creation error
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
	inputPath := inputFile.Name()

	// Write mermaid code to temp file
	_, err = inputFile.WriteString(code)
	inputFile.Close()
	if err != nil {
		CleanupTempFiles(inputPath)
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
	defer CleanupTempFiles(inputPath, outputPath)

	// Check if mmdc is available
	mmdcPath, err := exec.LookPath("mmdc")
	if err != nil {
		// mmdc not found: render as normal code block with special label
		lines := strings.Split(code, "\n")
		renderCodeBox(&buf, lines, "mermaid (図の表示にはmmdcが必要です)", ctx)
		return buf.String()
	}

	// Build and execute mmdc command
	args := BuildMmdcArgs(inputPath, outputPath, theme)
	cmd := exec.Command(mmdcPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// mmdc execution error
		errText := string(output)
		if len(errText) > 100 {
			errText = errText[:100]
		}
		errMsg := fmt.Sprintf("[Mermaid変換エラー: %s]", errText)
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
			maxWidth := int(float64(ctx.TermWidth) * 0.8)
			if maxWidth < 1 {
				maxWidth = 1
			}
			img = sixel.ResizeImage(img, maxWidth)
			encoded, err := encodeImageByProtocol(img, maxWidth, ctx.ImageProtocol)
			if err == nil {
				buf.WriteString(encoded)
				buf.WriteByte('\n')
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
