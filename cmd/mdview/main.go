package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/user/mdrender/internal/cli"
	"github.com/user/mdrender/internal/pager"
	"github.com/user/mdrender/internal/parser"
	"github.com/user/mdrender/internal/prettyprint"
	"github.com/user/mdrender/internal/renderer"
	"github.com/user/mdrender/internal/terminal"
	"golang.org/x/term"
)

func main() {
	// 1. Parse CLI arguments
	config, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 2. Read input (file or stdin)
	source, err := cli.ReadInput(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 3. Parse Markdown into AST
	ast := parser.Parse(source)

	// 4. Pretty Print mode: output regenerated Markdown and exit
	if config.PrettyPrint {
		fmt.Print(prettyprint.PrettyPrint(ast, source))
		return
	}

	// 5. Detect terminal capabilities
	caps := terminal.DetectCapabilities()

	// Override ColorMode if --no-color is set
	if config.NoColor {
		caps.ColorMode = terminal.ColorNone
	}

	// 6. Build render context
	ctx := &renderer.RenderContext{
		TermWidth:    caps.Width,
		ColorMode:    caps.ColorMode,
		SixelSupport: caps.SixelSupport,
		Theme:        caps.Theme,
		IsTTY:        caps.IsTTY,
		MermaidTheme: config.MermaidTheme,
	}

	// 7. Render Markdown
	output := renderer.Render(ast, source, ctx)

	// 8. Determine terminal height for pager decision
	termHeight := 24
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && h > 0 {
		termHeight = h
	}

	// 9. Count lines in output
	lineCount := strings.Count(output, "\n") + 1

	// 10. Pager decision
	if pager.ShouldPage(lineCount, termHeight, caps.IsTTY, config.NoPager) {
		p := pager.New(output, caps.Width, termHeight)
		if err := p.Run(); err != nil {
			// Fallback: print directly on pager error
			fmt.Print(output)
		}
	} else {
		fmt.Print(output)
	}
}
