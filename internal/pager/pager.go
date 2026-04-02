package pager

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Pager manages interactive paged display of content.
type Pager struct {
	lines      []string // content lines
	offset     int      // current scroll position (top visible line index)
	height     int      // visible lines (terminal height minus status bar)
	width      int      // terminal width
	searchTerm string   // current search term
	matches    []int    // line indices matching search
	matchIdx   int      // current match index
}

// ShouldPage determines whether pager mode should be used.
// Returns true only when lineCount > termHeight AND isTTY AND !noPager.
func ShouldPage(lineCount, termHeight int, isTTY, noPager bool) bool {
	return lineCount > termHeight && isTTY && !noPager
}

// New creates a new Pager instance from content string.
// termHeight is reduced by 1 to reserve space for the status bar.
func New(content string, termWidth, termHeight int) *Pager {
	lines := strings.Split(content, "\n")
	h := termHeight - 1
	if h < 1 {
		h = 1
	}
	return &Pager{
		lines:    lines,
		offset:   0,
		height:   h,
		width:    termWidth,
		matches:  nil,
		matchIdx: -1,
	}
}

// ScrollDown scrolls down by n lines, clamping to the maximum offset.
func (p *Pager) ScrollDown(n int) {
	maxOffset := len(p.lines) - p.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	p.offset += n
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
}

// ScrollUp scrolls up by n lines, clamping to 0.
func (p *Pager) ScrollUp(n int) {
	p.offset -= n
	if p.offset < 0 {
		p.offset = 0
	}
}

// GoToTop moves to the beginning of the document.
func (p *Pager) GoToTop() {
	p.offset = 0
}

// GoToBottom moves to the end of the document.
func (p *Pager) GoToBottom() {
	maxOffset := len(p.lines) - p.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	p.offset = maxOffset
}

// VisibleLines returns the lines currently visible in the viewport.
func (p *Pager) VisibleLines() []string {
	end := p.offset + p.height
	if end > len(p.lines) {
		end = len(p.lines)
	}
	if p.offset >= len(p.lines) {
		return nil
	}
	return p.lines[p.offset:end]
}

// Offset returns the current scroll offset.
func (p *Pager) Offset() int {
	return p.offset
}

// LineCount returns the total number of lines.
func (p *Pager) LineCount() int {
	return len(p.lines)
}

// Height returns the visible height.
func (p *Pager) Height() int {
	return p.height
}

// render draws the current viewport and status bar to the terminal.
func (p *Pager) render(fd int) {
	var buf strings.Builder
	// Move cursor to top-left
	buf.WriteString("\x1b[H")
	visible := p.VisibleLines()
	for i := 0; i < p.height; i++ {
		// Clear line
		buf.WriteString("\x1b[2K")
		if i < len(visible) {
			buf.WriteString(visible[i])
		}
		buf.WriteString("\r\n")
	}
	// Status bar on last line
	buf.WriteString("\x1b[2K")
	buf.WriteString("\x1b[7m") // reverse video
	buf.WriteString(p.StatusBar())
	buf.WriteString("\x1b[0m") // reset
	os.Stdout.WriteString(buf.String())
}

// Run starts the pager main loop with alternate screen buffer and raw mode.
// On raw mode failure, it falls back to printing content directly.
func (p *Pager) Run() error {
	fd := int(os.Stdin.Fd())

	// Try to enter raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: print content directly
		fmt.Print(strings.Join(p.lines, "\n"))
		return nil
	}
	defer term.Restore(fd, oldState)

	// Switch to alternate screen buffer
	os.Stdout.WriteString("\x1b[?1049h")
	defer os.Stdout.WriteString("\x1b[?1049l")

	// Hide cursor
	os.Stdout.WriteString("\x1b[?25l")
	defer os.Stdout.WriteString("\x1b[?25h")

	p.render(fd)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}

		key := buf[0]

		// Check for escape sequences (arrow keys, page up/down)
		if n >= 3 && buf[0] == 0x1b && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up arrow
				p.ScrollUp(1)
			case 'B': // Down arrow
				p.ScrollDown(1)
			case '5': // Page Up (ESC [ 5 ~)
				p.ScrollUp(p.height)
			case '6': // Page Down (ESC [ 6 ~)
				p.ScrollDown(p.height)
			}
			p.render(fd)
			continue
		}

		switch key {
		case 'q':
			return nil
		case 'j':
			p.ScrollDown(1)
		case 'k':
			p.ScrollUp(1)
		case ' ': // space = page down
			p.ScrollDown(p.height)
		case 'b': // page up
			p.ScrollUp(p.height)
		case 'g':
			p.GoToTop()
		case 'G':
			p.GoToBottom()
		case 'n':
			p.NextMatch()
		case 'N':
			p.PrevMatch()
		case '/':
			p.handleSearch(fd, oldState)
		}
		p.render(fd)
	}
	return nil
}

// handleSearch enters search mode, reads a search term, and performs search.
func (p *Pager) handleSearch(fd int, oldState *term.State) {
	// Show search prompt on status bar line
	var buf strings.Builder
	buf.WriteString("\x1b[")
	buf.WriteString(fmt.Sprintf("%d", p.height+1))
	buf.WriteString(";1H")
	buf.WriteString("\x1b[2K")
	buf.WriteString("\x1b[7m/\x1b[0m")
	os.Stdout.WriteString(buf.String())

	// Read search term character by character
	var searchBuf []byte
	readBuf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(readBuf)
		if err != nil || n == 0 {
			break
		}
		ch := readBuf[0]
		if ch == '\r' || ch == '\n' {
			break
		}
		if ch == 0x1b { // Escape - cancel search
			return
		}
		if ch == 0x7f || ch == 0x08 { // Backspace
			if len(searchBuf) > 0 {
				searchBuf = searchBuf[:len(searchBuf)-1]
			}
		} else {
			searchBuf = append(searchBuf, ch)
		}
		// Update search prompt display
		var prompt strings.Builder
		prompt.WriteString("\x1b[")
		prompt.WriteString(fmt.Sprintf("%d", p.height+1))
		prompt.WriteString(";1H")
		prompt.WriteString("\x1b[2K")
		prompt.WriteString("\x1b[7m/")
		prompt.WriteString(string(searchBuf))
		prompt.WriteString("\x1b[0m")
		os.Stdout.WriteString(prompt.String())
	}

	term := string(searchBuf)
	if term != "" {
		p.Search(term)
		if len(p.matches) > 0 {
			p.matchIdx = 0
			p.offset = p.matches[0]
		}
	}
}
