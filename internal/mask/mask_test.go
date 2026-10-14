package mask

import "testing"

func TestMask(t *testing.T) {
	var m Masker
	m.Add("abcd1234", "vault://p/TOKEN")
	m.Add("abcd1234XYZ", "vault://p/LONGO")
	m.Add("abc", "vault://p/CURTO") // ignorado: menor que MinLen

	got := m.Mask("t=abcd1234 l=abcd1234XYZ c=abc")
	want := "t=[oculto:vault://p/TOKEN] l=[oculto:vault://p/LONGO] c=abc"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestMaskEmpty(t *testing.T) {
	var m Masker
	if got := m.Mask("nada a fazer"); got != "nada a fazer" {
		t.Fatalf("got %q", got)
	}
}
