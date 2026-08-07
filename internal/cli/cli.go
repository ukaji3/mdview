package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Config holds CLI options parsed from command-line arguments.
type Config struct {
	FilePath     string // Input file path (empty means stdin)
	MermaidTheme string // Mermaid theme (default, dark, forest, neutral)
	NoColor      bool   // Disable color output
	NoPager      bool   // Disable pager mode
	NoMermaid    bool   // Disable Mermaid diagram rendering
	PrettyPrint  bool   // Pretty print mode (output Markdown from AST)
	CheckImage   bool   // Check and display image protocol support
}

// Usage message printed when no input is provided.
const usageText = `Usage: mdview [options] [file]

Render Markdown beautifully in the terminal.

Arguments:
  file                  Markdown file to render (reads stdin if omitted)

Options:
  --mermaid-theme <theme>  Mermaid diagram theme (default, dark, forest, neutral)
  --no-mermaid               Disable Mermaid diagram rendering
  --no-pager               Disable pager mode
  --pretty-print           Output re-generated Markdown from AST
  --no-color               Disable color output
  --check-image            Check terminal image protocol support
`

// ErrHelp is returned by ParseArgs when the user requests help (-h / --help).
// The caller should print Usage() and exit with status 0.
var ErrHelp = flag.ErrHelp

// Usage returns the usage message.
func Usage() string {
	return usageText
}

// validMermaidThemes is the set of accepted --mermaid-theme values.
var validMermaidThemes = map[string]bool{
	"default": true,
	"dark":    true,
	"forest":  true,
	"neutral": true,
}

// reorderArgs partitions args so that all flags come before positional
// arguments. Go's flag package stops parsing at the first non-flag argument,
// which would silently ignore flags in "mdview FILE --no-color".
// A bare "--" terminates flag parsing as usual.
func reorderArgs(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			// --mermaid-theme takes a separate value argument
			if (a == "--mermaid-theme" || a == "-mermaid-theme") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return append(flags, positionals...)
}

// ParseArgs parses command-line arguments and returns a Config.
// args should be os.Args[1:] style (without the program name).
// Returns ErrHelp if the user requested help.
func ParseArgs(args []string) (*Config, error) {
	fs := flag.NewFlagSet("mdview", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress default flag error output

	cfg := &Config{}
	fs.StringVar(&cfg.MermaidTheme, "mermaid-theme", "default", "")
	fs.BoolVar(&cfg.NoPager, "no-pager", false, "")
	fs.BoolVar(&cfg.NoMermaid, "no-mermaid", false, "")
	fs.BoolVar(&cfg.PrettyPrint, "pretty-print", false, "")
	fs.BoolVar(&cfg.NoColor, "no-color", false, "")
	fs.BoolVar(&cfg.CheckImage, "check-image", false, "")

	if err := fs.Parse(reorderArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if !validMermaidThemes[cfg.MermaidTheme] {
		return nil, fmt.Errorf("invalid --mermaid-theme %q (valid: default, dark, forest, neutral)", cfg.MermaidTheme)
	}

	// Remaining positional argument is the file path
	remaining := fs.Args()
	if len(remaining) > 1 {
		return nil, fmt.Errorf("too many arguments: expected at most one file, got %d", len(remaining))
	}
	if len(remaining) == 1 {
		cfg.FilePath = remaining[0]
	}

	return cfg, nil
}

// IsStdinPiped reports whether stdin has piped data (is not a terminal).
// This is extracted as a variable so tests can override it.
var IsStdinPiped = func() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

// ReadInput reads Markdown content from a file or stdin based on the Config.
// If FilePath is set, it reads from the file. Otherwise it reads from stdin.
// Returns an error if no input source is available.
func ReadInput(config *Config) ([]byte, error) {
	if config.FilePath != "" {
		return readFile(config.FilePath)
	}

	// No file path — try stdin
	if !IsStdinPiped() {
		return nil, errors.New("no input provided\n\n" + usageText)
	}

	return io.ReadAll(os.Stdin)
}

// readFile reads the contents of the given file path with descriptive errors.
func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("permission denied: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %s: %w", path, err)
	}
	return data, nil
}
