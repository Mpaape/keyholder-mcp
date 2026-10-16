// keyholder-mcp is an MCP server (stdio) that lets AI agents use credentials
// stored in OpenBao without ever seeing their values.
//
// Configuration comes from environment variables:
//
//	BAO_ADDR              OpenBao address (client default: https://127.0.0.1:8200)
//	BAO_WRAPPING_TOKEN    wrapped, single-use token (preferred)
//	BAO_TOKEN             plain token (accepted for local testing)
//	KEYHOLDER_KV_MOUNT    KV v2 mount (default: secret)
//	KEYHOLDER_SCOPE       project path prefix, for example my-project
//	KEYHOLDER_ALLOW       allowed commands, comma separated, for example curl,aws
//	KEYHOLDER_DYNAMIC     OpenBao paths of dynamic secrets the agent may use, comma
//	                      separated, for example aws/creds/deploy,gcp/roleset/viewer/token
//	KEYHOLDER_TIMEOUT     maximum time per run (default: 5m)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"keyholder-mcp/internal/bao"
	"keyholder-mcp/internal/ref"
	"keyholder-mcp/internal/runner"
)

var version = "0.4.1-dev"

type config struct {
	mount   string
	scope   string
	allow   map[string]bool
	dynamic []string
	timeout time.Duration
}

func loadConfig() (config, error) {
	c := config{
		mount:   envOr("KEYHOLDER_KV_MOUNT", "secret"),
		scope:   strings.Trim(os.Getenv("KEYHOLDER_SCOPE"), "/"),
		allow:   map[string]bool{},
		timeout: runner.DefaultTimeout,
	}
	for _, name := range strings.Split(os.Getenv("KEYHOLDER_ALLOW"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			c.allow[name] = true
		}
	}
	for _, p := range strings.Split(os.Getenv("KEYHOLDER_DYNAMIC"), ",") {
		if p = strings.Trim(strings.TrimSpace(p), "/"); p == "" {
			continue
		}
		if err := ref.ValidPath(p); err != nil {
			return c, fmt.Errorf("invalid KEYHOLDER_DYNAMIC entry: %w", err)
		}
		c.dynamic = append(c.dynamic, p)
	}
	if v := os.Getenv("KEYHOLDER_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return c, fmt.Errorf("invalid KEYHOLDER_TIMEOUT: %q", v)
		}
		c.timeout = d
	}
	return c, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type listOutput struct {
	Scope   string         `json:"scope"`
	Entries []bao.Entry    `json:"entries"`
	Dynamic []dynamicEntry `json:"dynamic,omitempty"`
	Note    string         `json:"note"`
}

// dynamicEntry describes an allowed dynamic path without reading it (a read
// would create a credential).
type dynamicEntry struct {
	Path string `json:"path"`
	Use  string `json:"use"`
}

// dynamicUse suggests how to use a dynamic path, from the path shapes of the
// OpenBao AWS and GCP secrets engines.
func dynamicUse(p string) string {
	parts := strings.Split(p, "/")
	n := len(parts)
	switch {
	case n >= 3 && (parts[n-2] == "creds" || parts[n-2] == "sts"):
		return `credentials: ["aws:` + p + `"]`
	case n >= 4 && parts[n-1] == "token" && (parts[n-3] == "roleset" || parts[n-3] == "static-account" || parts[n-3] == "impersonated-account"):
		return `credentials: ["gcp:` + p + `"]`
	}
	return "{{bao://" + p + "#<field>}}"
}

var errNoSession = errors.New("no session with the vault; a human must renew access outside the chat")

func main() {
	// `keyholder-mcp exec ...` launches a service with secrets in its
	// environment (exec.go); everything else is the MCP server.
	if len(os.Args) > 1 && os.Args[1] == "exec" {
		os.Exit(runExec(os.Args[2:]))
	}
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("keyholder-mcp", version)
		return
	}

	// stdout belongs to the MCP protocol; logs and the audit trail go to stderr.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Without a session the server still starts: keyholder_status explains why
	// and the other tools fail in a predictable way.
	store, openErr := bao.Open(ctx, cfg.mount, cfg.scope)
	if openErr != nil {
		log.Printf("session not opened: %v", openErr)
	}
	if cfg.scope == "" {
		log.Printf("warning: KEYHOLDER_SCOPE is empty; every reference in mount %q is reachable", cfg.mount)
	}

	dyn := map[string]bool{}
	for _, p := range cfg.dynamic {
		dyn[p] = true
	}
	run := &runner.Runner{
		Allow:   cfg.allow,
		Scope:   cfg.scope,
		Dynamic: dyn,
		Timeout: cfg.timeout,
	}
	if store != nil {
		run.Begin = func() runner.Session { return store.Begin() }
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "keyholder-mcp", Version: version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keyholder_status",
		Description: "Vault and session status: sealed or not, remaining token TTL and policies. Never returns tokens or secret values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, bao.Status, error) {
		if store == nil {
			st := bao.Probe(ctx, cfg.mount, cfg.scope)
			st.SessionError = openErr.Error()
			return nil, st, nil
		}
		st, err := store.Status(ctx)
		return nil, st, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keyholder_list",
		Description: "Lists the vault:// references in the project scope and the dynamic paths (cloud credentials) the agent may use. Returns names only, never values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listOutput, error) {
		if store == nil {
			return nil, listOutput{}, errNoSession
		}
		entries, err := store.List(ctx)
		if err != nil {
			return nil, listOutput{}, err
		}
		if entries == nil {
			entries = []bao.Entry{}
		}
		out := listOutput{
			Scope:   cfg.scope,
			Entries: entries,
			Note:    "Use {{vault://path/KEY}} in the args, env or files of keyholder_run.",
		}
		for _, p := range cfg.dynamic {
			out.Dynamic = append(out.Dynamic, dynamicEntry{Path: p, Use: dynamicUse(p)})
		}
		if len(out.Dynamic) > 0 {
			out.Note += " Dynamic paths are read when the command runs and revoked after it; use them through credentials or {{bao://path#field}}."
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "keyholder_run",
		Description: "Runs an allowed command, without a shell, replacing {{vault://path/KEY}} with the value only at execution time. " +
			"Example: command=curl, args=[\"-H\", \"Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}\", \"https://api.example.com/health\"]. " +
			"Cloud CLIs: credentials=[\"aws:aws/creds/deploy\"] sets AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and AWS_SESSION_TOKEN from one fresh credential, revoked after the run; credentials=[\"gcp:gcp/roleset/viewer/token\"] gives gcloud an access token. " +
			"files={\"KEY\": \"{{vault://...}}\"} writes the value to a temporary file, and {{file:KEY}} in args or env is that file's path, for tools that take a credential file as an argument (ssh -i, curl --key, curl -K). " +
			"The output comes back with the values masked. Masking only catches the exact value: do not encode or print secrets on purpose.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in runner.Request) (*mcp.CallToolResult, runner.Result, error) {
		if store == nil {
			return nil, runner.Result{}, errNoSession
		}
		start := time.Now()
		res, err := run.Run(ctx, in)
		// Audit: command, references and result. Never values or arguments.
		status := "ok"
		if err != nil {
			status = "error: " + err.Error()
		}
		log.Printf("audit cmd=%q refs=%v exit=%d dur=%s status=%s", in.Command, res.RefsUsed, res.ExitCode, time.Since(start).Round(time.Millisecond), status)
		return nil, res, err
	})

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("MCP server: %v", err)
	}
}
