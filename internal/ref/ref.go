// Package ref parses references and their {{...}} placeholders.
//
// A vault:// reference points at one field of a stored secret in the KV:
//
//	vault://my-project/dev/API_TOKEN
//	        └──── Path ───┘ └─ Key ─┘
//
// A bao:// reference points at one field of a dynamic secret: an OpenBao
// path read when the command runs, such as AWS credentials:
//
//	bao://aws/creds/deploy#access_key
//	      └─── Path ────┘ └── Key ─┘
//
// Path is relative to the KV mount for vault:// and to the OpenBao root for
// bao://; Key is the field in the response.
package ref

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Schemes.
const (
	Scheme        = "vault://" // stored secret in the KV mount
	DynamicScheme = "bao://"   // dynamic secret, read at execution time
)

// Ref identifies one field of a secret.
type Ref struct {
	Path    string // secret path, for example "my-project/dev" or "aws/creds/deploy"
	Key     string // field, for example "API_TOKEN" or "access_key"
	Dynamic bool   // true for bao:// references
}

// String returns the canonical form: vault://path/key or bao://path#key.
func (r Ref) String() string {
	if r.Dynamic {
		return DynamicScheme + r.Path + "#" + r.Key
	}
	return Scheme + r.Path + "/" + r.Key
}

// "@" is allowed because OpenBao role names may contain it.
var segmentRe = regexp.MustCompile(`^[A-Za-z0-9_.@-]+$`)

// ErrInvalid reports a malformed reference.
var ErrInvalid = errors.New("invalid reference")

// Parse parses "vault://a/b/KEY" or "bao://a/b#field". It requires a path
// and a key, and rejects empty segments, "." and "..".
func Parse(s string) (Ref, error) {
	if strings.HasPrefix(s, DynamicScheme) {
		path, key, ok := strings.Cut(strings.TrimPrefix(s, DynamicScheme), "#")
		if !ok {
			return Ref{}, fmt.Errorf("%w: %q needs #field after the path", ErrInvalid, s)
		}
		if err := ValidPath(path); err != nil {
			return Ref{}, fmt.Errorf("%w: %q", ErrInvalid, s)
		}
		if err := ValidKey(key); err != nil {
			return Ref{}, fmt.Errorf("%w: %q", ErrInvalid, s)
		}
		return Ref{Path: path, Key: key, Dynamic: true}, nil
	}
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

// ValidPath checks a secret path ("my-project/dev") with the same segment
// rules as a reference, so anything stored can later be referenced.
func ValidPath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrInvalid)
	}
	for _, p := range strings.Split(path, "/") {
		if p == "." || p == ".." || !segmentRe.MatchString(p) {
			return fmt.Errorf("%w: segment %q in %q", ErrInvalid, p, path)
		}
	}
	return nil
}

// ValidKey checks a field name with the same rule as a path segment.
func ValidKey(key string) error {
	if key == "." || key == ".." || !segmentRe.MatchString(key) {
		return fmt.Errorf("%w: key %q", ErrInvalid, key)
	}
	return nil
}

// InScope reports whether a vault:// reference is inside the scope (a path
// prefix). An empty scope accepts everything. Dynamic references are checked
// against their own allow list by the runner, not here.
func (r Ref) InScope(scope string) bool {
	scope = strings.Trim(scope, "/")
	if scope == "" {
		return true
	}
	return r.Path == scope || strings.HasPrefix(r.Path, scope+"/")
}

var placeholderRe = regexp.MustCompile(`\{\{\s*((?:vault|bao)://[^{}\s]+)\s*\}\}`)

// HasPlaceholder reports whether the text contains any {{vault://...}} or
// {{bao://...}}.
func HasPlaceholder(s string) bool { return placeholderRe.MatchString(s) }

// Resolver returns the value of a reference.
type Resolver func(Ref) (string, error)

// Expand replaces each placeholder with its resolved value. It returns
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
