// Package bao fala com o OpenBao: abre a sessão, resolve referências no KV v2,
// lista nomes e informa o estado. Nenhuma função daqui devolve valores para o
// MCP, exceto Resolve, que só o runner chama.
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

// Variáveis de ambiente lidas na inicialização.
const (
	EnvWrappingToken = "BAO_WRAPPING_TOKEN" // token embrulhado (uso único); preferido
	EnvToken         = "BAO_TOKEN"          // token comum; aceito para a prova de conceito
)

// Store é a sessão aberta com o OpenBao.
type Store struct {
	c     *api.Client
	mount string
	scope string
	// expiresAt é a estimativa local de expiração, vinda do desembrulho.
	// Zero quando o token veio por BAO_TOKEN.
	expiresAt time.Time
}

// Open cria o cliente (BAO_ADDR e TLS vêm do ambiente, como no CLI bao),
// desembrulha o token se houver um embrulhado e apaga as variáveis de token do
// ambiente deste processo, para que nada abaixo dele as herde.
func Open(ctx context.Context, mount, scope string) (*Store, error) {
	wrapped := strings.TrimSpace(os.Getenv(EnvWrappingToken))
	defer func() {
		for _, k := range []string{EnvWrappingToken, EnvToken, "VAULT_TOKEN"} {
			_ = os.Unsetenv(k)
		}
	}()

	c, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("cliente OpenBao: %w", err)
	}
	if wrapped != "" {
		c.ClearToken()
		sec, err := c.Logical().UnwrapWithContext(ctx, wrapped)
		if err != nil {
			return nil, fmt.Errorf("desembrulhar o token: %w (já foi usado ou expirou?)", err)
		}
		if sec == nil || sec.Auth == nil || sec.Auth.ClientToken == "" {
			return nil, errors.New("o token embrulhado não contém um token de cliente")
		}
		c.SetToken(sec.Auth.ClientToken)
		st := &Store{c: c, mount: strings.Trim(mount, "/"), scope: strings.Trim(scope, "/")}
		if sec.Auth.LeaseDuration > 0 {
			st.expiresAt = time.Now().Add(time.Duration(sec.Auth.LeaseDuration) * time.Second)
		}
		return st, nil
	}
	if c.Token() == "" {
		return nil, fmt.Errorf("nenhum token: defina %s (preferido) ou %s", EnvWrappingToken, EnvToken)
	}
	return &Store{c: c, mount: strings.Trim(mount, "/"), scope: strings.Trim(scope, "/")}, nil
}

// ErrDenied indica 403: token expirado, revogado ou política sem permissão.
var ErrDenied = errors.New("acesso negado pelo cofre: token expirado ou revogado, ou a política não permite este caminho; renove o acesso fora do chat")

// cleanErr troca erros HTTP do cliente por mensagens curtas, sem URL nem corpo.
func cleanErr(what string, err error) error {
	var re *api.ResponseError
	if errors.As(err, &re) {
		if re.StatusCode == 403 {
			return fmt.Errorf("%s: %w", what, ErrDenied)
		}
		return fmt.Errorf("%s: cofre respondeu HTTP %d", what, re.StatusCode)
	}
	return fmt.Errorf("%s: %w", what, err)
}

// ErrNotFound indica referência inexistente (segredo ou campo).
var ErrNotFound = errors.New("referência não encontrada")

// Resolve lê o campo do segredo no KV v2. Só o runner deve chamar.
func (s *Store) Resolve(ctx context.Context, r ref.Ref) (string, error) {
	sec, err := s.c.KVv2(s.mount).Get(ctx, r.Path)
	if err != nil {
		if errors.Is(err, api.ErrSecretNotFound) {
			return "", ErrNotFound
		}
		return "", cleanErr("ler "+r.Path, err)
	}
	v, ok := sec.Data[r.Key]
	if !ok {
		return "", ErrNotFound
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("campo %s não é texto", r.Key)
	}
	return str, nil
}

// Entry descreve uma referência sem o valor.
type Entry struct {
	Ref     string `json:"ref"`
	Version int    `json:"version,omitempty"`
}

// MaxEntries limita a listagem para não varrer um KV inteiro.
const MaxEntries = 500

// List percorre o escopo e devolve as referências (caminho e nome do campo).
// Lê os segredos para saber os nomes dos campos, mas descarta os valores.
func (s *Store) List(ctx context.Context) ([]Entry, error) {
	var out []Entry
	var walk func(prefix string) error
	walk = func(prefix string) error {
		if len(out) >= MaxEntries {
			return nil
		}
		sec, err := s.c.Logical().ListWithContext(ctx, s.mount+"/metadata/"+prefix)
		if err != nil {
			return cleanErr("listar "+prefix, err)
		}
		if sec == nil || sec.Data == nil {
			// Pode ser um segredo folha (o escopo aponta direto para ele).
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
		return cleanErr("ler "+path, err)
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

// Status resume o cofre e a sessão, sem token nem accessor.
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

// Status consulta sys/health e o próprio token.
func (s *Store) Status(ctx context.Context) (Status, error) {
	st := Status{Addr: s.c.Address(), Mount: s.mount, Scope: s.scope}
	h, err := s.c.Sys().HealthWithContext(ctx)
	if err != nil {
		return st, fmt.Errorf("sys/health: %w", err)
	}
	st.Initialized, st.Sealed, st.Version = h.Initialized, h.Sealed, h.Version

	sec, err := s.c.Auth().Token().LookupSelfWithContext(ctx)
	if err != nil {
		// Sem lookup-self, usa a estimativa local do desembrulho.
		switch {
		case !s.expiresAt.IsZero() && time.Now().After(s.expiresAt):
			st.SessionError = "token expirado; renove o acesso fora do chat"
		case !s.expiresAt.IsZero():
			st.TTLSeconds = int64(time.Until(s.expiresAt) / time.Second)
			st.ExpireTime = s.expiresAt.UTC().Format(time.RFC3339)
			st.SessionError = "a política não permite auth/token/lookup-self; TTL estimado localmente"
		default:
			st.SessionError = cleanErr("consultar o token", err).Error()
		}
		return st, nil
	}
	if ttl, err := sec.TokenTTL(); err == nil {
		st.TTLSeconds = int64(ttl / time.Second)
	}
	if exp, ok := sec.Data["expire_time"].(string); ok {
		st.ExpireTime = exp
	}
	st.Renewable, _ = sec.TokenIsRenewable()
	st.Policies, _ = sec.TokenPolicies()
	return st, nil
}
