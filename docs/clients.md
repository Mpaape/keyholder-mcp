# MCP clients

`keyholder-mcp` speaks MCP over **stdio**, so any MCP client can start it. This page shows Claude Code, Cursor and Codex.

## Keep the token out of config files

Every client below can read the wrapped token from your shell environment instead of a file. Export it in the terminal that starts the client:

```bash
export BAO_WRAPPING_TOKEN="$(bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token)"
```

Even if it ends up in a file, a wrapped token is single use: once `keyholder-mcp` unwraps it, the value is worthless. Each server start needs a fresh one, and that is how access is renewed outside the chat.

## Claude Code

With the CLI ([official docs](https://code.claude.com/docs/en/mcp)):

```bash
claude mcp add \
  --env BAO_ADDR=http://127.0.0.1:8200 \
  --env KEYHOLDER_SCOPE=my-project \
  --env KEYHOLDER_ALLOW=curl,aws,gcloud \
  --transport stdio --scope project \
  keyholder -- keyholder-mcp
```

The server name must not come right after an `--env`, or the CLI reads it as another `KEY=value`; keep `--transport` or `--scope` in between. With `--scope project` the entry is written to `.mcp.json`; add the `BAO_WRAPPING_TOKEN` line shown below so the token comes from your environment.

In `.mcp.json` at the project root, Claude Code expands `${VAR}` from your environment:

```json
{
  "mcpServers": {
    "keyholder": {
      "type": "stdio",
      "command": "keyholder-mcp",
      "env": {
        "BAO_ADDR": "http://127.0.0.1:8200",
        "BAO_WRAPPING_TOKEN": "${BAO_WRAPPING_TOKEN}",
        "KEYHOLDER_SCOPE": "my-project",
        "KEYHOLDER_ALLOW": "curl,aws,gcloud"
      }
    }
  }
}
```

Check it with `claude mcp get keyholder` or `/mcp` inside Claude Code.

## Cursor

In `.cursor/mcp.json` (project) or `~/.cursor/mcp.json` (global) ([official docs](https://cursor.com/docs/mcp)); Cursor expands `${env:NAME}`:

```json
{
  "mcpServers": {
    "keyholder": {
      "type": "stdio",
      "command": "keyholder-mcp",
      "env": {
        "BAO_ADDR": "http://127.0.0.1:8200",
        "BAO_WRAPPING_TOKEN": "${env:BAO_WRAPPING_TOKEN}",
        "KEYHOLDER_SCOPE": "my-project",
        "KEYHOLDER_ALLOW": "curl,aws,gcloud"
      }
    }
  }
}
```

## Codex

In `~/.codex/config.toml` ([official docs](https://developers.openai.com/codex/mcp)); `env_vars` forwards variables from your environment:

```toml
[mcp_servers.keyholder]
command = "keyholder-mcp"
env_vars = ["BAO_WRAPPING_TOKEN"]

[mcp_servers.keyholder.env]
BAO_ADDR = "http://127.0.0.1:8200"
KEYHOLDER_SCOPE = "my-project"
KEYHOLDER_ALLOW = "curl,aws,gcloud"
```

Or with the CLI:

```bash
codex mcp add keyholder --env BAO_ADDR=http://127.0.0.1:8200 --env KEYHOLDER_SCOPE=my-project \
  --env KEYHOLDER_ALLOW=curl,aws,gcloud -- keyholder-mcp
```

## Other clients

Any client that starts a stdio server with a command and environment variables works: the command is `keyholder-mcp`, with no arguments, and the variables are in [Configuration](configuration.md#environment-variables).

## What the agent sees

| Tool | Returns |
| --- | --- |
| `keyholder_status` | Vault and session status, TTL, policies; never the token |
| `keyholder_list` | `vault://` reference names in the scope |
| `keyholder_run` | Exit code, masked stdout and stderr, references used |

Inputs and outputs field by field: [Configuration → Tools](configuration.md#tools).

!!! note "Verified"
    The commands and file formats on this page come from each client's official documentation, linked above. The tutorials exercise the server with `khcall`; the clients themselves were not run in CI.
