package mask

import "testing"

func TestMask(t *testing.T) {
	var m Masker
	m.Add("abcd1234", "vault://p/TOKEN")
	m.Add("abcd1234XYZ", "vault://p/LONG")
	m.Add("abc", "vault://p/SHORT") // ignored: shorter than MinLen

	got := m.Mask("t=abcd1234 l=abcd1234XYZ c=abc")
	want := "t=[hidden:vault://p/TOKEN] l=[hidden:vault://p/LONG] c=abc"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestMaskEmpty(t *testing.T) {
	var m Masker
	if got := m.Mask("nothing to do"); got != "nothing to do" {
		t.Fatalf("got %q", got)
	}
}
