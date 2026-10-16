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
	"path/filepath"
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

// Session resolves the references of one run. Finish is called once the
// command ends (or the run is refused after something was resolved); it
// releases what the run created, such as dynamic credential leases, and
// returns notes for the agent.
type Session interface {
	Resolve(context.Context, ref.Ref) (string, error)
	Finish(context.Context) []string
}

// Runner holds the execution policy.
type Runner struct {
	Allow   map[string]bool // allowed commands, by exact name
	Scope   string          // path prefix accepted in vault:// references; empty accepts all
	Dynamic map[string]bool // bao:// paths that may be read, by exact path
	Resolve Resolver        // used when Begin is nil; vault:// only
	// Begin opens the resolution session of one run. Nil uses Resolve.
	Begin     func() Session
	Timeout   time.Duration
	MaxOutput int
	// BaseEnv is the environment inherited by the child. Nil uses os.Environ().
	// BAO_*, VAULT_* and KEYHOLDER_* variables are always removed.
	BaseEnv []string
}

// Request is what the agent asks for. Secrets only enter as placeholders.
type Request struct {
	Command     string            `json:"command" jsonschema:"name of an allowed command, without a path or placeholders, for example curl or aws"`
	Args        []string          `json:"args,omitempty" jsonschema:"arguments; may contain {{vault://path/KEY}} or {{bao://path#field}}, and {{file:VAR}} for the path of a file declared in files"`
	Env         map[string]string `json:"env,omitempty" jsonschema:"extra variables for the process; values may contain placeholders, including {{file:VAR}}"`
	Files       map[string]string `json:"files,omitempty" jsonschema:"variables that receive the path of a temporary file holding the value; values may contain placeholders; the file is deleted after the run; {{file:VAR}} in args or env is that same path"`
	Credentials []string          `json:"credentials,omitempty" jsonschema:"cloud credentials set the way each CLI expects: aws:<path> (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN) or gcp:<path> (gcloud access token file), for example aws:aws/creds/deploy"`
}

// Result is what goes back to the agent. It never contains resolved values.
type Result struct {
	ExitCode  int      `json:"exit_code"`
	Stdout    string   `json:"stdout"`
	Stderr    string   `json:"stderr"`
	RefsUsed  []string `json:"refs_used"`
	Truncated bool     `json:"truncated,omitempty"`
	TimedOut  bool     `json:"timed_out,omitempty"`
	Notes     []string `json:"notes,omitempty"`
}

// Policy errors.
var (
	ErrNotAllowed      = errors.New("command not in the allow list")
	ErrOutOfScope      = errors.New("reference outside the project scope")
	ErrDynamicNotAllow = errors.New("dynamic path not allowed: it must be listed in KEYHOLDER_DYNAMIC")
)

var envNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Run validates the request, resolves the references, runs the command and
// masks the output.
func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	if req.Command == "" || ref.HasPlaceholder(req.Command) || filePlaceholderRe.MatchString(req.Command) || strings.ContainsAny(req.Command, `/\`) {
		return Result{}, fmt.Errorf("%w: give only the command name", ErrNotAllowed)
	}
	if !r.Allow[req.Command] {
		return Result{}, fmt.Errorf("%w: %q (allowed: %s)", ErrNotAllowed, req.Command, strings.Join(r.allowed(), ", "))
	}
	path, err := exec.LookPath(req.Command)
	if err != nil {
		return Result{}, fmt.Errorf("command %q not found in PATH", req.Command)
	}

	sess := r.session()
	finished := false
	finish := func() []string {
		if finished {
			return nil
		}
		finished = true
		// The run's own context may already be cancelled (timeout): cleanup
		// gets a fresh one.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		return sess.Finish(fctx)
	}
	defer finish()

	var (
		m      mask.Masker
		seen   = map[string]bool{}
		used   []string
		values = map[string]string{} // per-reference cache within this run
	)
	resolve := func(rf ref.Ref) (string, error) {
		switch {
		case rf.Dynamic && !r.Dynamic[rf.Path]:
			return "", fmt.Errorf("%w (%s)", ErrDynamicNotAllow, rf.Path)
		case !rf.Dynamic && !rf.InScope(r.Scope):
			return "", fmt.Errorf("%w (%s)", ErrOutOfScope, r.Scope)
		}
		key := rf.String()
		if v, ok := values[key]; ok {
			return v, nil
		}
		v, err := sess.Resolve(ctx, rf)
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

	vars, files, optional, drop, err := expandCredentials(req)
	if err != nil {
		return Result{}, err
	}

	// Temporary files come first: {{file:VAR}} in args and env needs their paths.
	filePaths := map[string]string{}
	if len(files) > 0 {
		dir, err := os.MkdirTemp("", "keyholder-")
		if err != nil {
			return Result{}, fmt.Errorf("temporary directory: %w", err)
		}
		defer os.RemoveAll(dir) // best effort: the files hold resolved values
		for i, k := range sortedKeys(files) {
			if !envNameRe.MatchString(k) || isReserved(k) {
				return Result{}, fmt.Errorf("variable name not allowed: %q", k)
			}
			if filePlaceholderRe.MatchString(files[k]) {
				return Result{}, fmt.Errorf("files[%s]: a file cannot contain {{file:...}}", k)
			}
			v, _, err := ref.Expand(files[k], resolve)
			if err != nil {
				return Result{}, err
			}
			fp := filepath.Join(dir, fmt.Sprintf("value-%d", i))
			if err := os.WriteFile(fp, []byte(v), 0o600); err != nil {
				return Result{}, fmt.Errorf("write temporary file for %s: %w", k, err)
			}
			filePaths[k] = fp
		}
	}

	args := make([]string, len(req.Args))
	for i, a := range req.Args {
		if args[i], _, err = ref.Expand(a, resolve); err != nil {
			return Result{}, err
		}
		if args[i], err = expandFilePaths(args[i], filePaths); err != nil {
			return Result{}, err
		}
	}
	env := dropPrefixes(filterEnv(r.BaseEnv), drop)
	for _, k := range sortedKeys(vars) {
		if !envNameRe.MatchString(k) || isReserved(k) {
			return Result{}, fmt.Errorf("variable name not allowed: %q", k)
		}
		v, _, err := ref.Expand(vars[k], resolve)
		if err != nil {
			if optional[k] {
				continue // for example AWS_SESSION_TOKEN with IAM user credentials
			}
			return Result{}, err
		}
		if v == "" && optional[k] {
			continue
		}
		if v, err = expandFilePaths(v, filePaths); err != nil {
			return Result{}, err
		}
		env = append(env, k+"="+v)
	}
	for _, k := range sortedKeys(filePaths) {
		env = append(env, k+"="+filePaths[k])
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
	res.Notes = finish()
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

// session returns the resolver for one run.
func (r *Runner) session() Session {
	if r.Begin != nil {
		return r.Begin()
	}
	return funcSession(r.Resolve)
}

// funcSession adapts a plain Resolver; it has nothing to release.
type funcSession Resolver

func (f funcSession) Resolve(ctx context.Context, rf ref.Ref) (string, error) {
	if f == nil {
		return "", errors.New("no resolver configured")
	}
	return f(ctx, rf)
}

func (funcSession) Finish(context.Context) []string { return nil }

// expandCredentials merges Env, Files and the Credentials presets into the
// variables to set. It returns which variables may be left out when their
// field is absent, and the inherited variable prefixes to drop so a preset
// never mixes with credentials from the parent environment.
func expandCredentials(req Request) (vars, files map[string]string, optional map[string]bool, drop []string, err error) {
	vars = map[string]string{}
	files = map[string]string{}
	optional = map[string]bool{}
	for k, v := range req.Env {
		vars[k] = v
	}
	for k, v := range req.Files {
		if _, dup := vars[k]; dup {
			return nil, nil, nil, nil, fmt.Errorf("variable %s is set in both env and files", k)
		}
		files[k] = v
	}
	set := func(m map[string]string, k, v string) error {
		_, inVars := vars[k]
		_, inFiles := files[k]
		if inVars || inFiles {
			return fmt.Errorf("variable %s is set twice (credentials and env or files)", k)
		}
		m[k] = v
		return nil
	}
	for _, c := range req.Credentials {
		kind, path, ok := strings.Cut(c, ":")
		if !ok || path == "" {
			return nil, nil, nil, nil, fmt.Errorf("credentials entry %q: use aws:<path> or gcp:<path>", c)
		}
		if err := ref.ValidPath(path); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("credentials entry %q: %w", c, err)
		}
		ph := func(field string) string {
			return "{{" + ref.Ref{Path: path, Key: field, Dynamic: true}.String() + "}}"
		}
		switch kind {
		case "aws":
			for k, f := range map[string]string{"AWS_ACCESS_KEY_ID": "access_key", "AWS_SECRET_ACCESS_KEY": "secret_key", "AWS_SESSION_TOKEN": "session_token"} {
				if err := set(vars, k, ph(f)); err != nil {
					return nil, nil, nil, nil, err
				}
			}
			optional["AWS_SESSION_TOKEN"] = true
			drop = append(drop, "AWS_")
		case "gcp":
			if err := set(files, "CLOUDSDK_AUTH_ACCESS_TOKEN_FILE", ph("token")); err != nil {
				return nil, nil, nil, nil, err
			}
			if err := set(vars, "GOOGLE_OAUTH_ACCESS_TOKEN", ph("token")); err != nil {
				return nil, nil, nil, nil, err
			}
			drop = append(drop, "CLOUDSDK_AUTH_", "GOOGLE_OAUTH_ACCESS_TOKEN", "GOOGLE_APPLICATION_CREDENTIALS")
		default:
			return nil, nil, nil, nil, fmt.Errorf("credentials entry %q: unknown kind %q (use aws or gcp)", c, kind)
		}
	}
	return vars, files, optional, drop, nil
}

// dropPrefixes removes inherited variables whose names start with a prefix.
func dropPrefixes(env []string, prefixes []string) []string {
	if len(prefixes) == 0 {
		return env
	}
	out := env[:0]
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		keep := true
		for _, p := range prefixes {
			if strings.HasPrefix(strings.ToUpper(name), p) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, kv)
		}
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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

// filePlaceholderRe matches {{file:VAR}}: the path of the temporary file
// that `files` created for VAR, for tools that take a credential file as an
// argument (ssh -i, curl --key, curl -K) rather than from a variable.
var filePlaceholderRe = regexp.MustCompile(`\{\{\s*file:([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// expandFilePaths replaces each {{file:VAR}} with the path of VAR's
// temporary file. An unknown VAR is an error: nothing passes through as text.
func expandFilePaths(s string, paths map[string]string) (string, error) {
	var firstErr error
	out := filePlaceholderRe.ReplaceAllStringFunc(s, func(m string) string {
		name := filePlaceholderRe.FindStringSubmatch(m)[1]
		p, ok := paths[name]
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("{{file:%s}}: no such entry in files", name)
			}
			return m
		}
		return p
	})
	return out, firstErr
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
