# Getting started

This page takes you from zero to an agent calling an API with a credential it never sees. Each step links to the reference or tutorial that explains it in depth.

## 1. Requirements

| Tool | Version | Why |
| --- | --- | --- |
| [Go](https://go.dev/dl/) | 1.25 or later | Build `keyholder-mcp` (not needed with a release binary) |
| [OpenBao](https://openbao.org/docs/install/) (`bao`) | 2.7.x tested | The vault |
| An MCP client | Claude Code, Cursor, Codex... | The agent |

## 2. Install

**Release binary.** Download the archive for your system from [Releases](https://github.com/Mpaape/keyholder-mcp/releases) (Linux, macOS and Windows, amd64 and arm64). Each holds `keyholder-mcp` and `khcall`. Check it before extracting:

```bash
sha256sum --check --ignore-missing SHA256SUMS
```

**From source:**

```bash
git clone https://github.com/Mpaape/keyholder-mcp.git
cd keyholder-mcp
go test ./...
go build -o keyholder-mcp ./cmd/keyholder-mcp     # Windows: -o keyholder-mcp.exe
go build -o khcall ./cmd/khcall                   # optional debugging client
./keyholder-mcp -version
```

Put both binaries somewhere on your `PATH`.

## 3. Start a vault

For a first try, the OpenBao **dev** server is enough. It keeps everything in memory, listens on `127.0.0.1:8200` without TLS and loses all data when it stops: never use it with real secrets.

```bash
bao server -dev -dev-root-token-id=root-dev
```

For real use, run a persistent OpenBao with TLS: [Persistent vault](server.md) sets one up step by step, and [Windows and WSL](windows-wsl.md) says where to run it.

## 4. Store a secret and a policy

In another terminal, as the vault administrator:

```bash
export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put -mount=secret my-project/dev API_TOKEN=your-api-token
bao policy write my-project-read deploy/policy-project-read.hcl
```

The [policy](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) only allows reading and listing `secret/my-project/*`.

## 5. Issue a wrapped token

```bash
bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token
```

The printed value is a **wrapped** token: it can be unwrapped once, within 5 minutes, and the token inside lasts 8 hours at most. Give it to `keyholder-mcp` through `BAO_WRAPPING_TOKEN`. Why this is safer: [Temporary access](tutorials/access.md).

## 6. Connect your agent

Add `keyholder-mcp` to your MCP client with these variables. Exact steps for Claude Code, Cursor and Codex: [MCP clients](clients.md).

| Variable | Example |
| --- | --- |
| `BAO_ADDR` | `http://127.0.0.1:8200` |
| `BAO_WRAPPING_TOKEN` | the value from step 5 |
| `KEYHOLDER_SCOPE` | `my-project` |
| `KEYHOLDER_ALLOW` | `curl,aws,gcloud` |

## 7. Ask the agent

> List the credentials you have, then call https://api.example.com/health with the API token.

The agent calls `keyholder_list`, sees `vault://my-project/dev/API_TOKEN`, and calls `keyholder_run` with `{{vault://my-project/dev/API_TOKEN}}` in a header. The response it reads has the value masked.

## Check without an agent

`khcall` is a minimal MCP client for debugging. It starts the server, calls tools in one session and prints JSON:

```bash
export BAO_ADDR=http://127.0.0.1:8200 KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl
BAO_WRAPPING_TOKEN=<wrapped token> khcall keyholder_status keyholder_list
```

The [quickstart tutorial](tutorials/quickstart.md) shows the full, real output.

## Next

- [Tutorials](tutorials/index.md): every behavior, with real outputs.
- [Configuration](configuration.md): every variable, tool and field.
- [Security model](security.md): what is protected and what is not.
