package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ukaji3/mdview/internal/cli"
	"github.com/ukaji3/mdview/internal/pager"
	"github.com/ukaji3/mdview/internal/parser"
	"github.com/ukaji3/mdview/internal/prettyprint"
	"github.com/ukaji3/mdview/internal/renderer"
	"github.com/ukaji3/mdview/internal/terminal"
	"golang.org/x/term"
)

func main() {
	// 1. Parse CLI arguments
	config, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 2. Check image mode: display diagnostics and exit
	if config.CheckImage {
		terminal.CheckImageSupport()
		return
	}

	// 3. Read input (file or stdin)
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
		TermWidth:     caps.Width,
		CellHeight:    caps.CellHeight,
		ColorMode:     caps.ColorMode,
		ImageProtocol: caps.ImageProtocol,
		Theme:         caps.Theme,
		IsTTY:         caps.IsTTY,
		MermaidTheme:  config.MermaidTheme,
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
		// Create render function closure for re-rendering on resize/file change
		renderFunc := func(termWidth int) string {
			source, err := cli.ReadInput(config)
			if err != nil {
				return fmt.Sprintf("Error reading file: %v", err)
			}
			ast := parser.Parse(source)
			rctx := &renderer.RenderContext{
				TermWidth:     termWidth,
				CellHeight:    caps.CellHeight,
				ColorMode:     caps.ColorMode,
				ImageProtocol: caps.ImageProtocol,
				Theme:         caps.Theme,
				IsTTY:         caps.IsTTY,
				MermaidTheme:  config.MermaidTheme,
			}
			return renderer.Render(ast, source, rctx)
		}

		p := pager.New(output, caps.Width, termHeight)
		p.SetRenderFunc(renderFunc)
		if config.FilePath != "" {
			p.SetFilePath(config.FilePath)
		}
		if err := p.Run(); err != nil {
			// Fallback: print directly on pager error
			fmt.Print(output)
		}
	} else {
		fmt.Print(output)
	}
}
