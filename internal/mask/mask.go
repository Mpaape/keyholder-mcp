// Package mask oculta valores conhecidos em textos de saída.
//
// É uma defesa, não uma garantia: só casa o valor exato. Um valor codificado
// (base64, hex, URL-encoded) ou fatiado passa sem ser mascarado.
package mask

import (
	"sort"
	"strings"
)

// MinLen é o menor valor mascarado. Valores mais curtos destruiriam a saída
// (um segredo "a" mascararia toda letra a) e são fracos de qualquer forma.
const MinLen = 4

// Masker troca valores secretos por um rótulo com a referência.
type Masker struct {
	pairs []pair
}

type pair struct{ value, label string }

// Add registra um valor e o rótulo que o substitui, por exemplo a referência.
// Valores menores que MinLen são ignorados.
func (m *Masker) Add(value, label string) {
	if len(value) < MinLen {
		return
	}
	for _, p := range m.pairs {
		if p.value == value {
			return
		}
	}
	m.pairs = append(m.pairs, pair{value, "[oculto:" + label + "]"})
	// Valores mais longos primeiro, para um valor que contém outro ser trocado inteiro.
	sort.SliceStable(m.pairs, func(i, j int) bool { return len(m.pairs[i].value) > len(m.pairs[j].value) })
}

// Mask devolve o texto com todos os valores registrados trocados.
func (m *Masker) Mask(s string) string {
	for _, p := range m.pairs {
		s = strings.ReplaceAll(s, p.value, p.label)
	}
	return s
}
