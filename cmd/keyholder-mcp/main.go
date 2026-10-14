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
	"keyholder-mcp/internal/runner"
)

var version = "0.1.0-dev"

type config struct {
	mount   string
	scope   string
	allow   map[string]bool
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
	Scope   string      `json:"scope"`
	Entries []bao.Entry `json:"entries"`
	Note    string      `json:"note"`
}

var errNoSession = errors.New("no session with the vault; a human must renew access outside the chat")

func main() {
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

	run := &runner.Runner{
		Allow:   cfg.allow,
		Scope:   cfg.scope,
		Timeout: cfg.timeout,
	}
	if store != nil {
		run.Resolve = store.Resolve
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "keyholder-mcp", Version: version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keyholder_status",
		Description: "Vault and session status: sealed or not, remaining token TTL and policies. Never returns tokens or secret values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, bao.Status, error) {
		if store == nil {
			return nil, bao.Status{Mount: cfg.mount, Scope: cfg.scope, SessionError: openErr.Error()}, nil
		}
		st, err := store.Status(ctx)
		return nil, st, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keyholder_list",
		Description: "Lists the vault:// references available in the project scope. Returns names only, never values.",
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
		return nil, listOutput{
			Scope:   cfg.scope,
			Entries: entries,
			Note:    "Use {{vault://path/KEY}} in the args or env of keyholder_run.",
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "keyholder_run",
		Description: "Runs an allowed command, without a shell, replacing {{vault://path/KEY}} with the value only at execution time. " +
			"Example: command=curl, args=[\"-H\", \"Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}\", \"https://api.example.com/health\"]. " +
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
