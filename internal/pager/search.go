package pager

import "strings"

// Search finds all line indices containing the given term (case-sensitive).
// It stores the results in the pager's matches and resets matchIdx.
func (p *Pager) Search(term string) []int {
	p.searchTerm = term
	p.matches = nil
	p.matchIdx = -1

	for i, line := range p.lines {
		if strings.Contains(line, term) {
			p.matches = append(p.matches, i)
		}
	}
	return p.matches
}

// NextMatch advances to the next search match and scrolls to it.
func (p *Pager) NextMatch() {
	if len(p.matches) == 0 {
		return
	}
	p.matchIdx++
	if p.matchIdx >= len(p.matches) {
		p.matchIdx = 0 // wrap around
	}
	p.offset = p.matches[p.matchIdx]
	// Clamp offset
	maxOffset := len(p.lines) - p.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
}

// PrevMatch moves to the previous search match and scrolls to it.
func (p *Pager) PrevMatch() {
	if len(p.matches) == 0 {
		return
	}
	p.matchIdx--
	if p.matchIdx < 0 {
		p.matchIdx = len(p.matches) - 1 // wrap around
	}
	p.offset = p.matches[p.matchIdx]
	// Clamp offset
	maxOffset := len(p.lines) - p.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
}
