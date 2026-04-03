package pager

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// RenderFunc is called to re-render content. It receives the new terminal width
// and should return the rendered content string.
type RenderFunc func(termWidth int) string

// Pager manages interactive paged display of content.
type Pager struct {
	lines      []string   // content lines
	offset     int        // current scroll position (top visible line index)
	height     int        // visible lines (terminal height minus status bar)
	width      int        // terminal width
	searchTerm string     // current search term
	matches    []int      // line indices matching search
	matchIdx   int        // current match index
	renderFunc RenderFunc // callback for re-rendering (nil = no re-render)
	filePath   string     // file to watch (empty = no watching)
	hasImages  bool       // whether content contains image escape sequences
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
	hasImages := containsImageSequences(content)
	return &Pager{
		lines:     lines,
		offset:    0,
		height:    h,
		width:     termWidth,
		matches:   nil,
		matchIdx:  -1,
		hasImages: hasImages,
	}
}

// containsImageSequences checks if content contains terminal image escape sequences.
func containsImageSequences(content string) bool {
	return strings.Contains(content, "\x1b_G") || // Kitty
		strings.Contains(content, "\x1bPq") || // Sixel
		strings.Contains(content, "\x1b]1337;") // iTerm2
}

// SetRenderFunc sets the callback used to re-render content on resize or file change.
func (p *Pager) SetRenderFunc(fn RenderFunc) {
	p.renderFunc = fn
}

// SetFilePath sets the file path to watch for changes.
func (p *Pager) SetFilePath(path string) {
	p.filePath = path
}

// UpdateContent replaces the pager content and dimensions, clamping the scroll offset.
func (p *Pager) UpdateContent(content string, newWidth, newHeight int) {
	p.lines = strings.Split(content, "\n")
	p.width = newWidth
	p.height = newHeight - 1
	if p.height < 1 {
		p.height = 1
	}
	p.hasImages = containsImageSequences(content)
	// Clamp offset
	maxOffset := len(p.lines) - p.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
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

	// Clear any existing terminal images before redrawing.
	// Kitty: delete all images on screen
	// Sixel/iTerm2: clearing the screen is sufficient since they are inline
	if p.hasImages {
		buf.WriteString("\x1b_Ga=d\x1b\\") // Kitty: delete all placements
	}

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

// keyEvent represents a single key read from stdin.
type keyEvent struct {
	buf [3]byte
	n   int
	err error
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

	// Stdin reader goroutine.
	// NOTE: This goroutine intentionally leaks when Run() returns. os.Stdin.Read
	// blocks on terminal input and cannot be reliably interrupted across platforms.
	// This is acceptable because the process exits immediately after Run() returns
	// in main(), so the goroutine is cleaned up by the OS. The goroutine does not
	// access any Pager state — it only sends events on keyCh.
	keyCh := make(chan keyEvent, 1)
	go func() {
		for {
			var ev keyEvent
			ev.n, ev.err = os.Stdin.Read(ev.buf[:])
			keyCh <- ev
		}
	}()

	// SIGWINCH handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// File watcher
	doneCh := make(chan struct{})
	defer close(doneCh)

	var fileChangeCh <-chan struct{}
	if p.filePath != "" {
		ch := make(chan struct{}, 1)
		fileChangeCh = ch
		go p.watchFile(ch, doneCh)
	}

	for {
		select {
		case ev := <-keyCh:
			if ev.err != nil || ev.n == 0 {
				return nil
			}
			if p.handleKeyEvent(ev, fd, oldState) {
				return nil // quit
			}
			p.render(fd)

		case <-sigCh:
			p.handleResize(fd)

		case <-fileChangeCh:
			p.handleFileChange(fd)
		}
	}
}

// handleKeyEvent processes a single key event. Returns true if the pager should quit.
func (p *Pager) handleKeyEvent(ev keyEvent, fd int, oldState *term.State) bool {
	// Check for escape sequences (arrow keys, page up/down)
	if ev.n >= 3 && ev.buf[0] == 0x1b && ev.buf[1] == '[' {
		switch ev.buf[2] {
		case 'A': // Up arrow
			p.ScrollUp(1)
		case 'B': // Down arrow
			p.ScrollDown(1)
		case '5': // Page Up (ESC [ 5 ~)
			p.ScrollUp(p.height)
		case '6': // Page Down (ESC [ 6 ~)
			p.ScrollDown(p.height)
		}
		return false
	}

	key := ev.buf[0]
	switch key {
	case 'q':
		return true
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
	return false
}

// handleResize handles terminal resize (SIGWINCH).
func (p *Pager) handleResize(fd int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return
	}
	if p.renderFunc != nil {
		content := p.renderFunc(w)
		p.UpdateContent(content, w, h)
	} else {
		p.width = w
		p.height = h - 1
		if p.height < 1 {
			p.height = 1
		}
		maxOffset := len(p.lines) - p.height
		if maxOffset < 0 {
			maxOffset = 0
		}
		if p.offset > maxOffset {
			p.offset = maxOffset
		}
	}
	p.render(fd)
}

// handleFileChange handles file change notifications.
func (p *Pager) handleFileChange(fd int) {
	if p.renderFunc != nil {
		content := p.renderFunc(p.width)
		p.UpdateContent(content, p.width, p.height+1)
	}
	p.render(fd)
}

// watchFile polls the file for mtime changes and sends on ch when detected.
// It stops when doneCh is closed.
func (p *Pager) watchFile(ch chan<- struct{}, doneCh <-chan struct{}) {
	info, _ := os.Stat(p.filePath)
	var lastMod time.Time
	if info != nil {
		lastMod = info.ModTime()
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-doneCh:
			return
		case <-ticker.C:
			info, err := os.Stat(p.filePath)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}
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

	searchTermStr := string(searchBuf)
	if searchTermStr != "" {
		p.Search(searchTermStr)
		if len(p.matches) > 0 {
			p.matchIdx = 0
			p.offset = p.matches[0]
		}
	}
}
