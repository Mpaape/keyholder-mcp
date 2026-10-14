// khcall is a minimal MCP client for keyholder-mcp. It starts the server over
// stdio, calls one or more tools in that single session and prints each result
// as JSON. It exists for debugging and for the executable tutorials; agents
// use their own MCP client.
//
//	khcall [-server keyholder-mcp] -tools
//	khcall [-server keyholder-mcp] <tool> ['<json arguments>'] [+wait=<duration>] [<tool> ...]
//
// "+wait=<duration>" (for example +wait=6s) pauses between two calls of the
// same session, to show what happens when the token expires mid-session.
//
// One session means one server process, so a wrapped token is unwrapped once
// for all the calls. The server inherits khcall's environment (BAO_ADDR,
// BAO_WRAPPING_TOKEN, KEYHOLDER_*). Exit status: 0 when every call succeeds,
// 1 when a tool reports an error, 2 on a usage or protocol error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	os.Exit(run())
}

func run() int {
	server := flag.String("server", "keyholder-mcp", "server command to start")
	tools := flag.Bool("tools", false, "list the server's tools and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: khcall [-server cmd] -tools | <tool> ['<json arguments>'] [<tool> ...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	calls, err := parseCalls(flag.Args())
	if err != nil || (!*tools && len(calls) == 0) {
		if err != nil {
			fmt.Fprintln(os.Stderr, "khcall:", err)
		}
		flag.Usage()
		return 2
	}

	ctx := context.Background()
	cmd := exec.Command(*server)
	cmd.Stderr = os.Stderr // the server's log and audit trail
	client := mcp.NewClient(&mcp.Implementation{Name: "khcall"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "khcall: connect:", err)
		return 2
	}
	defer session.Close()

	if *tools {
		res, err := session.ListTools(ctx, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "khcall: list tools:", err)
			return 2
		}
		names := make([]string, 0, len(res.Tools))
		for _, t := range res.Tools {
			names = append(names, t.Name)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Println(n)
		}
		return 0
	}

	status := 0
	for _, c := range calls {
		if c.wait > 0 {
			time.Sleep(c.wait)
			continue
		}
		if len(calls) > 1 {
			fmt.Println("### " + c.name)
		}
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: c.name, Arguments: c.args})
		if err != nil {
			fmt.Fprintln(os.Stderr, "khcall: call:", err)
			return 2
		}
		if res.IsError {
			for _, content := range res.Content {
				if t, ok := content.(*mcp.TextContent); ok {
					fmt.Println("error:", t.Text)
				}
			}
			status = 1
			continue
		}
		out, err := json.MarshalIndent(res.StructuredContent, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "khcall: encode result:", err)
			return 2
		}
		fmt.Println(string(out))
	}
	return status
}

type call struct {
	name string
	args map[string]any
	wait time.Duration
}

// parseCalls reads "<tool> [<json object>]" pairs: an argument that starts
// with "{" is the JSON of the tool before it.
func parseCalls(argv []string) ([]call, error) {
	var calls []call
	for _, a := range argv {
		if strings.HasPrefix(strings.TrimSpace(a), "{") {
			if len(calls) == 0 || calls[len(calls)-1].wait > 0 {
				return nil, fmt.Errorf("JSON arguments before any tool name")
			}
			if err := json.Unmarshal([]byte(a), &calls[len(calls)-1].args); err != nil {
				return nil, fmt.Errorf("arguments of %s must be a JSON object: %w", calls[len(calls)-1].name, err)
			}
			continue
		}
		if d, ok := strings.CutPrefix(a, "+wait="); ok {
			w, err := time.ParseDuration(d)
			if err != nil || w <= 0 {
				return nil, fmt.Errorf("invalid wait %q", a)
			}
			calls = append(calls, call{wait: w})
			continue
		}
		calls = append(calls, call{name: a, args: map[string]any{}})
	}
	return calls, nil
}
