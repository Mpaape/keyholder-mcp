package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"keyholder-mcp/internal/bao"
	"keyholder-mcp/internal/ref"
)

// execUsage documents the exec subcommand: a launcher for a long-running
// process (a service on the host) whose secrets come from the vault at start
// and live only in that process's environment. Nothing is written to disk.
const execUsage = `usage: keyholder-mcp exec [-e NAME=TEMPLATE]... -- command [args...]

Runs command with NAME set to TEMPLATE, where {{vault://path/KEY}} and
{{bao://path#field}} are resolved from the vault at start. The values go to
the child's environment only: no file, no argument, no shell. BAO_*, VAULT_*
and KEYHOLDER_* never reach the child. Signals are forwarded; the exit code
is the child's. Dynamic leases are revoked when the child exits.

  keyholder-mcp exec -e API_KEY='{{vault://my-project/prod/API_KEY}}' -- ./server --port 8080

Limits: the value is visible to any process of the same OS user (for example
through /proc). Passing it on to docker run -e puts it in docker inspect.`

type envList []string

func (e *envList) String() string     { return strings.Join(*e, ",") }
func (e *envList) Set(v string) error { *e = append(*e, v); return nil }

// runExec is the exec subcommand. It returns the process exit code.
func runExec(args []string) int {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.LUTC)

	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var specs envList
	fs.Var(&specs, "e", "NAME=TEMPLATE; repeatable")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, execUsage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cmdline := fs.Args()
	if len(cmdline) == 0 || len(specs) == 0 {
		fs.Usage()
		return 2
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Print(err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := bao.Open(ctx, cfg.mount, cfg.scope)
	if err != nil {
		log.Printf("session not opened: %v", err)
		return 2
	}
	dyn := map[string]bool{}
	for _, p := range cfg.dynamic {
		dyn[p] = true
	}
	run := store.Begin()
	defer func() {
		for _, n := range run.Finish(context.Background()) {
			log.Print(n)
		}
	}()

	resolve := guardedResolver(ctx, cfg.scope, dyn, run.Resolve)
	env, used, err := buildExecEnv(os.Environ(), specs, resolve)
	if err != nil {
		log.Print(err)
		return 2
	}

	child := exec.Command(cmdline[0], cmdline[1:]...)
	child.Env = env
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	// Signals from here on belong to the child. They are captured before
	// Start, so none can kill the launcher in between, and forwarded as
	// received once the child runs, so the service shuts down its own way.
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	start := time.Now()
	if err := child.Start(); err != nil {
		log.Printf("start %q: %v", cmdline[0], err)
		return 2
	}
	stop() // the context no longer cancels on a signal; the child handles it
	go func() {
		for s := range sigs {
			if err := child.Process.Signal(s); err != nil {
				log.Printf("forward %v to pid %d: %v", s, child.Process.Pid, err)
			}
		}
	}()
	code := exitCode(child.Wait())
	// Audit: command, references, exit code, duration. Never values.
	log.Printf("audit exec cmd=%q refs=%v pid=%d exit=%d dur=%s", cmdline[0], used, child.Process.Pid, code, time.Since(start).Round(time.Millisecond))
	return code
}

// exitCode maps the result of Wait to the launcher's exit code: the child's
// own code; 128 + signal number when a signal killed it (the shell
// convention, so a supervisor sees the same thing); 1 for any other failure.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return 1
	}
	if code := ee.ExitCode(); code >= 0 {
		return code
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return 1
}

// guardedResolver applies the same policy as keyholder_run before touching
// the vault: a vault:// reference must be inside KEYHOLDER_SCOPE, and a
// bao:// path must be listed in KEYHOLDER_DYNAMIC. Only then is inner called.
func guardedResolver(ctx context.Context, scope string, dyn map[string]bool, inner func(context.Context, ref.Ref) (string, error)) ref.Resolver {
	return func(r ref.Ref) (string, error) {
		if r.Dynamic {
			if !dyn[r.Path] {
				return "", fmt.Errorf("dynamic path not in KEYHOLDER_DYNAMIC: %s", r.Path)
			}
		} else if !r.InScope(scope) {
			return "", fmt.Errorf("reference outside KEYHOLDER_SCOPE %q", scope)
		}
		return inner(ctx, r)
	}
}

// buildExecEnv returns base without the vault variables, plus one entry per
// NAME=TEMPLATE with its placeholders resolved. Names must be plain
// identifiers and never reserved. It returns the references used.
func buildExecEnv(base []string, specs []string, resolve ref.Resolver) ([]string, []ref.Ref, error) {
	var used []ref.Ref
	set := map[string]string{}
	for i, s := range specs {
		name, tmpl, ok := strings.Cut(s, "=")
		if !ok || name == "" || strings.ContainsAny(name, " \t=") {
			// Never echo the spec: a malformed one may carry a literal secret.
			return nil, nil, fmt.Errorf("-e entry %d: expects NAME=TEMPLATE with a plain NAME", i+1)
		}
		if isReservedName(name) {
			return nil, nil, fmt.Errorf("%s is reserved", name)
		}
		if !ref.HasPlaceholder(tmpl) {
			return nil, nil, fmt.Errorf("%s has no {{vault://...}} placeholder; exec is for secrets only", name)
		}
		v, refs, err := ref.Expand(tmpl, resolve)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}
		set[name] = v
		used = append(used, refs...)
	}
	env := make([]string, 0, len(base)+len(set))
	for _, kv := range base {
		name, _, _ := strings.Cut(kv, "=")
		if isReservedName(name) {
			continue
		}
		if _, overridden := set[name]; overridden {
			continue
		}
		env = append(env, kv)
	}
	for name, v := range set {
		env = append(env, name+"="+v)
	}
	return env, used, nil
}

func isReservedName(name string) bool {
	u := strings.ToUpper(name)
	return strings.HasPrefix(u, "BAO_") || strings.HasPrefix(u, "VAULT_") || strings.HasPrefix(u, "KEYHOLDER_")
}
