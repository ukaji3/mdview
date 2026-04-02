package renderer

import (
	"os"
	"strings"
	"testing"

	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/terminal"
	"pgregory.net/rapid"
)

// Feature: markdown-terminal-renderer, Property 21: Mermaid図の検出
// **Validates: Requirements 14.1**
func TestProperty21_MermaidDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random language string: either "mermaid" or something else
		isMermaidInput := rapid.Bool().Draw(t, "isMermaidInput")

		var lang string
		if isMermaidInput {
			lang = "mermaid"
		} else {
			// Generate a non-"mermaid" language string
			lang = rapid.StringMatching(`[a-zA-Z0-9_\-]{0,20}`).Draw(t, "lang")
			// Ensure it's not accidentally "mermaid"
			if lang == "mermaid" {
				lang = "python"
			}
		}

		result := IsMermaid(lang)

		if isMermaidInput {
			if !result {
				t.Fatalf("IsMermaid(%q) = false, expected true", lang)
			}
		} else {
			if result {
				t.Fatalf("IsMermaid(%q) = true, expected false for non-mermaid lang", lang)
			}
		}

		// Also verify via full rendering pipeline: mermaid code blocks should be
		// treated differently from other language code blocks.
		codeContent := "graph TD\n    A-->B"
		markdown := "```" + lang + "\n" + codeContent + "\n```"
		source := []byte(markdown)
		node := parser.Parse(source)
		ctx := &RenderContext{
			TermWidth:     80,
			ColorMode:     terminal.ColorTrue,
			ImageProtocol: terminal.ImageNone,
			Theme:         terminal.DefaultTheme(),
			IsTTY:         true,
		}
		rendered := Render(node, source, ctx)

		if isMermaidInput {
			// When language is "mermaid", the output should contain mermaid-specific
			// markers (either mmdc-not-found label or Mermaid図 text)
			hasMermaidMarker := strings.Contains(rendered, "図の表示にはmmdcが必要です") ||
				strings.Contains(rendered, "Mermaid図") ||
				strings.Contains(rendered, "Mermaid変換エラー")
			if !hasMermaidMarker {
				t.Fatalf("mermaid code block should produce mermaid-specific output, got: %q", rendered)
			}
		} else if lang != "" {
			// Non-mermaid language: should render as normal code block with language label
			// and should NOT contain mermaid-specific markers
			hasMermaidMarker := strings.Contains(rendered, "図の表示にはmmdcが必要です") ||
				strings.Contains(rendered, "Mermaid図:")
			if hasMermaidMarker {
				t.Fatalf("non-mermaid code block (lang=%q) should not produce mermaid markers, got: %q", lang, rendered)
			}
		}
	})
}

// Feature: markdown-terminal-renderer, Property 22: Mermaidテーマオプションの受け渡し
// **Validates: Requirements 14.5**
func TestProperty22_MermaidThemeOption(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid Mermaid theme
		themes := []string{"default", "dark", "forest", "neutral"}
		themeIdx := rapid.IntRange(0, len(themes)-1).Draw(t, "themeIdx")
		theme := themes[themeIdx]

		inputFile := "/tmp/test-input.mmd"
		outputFile := "/tmp/test-output.png"

		args := BuildMmdcArgs(inputFile, outputFile, theme)

		// Verify -t <theme> is present in args
		foundTheme := false
		for i, arg := range args {
			if arg == "-t" && i+1 < len(args) && args[i+1] == theme {
				foundTheme = true
				break
			}
		}
		if !foundTheme {
			t.Fatalf("BuildMmdcArgs did not include -t %q in args: %v", theme, args)
		}

		// Verify -i <inputFile> is present
		foundInput := false
		for i, arg := range args {
			if arg == "-i" && i+1 < len(args) && args[i+1] == inputFile {
				foundInput = true
				break
			}
		}
		if !foundInput {
			t.Fatalf("BuildMmdcArgs did not include -i %q in args: %v", inputFile, args)
		}

		// Verify -o <outputFile> is present
		foundOutput := false
		for i, arg := range args {
			if arg == "-o" && i+1 < len(args) && args[i+1] == outputFile {
				foundOutput = true
				break
			}
		}
		if !foundOutput {
			t.Fatalf("BuildMmdcArgs did not include -o %q in args: %v", outputFile, args)
		}

		// Verify -b white is present
		foundBg := false
		for i, arg := range args {
			if arg == "-b" && i+1 < len(args) && args[i+1] == "white" {
				foundBg = true
				break
			}
		}
		if !foundBg {
			t.Fatalf("BuildMmdcArgs did not include -b white in args: %v", args)
		}
	})
}

