// Package bao talks to OpenBao: it opens the session, resolves references in
// KV v2, lists names and reports status. Nothing here returns values to the
// MCP layer except Resolve, which only the runner calls.
package bao

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/openbao/openbao/api/v2"

	"keyholder-mcp/internal/ref"
)

// Environment variables read at startup.
const (
	EnvWrappingToken = "BAO_WRAPPING_TOKEN" // wrapped, single-use token; preferred
	EnvToken         = "BAO_TOKEN"          // plain token; accepted for local testing
)

// Store is the open session with OpenBao.
type Store struct {
	c     *api.Client
	mount string
	scope string
	// expiresAt is the local expiry estimate taken from the unwrap response.
	// Zero when the token came from BAO_TOKEN.
	expiresAt time.Time
}

// Open creates the client (BAO_ADDR and TLS settings come from the
// environment, as with the bao CLI), unwraps the token when a wrapped one is
// given, and removes the token variables from this process's environment so
// nothing started below it inherits them.
func Open(ctx context.Context, mount, scope string) (*Store, error) {
	wrapped := strings.TrimSpace(os.Getenv(EnvWrappingToken))
	defer func() {
		for _, k := range []string{EnvWrappingToken, EnvToken, "VAULT_TOKEN"} {
			_ = os.Unsetenv(k) // best effort: Unsetenv only fails on invalid names
		}
	}()

	c, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("OpenBao client: %w", err)
	}
	if wrapped != "" {
		c.ClearToken()
		sec, err := c.Logical().UnwrapWithContext(ctx, wrapped)
		if err != nil {
			return nil, fmt.Errorf("%w (already used or expired?)", cleanErr("unwrap token", err))
		}
		if sec == nil || sec.Auth == nil || sec.Auth.ClientToken == "" {
			return nil, errors.New("the wrapped token does not contain a client token")
		}
		c.SetToken(sec.Auth.ClientToken)
		st := &Store{c: c, mount: strings.Trim(mount, "/"), scope: strings.Trim(scope, "/")}
		if sec.Auth.LeaseDuration > 0 {
			st.expiresAt = time.Now().Add(time.Duration(sec.Auth.LeaseDuration) * time.Second)
		}
		return st, nil
	}
	if c.Token() == "" {
		return nil, fmt.Errorf("no token: set %s (preferred) or %s", EnvWrappingToken, EnvToken)
	}
	return &Store{c: c, mount: strings.Trim(mount, "/"), scope: strings.Trim(scope, "/")}, nil
}

// ErrDenied means HTTP 403: the token expired or was revoked, or the policy
// does not allow the path.
var ErrDenied = errors.New("access denied by the vault: token expired or revoked, or the policy does not allow this path; renew access outside the chat")

// cleanErr turns client HTTP errors into short messages, without the URL or
// the response body.
func cleanErr(what string, err error) error {
	var re *api.ResponseError
	if errors.As(err, &re) {
		if re.StatusCode == 403 {
			return fmt.Errorf("%s: %w", what, ErrDenied)
		}
		return fmt.Errorf("%s: vault returned HTTP %d", what, re.StatusCode)
	}
	return fmt.Errorf("%s: %w", what, err)
}

// ErrNotFound means the reference does not exist (secret or field).
var ErrNotFound = errors.New("reference not found")

// Resolve reads the secret field from KV v2. Only the runner should call it.
func (s *Store) Resolve(ctx context.Context, r ref.Ref) (string, error) {
	sec, err := s.c.KVv2(s.mount).Get(ctx, r.Path)
	if err != nil {
		if errors.Is(err, api.ErrSecretNotFound) {
			return "", ErrNotFound
		}
		return "", cleanErr("read "+r.Path, err)
	}
	v, ok := sec.Data[r.Key]
	if !ok {
		return "", ErrNotFound
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("field %s is not text", r.Key)
	}
	return str, nil
}

// Entry describes a reference without its value.
type Entry struct {
	Ref     string `json:"ref"`
	Version int    `json:"version,omitempty"`
}

// MaxEntries caps the listing so a whole KV is never walked.
const MaxEntries = 500

