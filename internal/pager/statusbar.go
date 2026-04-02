package pager

import "fmt"

// StatusBar returns the status bar string showing current position.
// Format: "行 {offset+1}/{totalLines}"
func (p *Pager) StatusBar() string {
	return fmt.Sprintf("行 %d/%d", p.offset+1, len(p.lines))
}
