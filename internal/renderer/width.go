package renderer

import (
	"os"

	"github.com/mattn/go-runewidth"
)

// widthCond is the rune-width condition shared by all renderer width
// calculations.
//
// go-runewidth infers EastAsianWidth from the locale (LANG/LC_ALL), which
// makes East Asian *Ambiguous* characters — including the box-drawing runes
// (─ │ ┌ ┬ …) used for tables and code boxes — count as width 2 under a
// Japanese/Chinese/Korean locale. Virtually all modern terminals render
// ambiguous characters (and box drawing in particular) as ONE cell regardless
// of locale, so locale-based inference breaks border alignment exactly for
// CJK users.
//
// We therefore pin EastAsianWidth to false (narrow) by default. Users whose
// terminal is configured to render ambiguous characters wide can override
// this with the standard RUNEWIDTH_EASTASIAN=1 environment variable.
var widthCond = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	if os.Getenv("RUNEWIDTH_EASTASIAN") == "" {
		c.EastAsianWidth = false
	}
	return c
}()

// runeW returns the display width of a rune using the shared condition.
func runeW(r rune) int {
	return widthCond.RuneWidth(r)
}

// stringW returns the display width of a string using the shared condition.
func stringW(s string) int {
	return widthCond.StringWidth(s)
}
