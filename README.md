# Keyholder MCP

[![CI](https://github.com/Mpaape/keyholder-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/Mpaape/keyholder-mcp/actions/workflows/ci.yml)
[![Docs](https://github.com/Mpaape/keyholder-mcp/actions/workflows/docs.yml/badge.svg)](https://mpaape.github.io/keyholder-mcp/)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Let AI agents use your credentials without ever seeing them.**

[Português](README.pt-BR.md) · [Documentation](https://mpaape.github.io/keyholder-mcp/) · [Tutorials](https://mpaape.github.io/keyholder-mcp/tutorials/)

Keyholder MCP is an [MCP](https://modelcontextprotocol.io) server over [OpenBao](https://openbao.org). The agent sees credential **names** and asks to run commands with them; the value is filled in only when the command runs (late binding), and the output comes back with the value masked. Access is temporary: a wrapped, single-use token with a TTL.

```text
agent asks:   curl -H "Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}" https://api.example.com/health
agent reads:  api says: hello, caller with header Authorization: Bearer [hidden:vault://my-project/dev/API_TOKEN]
```

> Status: v0.3.0. Tested end to end against OpenBao 2.7.1; every tutorial is a real, re-checked run.

## Tools

| Tool | Does | Returns |
| --- | --- | --- |
| `keyholder_status` | Vault and session status | Sealed or not, remaining TTL, policies; never the token |
| `keyholder_list` | References in the project scope | `vault://` names only |
| `keyholder_run` | Runs an allowed command with `{{vault://...}}` in `args`, `env` or `files`, and cloud `credentials` made for the run | Exit code, masked stdout and stderr, references used |

## Quick start

Download `keyholder-mcp` (and the `khcall` debugging client) for Linux, macOS or Windows from [Releases](https://github.com/Mpaape/keyholder-mcp/releases) and check it against `SHA256SUMS`, or build it with Go 1.25+. You also need [OpenBao](https://openbao.org/docs/install/).

```bash
go build -o keyholder-mcp ./cmd/keyholder-mcp     # or use a release binary

bao server -dev -dev-root-token-id=root-dev        # in-memory, never with real secrets: see Persistent vault

export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put -mount=secret my-project/dev API_TOKEN=your-api-token
bao policy write my-project-read deploy/policy-project-read.hcl
bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token
```

Then add `keyholder-mcp` to your MCP client with `BAO_ADDR`, `BAO_WRAPPING_TOKEN`, `KEYHOLDER_SCOPE=my-project` and `KEYHOLDER_ALLOW=curl`. Full walkthrough: [Getting started](https://mpaape.github.io/keyholder-mcp/getting-started/). Client setup for Claude Code, Cursor and Codex: [MCP clients](https://mpaape.github.io/keyholder-mcp/clients/).

## Cloud credentials

With OpenBao's AWS or GCP secrets plugin, `credentials: ["aws:aws/creds/deploy"]` gives the AWS CLI a key created for that run and revoked when it ends; `gcp:` gives gcloud an access token file. See [Cloud credentials](https://mpaape.github.io/keyholder-mcp/tutorials/cloud/).

## Guarantees and limits

- Values never pass through MCP messages; the executed command does receive them, as any real use of a credential does.
- Commands run without a shell, only if named in `KEYHOLDER_ALLOW`, and only with references inside `KEYHOLDER_SCOPE`.
- `BAO_*`, `VAULT_*` and `KEYHOLDER_*` never reach the child process.
- Masking matches the exact value only. Never allow shells or interpreters (`sh`, `bash`, `python`, `env`).
- It does not protect against a hostile agent running as your own OS user.

Details: [Security model](https://mpaape.github.io/keyholder-mcp/security/).

## How it compares

Other MCP servers for OpenBao and Vault ([mcp-vault](https://github.com/Cloud-Temple/mcp-vault), [openbao-mcp](https://github.com/Knuckles-Team/openbao-mcp), HashiCorp's [vault-mcp-server](https://github.com/hashicorp/vault-mcp-server)) give the agent tools to read and manage the vault, so the model can receive secret values. Keyholder has no tool that reads a value: the agent only uses credentials, through commands, and reads masked output. Side by side, with sources: [How it compares](https://mpaape.github.io/keyholder-mcp/comparison/).

## Documentation

| Page | Content |
| --- | --- |
| [Getting started](https://mpaape.github.io/keyholder-mcp/getting-started/) | From zero to an agent using a credential |
| [Tutorials](https://mpaape.github.io/keyholder-mcp/tutorials/) | Quickstart, temporary access, guardrails, masking, cloud, persistent vault; real outputs |
| [Persistent vault](https://mpaape.github.io/keyholder-mcp/server/) | OpenBao with data on disk, TLS and unsealing, for real secrets |
| [Configuration](https://mpaape.github.io/keyholder-mcp/configuration/) | Every variable, tool and field |
| [Security model](https://mpaape.github.io/keyholder-mcp/security/) | What is protected and what is not |
| [Architecture](https://mpaape.github.io/keyholder-mcp/architecture/) | Components, access flow, packages |
| [Specification](https://mpaape.github.io/keyholder-mcp/spec/) | Requirements, decisions, acceptance criteria, roadmap |
| [Code review](https://mpaape.github.io/keyholder-mcp/review/) | AurumCode rules for security, review and SOLID |

The site sources are in [`docs/`](docs/) (English) and [`docs/pt-BR/`](docs/pt-BR/) (Portuguese).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Every pull request is reviewed by [AurumCode](https://github.com/Mpaape/AurumCode) with the rules in [`.aurumcode/skills/`](.aurumcode/skills/). Security issues: [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE).
