// Package ref interpreta referências vault:// e placeholders {{vault://...}}.
//
// Uma referência aponta para um campo de um segredo no KV:
//
//	vault://meu-projeto/dev/API_TOKEN
//	        └──── Path ────┘ └─ Key ─┘
//
// Path é o caminho do segredo dentro do mount do KV e Key é o campo dentro dele.
package ref

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Scheme é o prefixo de toda referência.
const Scheme = "vault://"

// Ref identifica um campo de um segredo.
type Ref struct {
	Path string // caminho do segredo, por exemplo "meu-projeto/dev"
	Key  string // campo dentro do segredo, por exemplo "API_TOKEN"
}

// String devolve a forma canônica vault://path/key.
func (r Ref) String() string { return Scheme + r.Path + "/" + r.Key }

var segmentRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// ErrInvalid indica uma referência malformada.
var ErrInvalid = errors.New("referência inválida")

// Parse interpreta "vault://a/b/KEY". Exige ao menos um segmento de caminho e
// a chave, e recusa segmentos vazios, "." e "..".
func Parse(s string) (Ref, error) {
	if !strings.HasPrefix(s, Scheme) {
		return Ref{}, fmt.Errorf("%w: %q não começa com %s", ErrInvalid, s, Scheme)
	}
	parts := strings.Split(strings.TrimPrefix(s, Scheme), "/")
	if len(parts) < 2 {
		return Ref{}, fmt.Errorf("%w: %q precisa de caminho e chave", ErrInvalid, s)
	}
	for _, p := range parts {
		if p == "." || p == ".." || !segmentRe.MatchString(p) {
			return Ref{}, fmt.Errorf("%w: segmento %q em %q", ErrInvalid, p, s)
		}
	}
	return Ref{Path: strings.Join(parts[:len(parts)-1], "/"), Key: parts[len(parts)-1]}, nil
}

// InScope informa se a referência fica dentro do escopo (prefixo de caminho).
// Escopo vazio aceita tudo.
func (r Ref) InScope(scope string) bool {
	scope = strings.Trim(scope, "/")
	if scope == "" {
		return true
	}
	return r.Path == scope || strings.HasPrefix(r.Path, scope+"/")
}

var placeholderRe = regexp.MustCompile(`\{\{\s*(vault://[^{}\s]+)\s*\}\}`)

// HasPlaceholder informa se o texto contém algum {{vault://...}}.
func HasPlaceholder(s string) bool { return placeholderRe.MatchString(s) }

// Resolver devolve o valor de uma referência.
type Resolver func(Ref) (string, error)

// Expand troca cada {{vault://...}} pelo valor resolvido. Devolve o texto
// expandido e as referências usadas, na ordem em que aparecem.
func Expand(s string, resolve Resolver) (string, []Ref, error) {
	var (
		used     []Ref
		firstErr error
	)
	out := placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		if firstErr != nil {
			return m
		}
		raw := placeholderRe.FindStringSubmatch(m)[1]
		r, err := Parse(raw)
		if err != nil {
			firstErr = err
			return m
		}
		v, err := resolve(r)
		if err != nil {
			firstErr = fmt.Errorf("%s: %w", r, err)
			return m
		}
		used = append(used, r)
		return v
	})
	if firstErr != nil {
		return "", nil, firstErr
	}
	return out, used, nil
}