// Feature: markdown-terminal-renderer, Property 22 (supplement): Default theme
// Validates: Requirements 14.6
func TestMermaidDefaultTheme(t *testing.T) {
	args := BuildMmdcArgs("/tmp/in.mmd", "/tmp/out.png", "")
	foundDefault := false
	for i, arg := range args {
		if arg == "-t" && i+1 < len(args) && args[i+1] == "default" {
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		t.Fatalf("BuildMmdcArgs with empty theme should default to 'default', got args: %v", args)
	}
}

// Feature: markdown-terminal-renderer, Property 23: Mermaid一時ファイルのクリーンアップ
// **Validates: Requirements 14.7**
func TestProperty23_MermaidTempFileCleanup(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random content to write to temp files
		content := rapid.StringMatching(`[A-Za-z0-9 \n]{1,100}`).Draw(t, "content")

		// Create temp files simulating what RenderMermaid would create
		inputFile, err := os.CreateTemp("", "mermaid-test-*.mmd")
		if err != nil {
			t.Fatalf("failed to create temp input file: %v", err)
		}
		inputPath := inputFile.Name()
		inputFile.WriteString(content)
		inputFile.Close()

		outputPath := inputPath + ".png"
		// Create the output file too
		outputFile, err := os.CreateTemp("", "mermaid-test-*.png")
		if err != nil {
			// Clean up input file
			os.Remove(inputPath)
			t.Fatalf("failed to create temp output file: %v", err)
		}
		// Rename to expected path
		outputFile.Close()
		os.Remove(outputFile.Name())
		f, err := os.Create(outputPath)
		if err != nil {
			os.Remove(inputPath)
			t.Fatalf("failed to create output at expected path: %v", err)
		}
		f.WriteString("fake png data")
		f.Close()

		// Verify files exist before cleanup
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			t.Fatalf("input temp file should exist before cleanup: %s", inputPath)
		}
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("output temp file should exist before cleanup: %s", outputPath)
		}

		// Call cleanup
		CleanupTempFiles(inputPath, outputPath)

		// Verify files are deleted after cleanup
		if _, err := os.Stat(inputPath); !os.IsNotExist(err) {
			t.Fatalf("input temp file should not exist after cleanup: %s", inputPath)
		}
		if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
			t.Fatalf("output temp file should not exist after cleanup: %s", outputPath)
		}
	})
}

// Unit test: mmdc not found fallback
func TestMermaidMmdcNotFound(t *testing.T) {
	// Ensure mmdc is not in PATH by using a restricted PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	ctx := &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         true,
	}

	result := RenderMermaid("graph TD\n    A-->B", "default", ctx)

	if !strings.Contains(result, "図の表示にはmmdcが必要です") {
		t.Fatalf("expected mmdc-not-found label, got: %q", result)
	}
	// Should still contain box drawing (rendered as code block)
	if !strings.Contains(result, "┌") || !strings.Contains(result, "┘") {
		t.Fatalf("expected code block box drawing in fallback, got: %q", result)
	}
}

// Unit test: temp file error message format
func TestMermaidTempFileError(t *testing.T) {
	ctx := &RenderContext{
		TermWidth:     80,
		ColorMode:     terminal.ColorTrue,
		ImageProtocol: terminal.ImageNone,
		Theme:         terminal.DefaultTheme(),
		IsTTY:         true,
	}

	// Test CleanupTempFiles with non-existent files (should not panic)
	CleanupTempFiles("/nonexistent/path1", "/nonexistent/path2")

	// Verify RenderMermaid with mmdc not found still produces valid output
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	result := RenderMermaid("sequenceDiagram\n    A->>B: Hello", "dark", ctx)
	if !strings.Contains(result, "mermaid") {
		t.Fatalf("expected 'mermaid' in output, got: %q", result)
	}
}

// Unit test: detectMermaidDiagramType
func TestDetectMermaidDiagramType(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"graph TD\n    A-->B", "graph"},
		{"flowchart LR\n    A-->B", "flowchart"},
		{"sequenceDiagram\n    A->>B: Hello", "sequenceDiagram"},
		{"classDiagram\n    Class01 <|-- Class02", "classDiagram"},
		{"stateDiagram-v2\n    [*] --> Active", "stateDiagram"},
		{"erDiagram\n    CUSTOMER ||--o{ ORDER : places", "erDiagram"},
		{"gantt\n    title A Gantt Diagram", "gantt"},
		{"pie\n    title Pets", "pie"},
		{"mindmap\n    root((mindmap))", "mindmap"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		result := detectMermaidDiagramType(tt.code)
		if result != tt.expected {
			t.Errorf("detectMermaidDiagramType(%q) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}
