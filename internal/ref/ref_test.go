package ref

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	r, err := Parse("vault://my-project/dev/API_TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	if r.Path != "my-project/dev" || r.Key != "API_TOKEN" {
		t.Fatalf("got %+v", r)
	}
	if r.String() != "vault://my-project/dev/API_TOKEN" {
		t.Fatalf("String() = %q", r.String())
	}
}

func TestParseInvalid(t *testing.T) {
	for _, s := range []string{
		"my-project/dev/X", // no scheme
		"vault://X",        // no path
		"vault://a//X",     // empty segment
		"vault://a/../X",   // traversal
		"vault://a/b c/X",  // space
		"vault://a/b/",     // empty key
	} {
		if _, err := Parse(s); !errors.Is(err, ErrInvalid) {
			t.Errorf("Parse(%q) err = %v, want ErrInvalid", s, err)
		}
	}
}

func TestInScope(t *testing.T) {
	r := Ref{Path: "client-a/prod", Key: "K"}
	cases := map[string]bool{"": true, "client-a": true, "client-a/prod": true, "/client-a/": true, "client": false, "client-b": false}
	for scope, want := range cases {
		if got := r.InScope(scope); got != want {
			t.Errorf("InScope(%q) = %v, want %v", scope, got, want)
		}
	}
}

func TestExpand(t *testing.T) {
	vals := map[string]string{"vault://p/dev/TOKEN": "s3cr3t", "vault://p/dev/USER": "alice"}
	resolve := func(r Ref) (string, error) {
		v, ok := vals[r.String()]
		if !ok {
			return "", errors.New("not found")
		}
		return v, nil
	}
	out, used, err := Expand("Bearer {{vault://p/dev/TOKEN}} user={{ vault://p/dev/USER }}", resolve)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Bearer s3cr3t user=alice" {
		t.Fatalf("out = %q", out)
	}
	if len(used) != 2 || used[0].Key != "TOKEN" || used[1].Key != "USER" {
		t.Fatalf("used = %+v", used)
	}
	if _, _, err := Expand("{{vault://p/dev/MISSING}}", resolve); err == nil {
		t.Fatal("expected an error for a missing reference")
	}
	if out, used, _ := Expand("no placeholder", resolve); out != "no placeholder" || used != nil {
		t.Fatalf("text without placeholder changed: %q %v", out, used)
	}
}
