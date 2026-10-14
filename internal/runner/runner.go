// Package runner executa comandos permitidos com segredos resolvidos só na
// execução (late binding).
//
// O comando roda sem shell: os argumentos vão como lista para o processo, então
// caracteres especiais de um segredo nunca são interpretados como código.
// A saída volta com os valores usados mascarados.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"keyholder-mcp/internal/mask"
	"keyholder-mcp/internal/ref"
)

// Padrões do runner.
const (
	DefaultTimeout   = 5 * time.Minute
	DefaultMaxOutput = 64 * 1024 // bytes por fluxo (stdout e stderr)
)

// Resolver devolve o valor de uma referência. Erros não podem conter o valor.
type Resolver func(context.Context, ref.Ref) (string, error)

// Runner guarda a política de execução.
type Runner struct {
	Allow     map[string]bool // comandos permitidos, pelo nome exato
	Scope     string          // prefixo de caminho aceito nas referências; vazio aceita tudo
	Resolve   Resolver
	Timeout   time.Duration
	MaxOutput int
	// BaseEnv é o ambiente herdado pelo filho. Nil usa os.Environ().
	// Variáveis BAO_*, VAULT_* e KEYHOLDER_* são sempre removidas.
	BaseEnv []string
}

// Request é o que o agente pede. Segredos entram só como {{vault://...}}.
type Request struct {
	Command string            `json:"command" jsonschema:"nome do comando permitido, sem caminho e sem placeholders, por exemplo curl ou aws"`
	Args    []string          `json:"args,omitempty" jsonschema:"argumentos; podem conter {{vault://caminho/CHAVE}}"`
	Env     map[string]string `json:"env,omitempty" jsonschema:"variáveis extras para o processo; valores podem conter {{vault://caminho/CHAVE}}"`
}

// Result é o que volta ao agente. Nunca contém valores resolvidos.
type Result struct {
	ExitCode  int      `json:"exit_code"`
	Stdout    string   `json:"stdout"`
	Stderr    string   `json:"stderr"`
	RefsUsed  []string `json:"refs_used"`
	Truncated bool     `json:"truncated,omitempty"`
	TimedOut  bool     `json:"timed_out,omitempty"`
}

// Erros de política.
var (
	ErrNotAllowed = errors.New("comando fora da lista permitida")
	ErrOutOfScope = errors.New("referência fora do escopo do projeto")
)

var envNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Run valida o pedido, resolve as referências, executa e mascara a saída.
func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	if req.Command == "" || ref.HasPlaceholder(req.Command) || strings.ContainsAny(req.Command, `/\`) {
		return Result{}, fmt.Errorf("%w: informe só o nome do comando", ErrNotAllowed)
	}
	if !r.Allow[req.Command] {
		return Result{}, fmt.Errorf("%w: %q (permitidos: %s)", ErrNotAllowed, req.Command, strings.Join(r.allowed(), ", "))
	}
	path, err := exec.LookPath(req.Command)
	if err != nil {
		return Result{}, fmt.Errorf("comando %q não encontrado no PATH", req.Command)
	}

	var (
		m      mask.Masker
		seen   = map[string]bool{}
		used   []string
		values = map[string]string{} // cache por referência dentro desta execução
	)
	resolve := func(rf ref.Ref) (string, error) {
		if !rf.InScope(r.Scope) {
			return "", fmt.Errorf("%w (%s)", ErrOutOfScope, r.Scope)
		}
		key := rf.String()
		if v, ok := values[key]; ok {
			return v, nil
		}
		v, err := r.Resolve(ctx, rf)
		if err != nil {
			return "", err
		}
		values[key] = v
		m.Add(v, key)
		if !seen[key] {
			seen[key] = true
			used = append(used, key)
		}
		return v, nil
	}

	args := make([]string, len(req.Args))
	for i, a := range req.Args {
		if args[i], _, err = ref.Expand(a, resolve); err != nil {
			return Result{}, err
		}
	}
	env := filterEnv(r.BaseEnv)
	names := make([]string, 0, len(req.Env))
	for k := range req.Env {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if !envNameRe.MatchString(k) || isReserved(k) {
			return Result{}, fmt.Errorf("nome de variável não permitido: %q", k)
		}
		v, _, err := ref.Expand(req.Env[k], resolve)
		if err != nil {
			return Result{}, err
		}
		env = append(env, k+"="+v)
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	limit := r.MaxOutput
	if limit <= 0 {
		limit = DefaultMaxOutput
	}
	stdout, stderr := &capped{max: limit}, &capped{max: limit}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = stdout, stderr
	// cmd.Stdin fica nil: o filho lê de um dispositivo nulo.

	res := Result{RefsUsed: used}
	if res.RefsUsed == nil {
		res.RefsUsed = []string{}
	}
	runErr := cmd.Run()
	res.Stdout = m.Mask(stdout.String())
	res.Stderr = m.Mask(stderr.String())
	res.Truncated = stdout.truncated || stderr.truncated
	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
	}
	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
	case errors.As(runErr, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	default:
		// Mensagem do sistema, mascarada por segurança.
		return res, errors.New(m.Mask(runErr.Error()))
	}
	return res, nil
}

func (r *Runner) allowed() []string {
	out := make([]string, 0, len(r.Allow))
	for k := range r.Allow {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func isReserved(name string) bool {
	u := strings.ToUpper(name)
	return strings.HasPrefix(u, "BAO_") || strings.HasPrefix(u, "VAULT_") || strings.HasPrefix(u, "KEYHOLDER_")
}

// filterEnv remove do ambiente herdado o token e a configuração do cofre.
func filterEnv(base []string) []string {
	if base == nil {
		base = os.Environ()
	}
	out := make([]string, 0, len(base))
	for _, kv := range base {
		name, _, _ := strings.Cut(kv, "=")
		if !isReserved(name) {
			out = append(out, kv)
		}
	}
	return out
}

// capped guarda até max bytes e descarta o resto.
type capped struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); room < len(p) {
		if room > 0 {
			c.buf.Write(p[:room])
		}
		c.truncated = true
		return len(p), nil
	}
	return c.buf.Write(p)
}

func (c *capped) String() string { return c.buf.String() }
