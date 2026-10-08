// Package textutil implements the small piece of Unicode text handling the
// emulator needs for correct Bangla editing: grapheme-cluster segmentation
// (a practical subset of UAX #29) and cursor/backspace helpers.
//
// Bangla (like Indic scripts in general) combines a base consonant with
// combining marks (Mn/Mc/Me), a virama/hasanta (্) that joins the next
// consonant, and the ZWJ/ZWNJ joiners. Treating code points as "characters"
// breaks ""ক্ষ"" into three units; treating grapheme clusters as characters
// keeps it whole for the cursor.
package textutil

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	zwj            = 0x200D
	zwnj           = 0x200C
	variationStart = 0xFE00
	variationEnd   = 0xFE0F
)

// isCombining reports whether r extends the previous grapheme cluster.
func isCombining(r rune) bool {
	return unicode.Is(unicode.Mn, r) ||
		unicode.Is(unicode.Mc, r) ||
		unicode.Is(unicode.Me, r) ||
		r == zwj || r == zwnj ||
		(r >= variationStart && r <= variationEnd)
}

// isVirama reports whether r is a virama/hasanta (joins the next consonant).
func isVirama(r rune) bool {
	switch r {
	case 0x094D, // Devanagari
		0x09CD, // Bengali
		0x0A4D, // Gurmukhi
		0x0ACD, // Gujarati
		0x0B4D, // Oriya
		0x0BCD, // Tamil
		0x0C4D, // Telugu
		0x0CCD, // Kannada
		0x0D4D, // Malayalam
		0x0DCA: // Sinhala
		return true
	}
	return false
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

// Clusters splits s into grapheme clusters.
func Clusters(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	var prev rune
	riCount := 0
	started := false

	flush := func() {
		if started {
			out = append(out, cur.String())
			cur.Reset()
		}
	}

	for _, r := range s {
		if !started {
			cur.WriteRune(r)
			started = true
			prev = r
			if isRegionalIndicator(r) {
				riCount = 1
			}
			continue
		}

		join := isCombining(r) ||
			isVirama(prev) ||
			prev == zwj
		// Keep pairs of regional indicators (flag emoji) together.
		if isRegionalIndicator(r) && isRegionalIndicator(prev) && riCount%2 == 1 {
			join = true
		}

		if join {
			cur.WriteRune(r)
		} else {
			flush()
			cur.WriteRune(r)
		}
		prev = r
		if isRegionalIndicator(r) {
			riCount++
		} else {
			riCount = 0
		}
	}
	flush()
	return out
}

// Boundaries returns the byte offsets at the start of each grapheme cluster,
// terminated by len(s).
func Boundaries(s string) []int {
	if s == "" {
		return []int{0}
	}
	out := []int{0}
	var prev rune
	riCount := 0
	for i, r := range s {
		if i == 0 {
			prev = r
			if isRegionalIndicator(r) {
				riCount = 1
			}
			continue
		}
		join := isCombining(r) || isVirama(prev) || prev == zwj ||
			(isRegionalIndicator(r) && isRegionalIndicator(prev) && riCount%2 == 1)
		if !join {
			out = append(out, i)
		}
		prev = r
		if isRegionalIndicator(r) {
			riCount++
		} else {
			riCount = 0
		}
	}
	out = append(out, len(s))
	return out
}

// Count returns the number of grapheme clusters in s.
func Count(s string) int {
	n := 0
	for range Clusters(s) {
		n++
	}
	return n
}

// DeleteLastCluster removes the final grapheme cluster from s.
func DeleteLastCluster(s string) string {
	c := Clusters(s)
	if len(c) == 0 {
		return s
	}
	return strings.Join(c[:len(c)-1], "")
}

// DeleteLastRune removes the final code point from s (the behaviour some
// input methods use for backspace).
func DeleteLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}

// PrevBoundary returns the byte offset of the cluster boundary before pos.
func PrevBoundary(s string, pos int) int {
	if pos <= 0 {
		return 0
	}
	if pos > len(s) {
		pos = len(s)
	}
	b := Boundaries(s[:pos])
	return b[len(b)-2]
}

// NextBoundary returns the byte offset of the cluster boundary after pos.
func NextBoundary(s string, pos int) int {
	if pos >= len(s) {
		return len(s)
	}
	if pos < 0 {
		pos = 0
	}
	b := Boundaries(s)
	for _, off := range b {
		if off > pos {
			return off
		}
	}
	return len(s)
}
