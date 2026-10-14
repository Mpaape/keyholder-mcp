# Configuration

`keyholder-mcp` has no config file and no flags besides `-version`. Everything comes from environment variables set by the MCP client that starts it.

## Environment variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `BAO_ADDR` | `https://127.0.0.1:8200` (OpenBao client default) | Vault address |
| `BAO_WRAPPING_TOKEN` | — | Wrapped, single-use token. **Preferred.** Unwrapped once at startup |
| `BAO_TOKEN` | — | Plain token. Accepted for local testing; prefer the wrapped token |
| `KEYHOLDER_KV_MOUNT` | `secret` | KV **v2** mount that holds the secrets |
| `KEYHOLDER_SCOPE` | empty (everything) | Path prefix of the project. References outside it are refused and not listed. Leaving it empty logs a warning |
| `KEYHOLDER_ALLOW` | empty (nothing) | Commands the agent may run, comma separated, by exact name: `curl,aws,gcloud` |
| `KEYHOLDER_TIMEOUT` | `5m` | Maximum time per run ([Go duration](https://pkg.go.dev/time#ParseDuration): `30s`, `2m`) |

TLS and other client settings follow the OpenBao CLI variables read by the official Go client (`BAO_CACERT`, `BAO_CLIENT_CERT`, `BAO_CLIENT_KEY`, `BAO_NAMESPACE`...). `VAULT_*` names are accepted as fallbacks by that client.

At startup the server removes `BAO_WRAPPING_TOKEN`, `BAO_TOKEN` and `VAULT_TOKEN` from its own environment, and every child process gets an environment without `BAO_*`, `VAULT_*` and `KEYHOLDER_*`.

!!! danger "Never allow shells or interpreters"
    `KEYHOLDER_ALLOW` must not contain `sh`, `bash`, `zsh`, `cmd`, `powershell`, `python`, `python3`, `node`, `env`, `printenv` or anything that runs code it receives. With them the agent can print or encode a secret; see [Masking and its limit](tutorials/masking.md).

## References

A reference names one field of one secret in the KV:

```text
vault://my-project/dev/API_TOKEN
        └─── path ───┘ └─ key ─┘
```

- **path**: the secret path inside the mount (`secret/my-project/dev` → `my-project/dev`).
- **key**: the field inside that secret.
- Segments use `A-Z a-z 0-9 _ . -`; empty segments, `.` and `..` are refused.
- The value must be text.

In `keyholder_run`, a reference goes inside a placeholder, `{{vault://path/KEY}}`, anywhere inside an argument or an environment value. Several placeholders can appear in one string.

## Tools

### `keyholder_status`

No input. Output:

| Field | Meaning |
| --- | --- |
| `addr` | Vault address |
| `initialized`, `sealed`, `server_version` | From `sys/health` |
| `kv_mount`, `scope` | The configuration in use |
| `token_ttl_seconds`, `token_expire_time` | Remaining life of the session token |
| `token_renewable`, `token_policies` | From the token lookup |
| `session_error` | Why there is no usable session: no token, unwrap refused, expired, or no permission to look itself up |

Never returns the token or its accessor.

### `keyholder_list`

No input. Output: `scope`, `entries` (each with `ref` and `version`) and a `note` for the agent. It walks the scope (up to 500 entries), reading each secret to learn its field names and discarding the values.

### `keyholder_run`

Input:

| Field | Type | Meaning |
| --- | --- | --- |
| `command` | string | Name of an allowed command. No path, no placeholder |
| `args` | list of strings | Arguments; may contain `{{vault://...}}` |
| `env` | map | Extra variables for the child; values may contain `{{vault://...}}`. Names `BAO_*`, `VAULT_*`, `KEYHOLDER_*` are refused |

Output:

| Field | Meaning |
| --- | --- |
| `exit_code` | The child's exit code |
| `stdout`, `stderr` | Up to 64 KiB each, with every resolved value replaced by `[hidden:<reference>]` |
| `refs_used` | References resolved in this run |
| `truncated` | Output went over 64 KiB |
| `timed_out` | The run hit `KEYHOLDER_TIMEOUT` |

The command runs **without a shell**: `exec` with the argument list, stdin from the null device. Pipes, redirections and `$VAR` are not interpreted, and special characters inside a secret are never treated as code.

Errors (not allowed, out of scope, missing reference, access denied) come back as tool errors with a message the agent can act on, without URLs or vault response bodies.

## Policy

The [example policy](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) grants exactly what the three tools need for one project:

| Path | Capabilities | Used by |
| --- | --- | --- |
| `secret/data/my-project/*` | read | `keyholder_run` (resolve), `keyholder_list` (field names) |
| `secret/metadata/my-project`, `.../*` | list, read | `keyholder_list` |
| `auth/token/lookup-self` | read | `keyholder_status` (already in OpenBao's `default` policy) |

Without `lookup-self`, `keyholder_status` estimates the TTL from the unwrap response and says so in `session_error`.

## Audit trail

Each `keyholder_run` writes one line to the server's stderr, which MCP clients keep in their server logs:

```text
2026/10/13 22:35:40 audit cmd="curl" refs=[vault://my-project/dev/API_TOKEN] exit=0 dur=8ms status=ok
```

It records the command name, the references, the exit code, the duration and the result. Never values, never arguments. OpenBao's own [audit devices](https://openbao.org/docs/audit/) record every read on the vault side.

## khcall

`khcall` is a minimal MCP client for debugging and for the tutorials:

```bash
khcall [-server keyholder-mcp] -tools
khcall [-server keyholder-mcp] <tool> ['<json arguments>'] [+wait=<duration>] [<tool> ...]
```

All calls run in one session (one server, one unwrap). `+wait=7s` pauses between calls. Exit status: 0 when every call succeeds, 1 when a tool reports an error, 2 on a usage or protocol error.
