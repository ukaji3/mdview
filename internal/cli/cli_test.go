package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// --- ParseArgs tests ---

func TestParseArgs_NoArgs(t *testing.T) {
	cfg, err := ParseArgs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FilePath != "" {
		t.Errorf("expected empty FilePath, got %q", cfg.FilePath)
	}
	if cfg.MermaidTheme != "default" {
		t.Errorf("expected MermaidTheme 'default', got %q", cfg.MermaidTheme)
	}
	if cfg.NoPager {
		t.Error("expected NoPager false")
	}
	if cfg.PrettyPrint {
		t.Error("expected PrettyPrint false")
	}
	if cfg.NoColor {
		t.Error("expected NoColor false")
	}
}

func TestParseArgs_FilePathOnly(t *testing.T) {
	cfg, err := ParseArgs([]string{"README.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FilePath != "README.md" {
		t.Errorf("expected FilePath 'README.md', got %q", cfg.FilePath)
	}
}

func TestParseArgs_AllOptions(t *testing.T) {
	cfg, err := ParseArgs([]string{
		"--mermaid-theme", "dark",
		"--no-pager",
		"--pretty-print",
		"--no-color",
		"doc.md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FilePath != "doc.md" {
		t.Errorf("expected FilePath 'doc.md', got %q", cfg.FilePath)
	}
	if cfg.MermaidTheme != "dark" {
		t.Errorf("expected MermaidTheme 'dark', got %q", cfg.MermaidTheme)
	}
	if !cfg.NoPager {
		t.Error("expected NoPager true")
	}
	if !cfg.PrettyPrint {
		t.Error("expected PrettyPrint true")
	}
	if !cfg.NoColor {
		t.Error("expected NoColor true")
	}
}

func TestParseArgs_MermaidThemeForest(t *testing.T) {
	cfg, err := ParseArgs([]string{"--mermaid-theme", "forest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MermaidTheme != "forest" {
		t.Errorf("expected MermaidTheme 'forest', got %q", cfg.MermaidTheme)
	}
}

func TestParseArgs_InvalidFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// --- ReadInput tests ---

func TestReadInput_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := []byte("# Hello World\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &Config{FilePath: path}
	data, err := ReadInput(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected %q, got %q", content, data)
	}
}

func TestReadInput_FileNotFound(t *testing.T) {
	cfg := &Config{FilePath: "/nonexistent/path/file.md"}
	_, err := ReadInput(cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if got := err.Error(); got != "file not found: /nonexistent/path/file.md" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestReadInput_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noperm.md")
	if err := os.WriteFile(path, []byte("test"), 0000); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &Config{FilePath: path}
	_, err := ReadInput(cfg)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}
	expected := "permission denied: " + path
	if got := err.Error(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestReadInput_NoInputNoStdin(t *testing.T) {
	// Override IsStdinPiped to simulate no pipe
	origFn := IsStdinPiped
	IsStdinPiped = func() bool { return false }
	defer func() { IsStdinPiped = origFn }()

	cfg := &Config{}
	_, err := ReadInput(cfg)
	if err == nil {
		t.Fatal("expected error when no input and no pipe")
	}
}

func TestReadInput_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &Config{FilePath: path}
	data, err := ReadInput(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(data))
	}
}

// --- Property-based tests ---

// Feature: markdown-terminal-renderer, Property 25: 入力読み込みの正確性
// Validates: Requirements 1.1, 1.2
func TestProperty25_ReadInputAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random byte slice
		data := rapid.SliceOf(rapid.Byte()).Draw(t, "data")

		// Write bytes to a temp file
		dir, err := os.MkdirTemp("", "prop25-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(dir)

		path := filepath.Join(dir, "input.md")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Create Config with the temp file path and call ReadInput
		cfg := &Config{FilePath: path}
		result, err := ReadInput(cfg)
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}

		// Verify the result matches the original bytes exactly
		if !bytes.Equal(result, data) {
			t.Fatalf("ReadInput result does not match original data: got %d bytes, want %d bytes", len(result), len(data))
		}
	})
}