// List walks the scope and returns the references (path and field name). It
// reads the secrets to learn the field names but discards the values.
func (s *Store) List(ctx context.Context) ([]Entry, error) {
	var out []Entry
	var walk func(prefix string) error
	walk = func(prefix string) error {
		if len(out) >= MaxEntries {
			return nil
		}
		sec, err := s.c.Logical().ListWithContext(ctx, s.mount+"/metadata/"+prefix)
		if err != nil {
			return cleanErr("list "+prefix, err)
		}
		if sec == nil || sec.Data == nil {
			// It may be a leaf secret (the scope points straight at it).
			return s.addSecret(ctx, strings.TrimSuffix(prefix, "/"), &out)
		}
		keys, _ := sec.Data["keys"].([]any)
		for _, k := range keys {
			name, _ := k.(string)
			if strings.HasSuffix(name, "/") {
				if err := walk(prefix + name); err != nil {
					return err
				}
				continue
			}
			if err := s.addSecret(ctx, prefix+name, &out); err != nil {
				return err
			}
		}
		return nil
	}
	start := ""
	if s.scope != "" {
		start = s.scope + "/"
	}
	if err := walk(start); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out, nil
}

func (s *Store) addSecret(ctx context.Context, path string, out *[]Entry) error {
	if path == "" {
		return nil
	}
	sec, err := s.c.KVv2(s.mount).Get(ctx, path)
	if err != nil {
		if errors.Is(err, api.ErrSecretNotFound) {
			return nil
		}
		return cleanErr("read "+path, err)
	}
	ver := 0
	if sec.VersionMetadata != nil {
		ver = sec.VersionMetadata.Version
	}
	for k := range sec.Data {
		if len(*out) >= MaxEntries {
			break
		}
		*out = append(*out, Entry{Ref: ref.Ref{Path: path, Key: k}.String(), Version: ver})
	}
	return nil
}

// Status summarizes the vault and the session, without the token or accessor.
type Status struct {
	Addr         string   `json:"addr"`
	Initialized  bool     `json:"initialized"`
	Sealed       bool     `json:"sealed"`
	Version      string   `json:"server_version"`
	Mount        string   `json:"kv_mount"`
	Scope        string   `json:"scope"`
	TTLSeconds   int64    `json:"token_ttl_seconds"`
	ExpireTime   string   `json:"token_expire_time,omitempty"`
	Renewable    bool     `json:"token_renewable"`
	Policies     []string `json:"token_policies"`
	SessionError string   `json:"session_error,omitempty"`
}

// Status queries sys/health and the token itself.
func (s *Store) Status(ctx context.Context) (Status, error) {
	st := Status{Addr: s.c.Address(), Mount: s.mount, Scope: s.scope}
	h, err := s.c.Sys().HealthWithContext(ctx)
	if err != nil {
		return st, fmt.Errorf("sys/health: %w", err)
	}
	st.Initialized, st.Sealed, st.Version = h.Initialized, h.Sealed, h.Version

	sec, err := s.c.Auth().Token().LookupSelfWithContext(ctx)
	if err != nil {
		// Without lookup-self, fall back to the local estimate from the unwrap.
		switch {
		case !s.expiresAt.IsZero() && time.Now().After(s.expiresAt):
			st.SessionError = "token expired; renew access outside the chat"
		case !s.expiresAt.IsZero():
			st.TTLSeconds = int64(time.Until(s.expiresAt) / time.Second)
			st.ExpireTime = s.expiresAt.UTC().Format(time.RFC3339)
			st.SessionError = "the policy does not allow auth/token/lookup-self; TTL estimated locally"
		default:
			st.SessionError = cleanErr("look up token", err).Error()
		}
		return st, nil
	}
	if ttl, err := sec.TokenTTL(); err == nil {
		st.TTLSeconds = int64(ttl / time.Second)
	}
	if exp, ok := sec.Data["expire_time"].(string); ok {
		st.ExpireTime = exp
	}
	// Both helpers only fail on malformed lookup data; the zero value is a
	// safe report in that case.
	st.Renewable, _ = sec.TokenIsRenewable()
	st.Policies, _ = sec.TokenPolicies()
	return st, nil
}
