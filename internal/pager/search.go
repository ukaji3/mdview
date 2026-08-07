package pager

import (
	"regexp"
	"strings"
)

// Regexes matching terminal escape sequences for stripping before search.
// Covering OSC/DCS/APC as well as CSI prevents search terms from matching
// invisible content such as image data (Kitty \x1b_G..., Sixel \x1bPq...,
// iTerm2 \x1b]1337;...) or OSC hyperlink URLs.
var (
	searchCSIRe = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)
	searchOSCRe = regexp.MustCompile(`\033\][^\x07\033]*(?:\x07|\033\\)`)
	searchDCSRe = regexp.MustCompile(`\033P[^\033]*\033\\`)
	searchAPCRe = regexp.MustCompile(`\033_[^\033]*\033\\`)
)

// stripANSI removes terminal escape sequences from a string so that
// search matching operates on visible text only.
func stripANSI(s string) string {
	// Fast path: no escape characters at all.
	if !strings.ContainsRune(s, '\033') {
		return s
	}
	s = searchDCSRe.ReplaceAllString(s, "")
	s = searchAPCRe.ReplaceAllString(s, "")
	s = searchOSCRe.ReplaceAllString(s, "")
	s = searchCSIRe.ReplaceAllString(s, "")
	return s
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
