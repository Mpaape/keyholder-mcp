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

func TestParseDynamic(t *testing.T) {
	r, err := Parse("bao://aws/creds/deploy#access_key")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Dynamic || r.Path != "aws/creds/deploy" || r.Key != "access_key" {
		t.Fatalf("got %+v", r)
	}
	if r.String() != "bao://aws/creds/deploy#access_key" {
		t.Errorf("String() = %q", r.String())
	}
	for _, bad := range []string{"bao://aws/creds/deploy", "bao://#k", "bao://a/../b#k", "bao://a/b#", "bao://a b#k"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) = nil error", bad)
		}
	}
}

func TestExpandDynamic(t *testing.T) {
	got, used, err := Expand("k={{bao://aws/creds/deploy#access_key}} v={{vault://p/K}}", func(r Ref) (string, error) {
		return "<" + r.Key + ">", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "k=<access_key> v=<K>" || len(used) != 2 || !used[0].Dynamic || used[1].Dynamic {
		t.Fatalf("got %q %+v", got, used)
	}
}

func TestValidPathAndKey(t *testing.T) {
	for _, p := range []string{"my-project", "my-project/dev", "a.b/c_d", "aws/creds/role@x"} {
		if err := ValidPath(p); err != nil {
			t.Errorf("ValidPath(%q) = %v", p, err)
		}
	}
	for _, p := range []string{"", "a//b", "../x", "a/./b", "a b", "a/{{x}}"} {
		if err := ValidPath(p); err == nil {
			t.Errorf("ValidPath(%q) = nil, want error", p)
		}
	}
	for _, k := range []string{"", "a/b", "..", "A B"} {
		if err := ValidKey(k); err == nil {
			t.Errorf("ValidKey(%q) = nil, want error", k)
		}
	}
}
