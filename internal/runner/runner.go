// Package runner runs allowed commands with secrets resolved only at execution
// time (late binding).
//
// The command runs without a shell: arguments go to the process as a list, so
// special characters in a secret are never interpreted as code. The output
// comes back with the values that were used masked.
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

// Runner defaults.
const (
	DefaultTimeout   = 5 * time.Minute
	DefaultMaxOutput = 64 * 1024 // bytes per stream (stdout and stderr)
)

// Resolver returns the value of a reference. Errors must not contain the value.
type Resolver func(context.Context, ref.Ref) (string, error)

// Runner holds the execution policy.
type Runner struct {
	Allow     map[string]bool // allowed commands, by exact name
	Scope     string          // path prefix accepted in references; empty accepts all
	Resolve   Resolver
	Timeout   time.Duration
	MaxOutput int
	// BaseEnv is the environment inherited by the child. Nil uses os.Environ().
	// BAO_*, VAULT_* and KEYHOLDER_* variables are always removed.
	BaseEnv []string
}

// Request is what the agent asks for. Secrets only enter as {{vault://...}}.
type Request struct {
	Command string            `json:"command" jsonschema:"name of an allowed command, without a path or placeholders, for example curl or aws"`
	Args    []string          `json:"args,omitempty" jsonschema:"arguments; may contain {{vault://path/KEY}}"`
	Env     map[string]string `json:"env,omitempty" jsonschema:"extra variables for the process; values may contain {{vault://path/KEY}}"`
}

// Result is what goes back to the agent. It never contains resolved values.
type Result struct {
	ExitCode  int      `json:"exit_code"`
	Stdout    string   `json:"stdout"`
	Stderr    string   `json:"stderr"`
	RefsUsed  []string `json:"refs_used"`
	Truncated bool     `json:"truncated,omitempty"`
	TimedOut  bool     `json:"timed_out,omitempty"`
}

// Policy errors.
var (
	ErrNotAllowed = errors.New("command not in the allow list")
	ErrOutOfScope = errors.New("reference outside the project scope")
)

var envNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Run validates the request, resolves the references, runs the command and
// masks the output.
func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	if req.Command == "" || ref.HasPlaceholder(req.Command) || strings.ContainsAny(req.Command, `/\`) {
		return Result{}, fmt.Errorf("%w: give only the command name", ErrNotAllowed)
	}
	if !r.Allow[req.Command] {
		return Result{}, fmt.Errorf("%w: %q (allowed: %s)", ErrNotAllowed, req.Command, strings.Join(r.allowed(), ", "))
	}
	path, err := exec.LookPath(req.Command)
	if err != nil {
		return Result{}, fmt.Errorf("command %q not found in PATH", req.Command)
	}

	var (
		m      mask.Masker
		seen   = map[string]bool{}
		used   []string
		values = map[string]string{} // per-reference cache within this run
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
			return Result{}, fmt.Errorf("variable name not allowed: %q", k)
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
	// cmd.Stdin stays nil: the child reads from the null device.

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
		// System message, masked to be safe.
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

// filterEnv removes the vault token and configuration from the inherited
// environment.
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

// capped keeps up to max bytes and discards the rest.
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
