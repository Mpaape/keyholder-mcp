package ref

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	r, err := Parse("vault://meu-projeto/dev/API_TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	if r.Path != "meu-projeto/dev" || r.Key != "API_TOKEN" {
		t.Fatalf("got %+v", r)
	}
	if r.String() != "vault://meu-projeto/dev/API_TOKEN" {
		t.Fatalf("String() = %q", r.String())
	}
}

func TestParseInvalid(t *testing.T) {
	for _, s := range []string{
		"meu-projeto/dev/X", // sem esquema
		"vault://X",         // sem caminho
		"vault://a//X",      // segmento vazio
		"vault://a/../X",    // travessia
		"vault://a/b c/X",   // espaço
		"vault://a/b/",      // chave vazia
	} {
		if _, err := Parse(s); !errors.Is(err, ErrInvalid) {
			t.Errorf("Parse(%q) err = %v, want ErrInvalid", s, err)
		}
	}
}

func TestInScope(t *testing.T) {
	r := Ref{Path: "cliente-a/prod", Key: "K"}
	cases := map[string]bool{"": true, "cliente-a": true, "cliente-a/prod": true, "/cliente-a/": true, "cliente": false, "cliente-b": false}
	for scope, want := range cases {
		if got := r.InScope(scope); got != want {
			t.Errorf("InScope(%q) = %v, want %v", scope, got, want)
		}
	}
}

func TestExpand(t *testing.T) {
	vals := map[string]string{"vault://p/dev/TOKEN": "s3cr3t", "vault://p/dev/USER": "ana"}
	resolve := func(r Ref) (string, error) {
		v, ok := vals[r.String()]
		if !ok {
			return "", errors.New("não encontrado")
		}
		return v, nil
	}
	out, used, err := Expand("Bearer {{vault://p/dev/TOKEN}} user={{ vault://p/dev/USER }}", resolve)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Bearer s3cr3t user=ana" {
		t.Fatalf("out = %q", out)
	}
	if len(used) != 2 || used[0].Key != "TOKEN" || used[1].Key != "USER" {
		t.Fatalf("used = %+v", used)
	}
	if _, _, err := Expand("{{vault://p/dev/NAO_EXISTE}}", resolve); err == nil {
		t.Fatal("esperava erro para referência inexistente")
	}
	if out, used, _ := Expand("sem placeholder", resolve); out != "sem placeholder" || used != nil {
		t.Fatalf("texto sem placeholder alterado: %q %v", out, used)
	}
}
