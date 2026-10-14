// Package ref parses vault:// references and {{vault://...}} placeholders.
//
// A reference points at one field of a secret in the KV:
//
//	vault://my-project/dev/API_TOKEN
//	        └──── Path ───┘ └─ Key ─┘
//
// Path is the secret's path inside the KV mount and Key is the field in it.
package ref

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Scheme prefixes every reference.
const Scheme = "vault://"

// Ref identifies one field of a secret.
type Ref struct {
	Path string // secret path, for example "my-project/dev"
	Key  string // field inside the secret, for example "API_TOKEN"
}

// String returns the canonical form vault://path/key.
func (r Ref) String() string { return Scheme + r.Path + "/" + r.Key }

var segmentRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// ErrInvalid reports a malformed reference.
var ErrInvalid = errors.New("invalid reference")

// Parse parses "vault://a/b/KEY". It requires at least one path segment and
// the key, and rejects empty segments, "." and "..".
func Parse(s string) (Ref, error) {
	if !strings.HasPrefix(s, Scheme) {
		return Ref{}, fmt.Errorf("%w: %q does not start with %s", ErrInvalid, s, Scheme)
	}
	parts := strings.Split(strings.TrimPrefix(s, Scheme), "/")
	if len(parts) < 2 {
		return Ref{}, fmt.Errorf("%w: %q needs a path and a key", ErrInvalid, s)
	}
	for _, p := range parts {
		if p == "." || p == ".." || !segmentRe.MatchString(p) {
			return Ref{}, fmt.Errorf("%w: segment %q in %q", ErrInvalid, p, s)
		}
	}
	return Ref{Path: strings.Join(parts[:len(parts)-1], "/"), Key: parts[len(parts)-1]}, nil
}

// InScope reports whether the reference is inside the scope (a path
// prefix). An empty scope accepts everything.
func (r Ref) InScope(scope string) bool {
	scope = strings.Trim(scope, "/")
	if scope == "" {
		return true
	}
	return r.Path == scope || strings.HasPrefix(r.Path, scope+"/")
}

var placeholderRe = regexp.MustCompile(`\{\{\s*(vault://[^{}\s]+)\s*\}\}`)

// HasPlaceholder reports whether the text contains any {{vault://...}}.
func HasPlaceholder(s string) bool { return placeholderRe.MatchString(s) }

// Resolver returns the value of a reference.
type Resolver func(Ref) (string, error)

// Expand replaces each {{vault://...}} with its resolved value. It returns
// the expanded text and the references used, in order of appearance.
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
