package pager

import (
	"regexp"
	"strings"
)

// ansiEscRe matches ANSI escape sequences of the form \033[...m for stripping.
var ansiEscRe = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape sequences from a string so that
// search matching operates on visible text only.
func stripANSI(s string) string {
	return ansiEscRe.ReplaceAllString(s, "")
}

// Search finds all line indices containing the given term (case-sensitive).
// ANSI escape codes are stripped before matching so that search operates
// on visible text only.
// It stores the results in the pager's matches and resets matchIdx.
func (p *Pager) Search(term string) []int {
	p.searchTerm = term
	p.matches = nil
	p.matchIdx = -1

	for i, line := range p.lines {
		plain := stripANSI(line)
		if strings.Contains(plain, term) {
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
