// Package mask hides known values in output text.
//
// It is a defense, not a guarantee: it only matches the exact value. An
// encoded (base64, hex, URL-encoded) or split value passes through unmasked.
package mask

import (
	"sort"
	"strings"
)

// MinLen is the shortest value that gets masked. Shorter values would wreck
// the output (a secret "a" would mask every letter a) and are weak anyway.
const MinLen = 4

// Masker replaces secret values with a label that names the reference.
type Masker struct {
	pairs []pair
}

type pair struct{ value, label string }

// Add registers a value and the label that replaces it, such as the
// reference. Values shorter than MinLen are ignored.
func (m *Masker) Add(value, label string) {
	if len(value) < MinLen {
		return
	}
	for _, p := range m.pairs {
		if p.value == value {
			return
		}
	}
	m.pairs = append(m.pairs, pair{value, "[hidden:" + label + "]"})
	// Longest values first, so a value that contains another is replaced whole.
	sort.SliceStable(m.pairs, func(i, j int) bool { return len(m.pairs[i].value) > len(m.pairs[j].value) })
}

// Mask returns the text with every registered value replaced.
func (m *Masker) Mask(s string) string {
	for _, p := range m.pairs {
		s = strings.ReplaceAll(s, p.value, p.label)
	}
	return s
}
