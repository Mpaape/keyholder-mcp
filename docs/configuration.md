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
| `KEYHOLDER_DYNAMIC` | empty (none) | OpenBao paths of [dynamic secrets](#dynamic-credentials) the agent may use, comma separated, by exact path: `aws/creds/deploy,gcp/roleset/viewer/token` |
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

In `keyholder_run`, a reference goes inside a placeholder, `{{vault://path/KEY}}`, anywhere inside an argument, an environment value or a file value. Several placeholders can appear in one string.

A **dynamic** reference names one field of a secret OpenBao creates when it is read, such as an AWS credential:

```text
bao://aws/creds/deploy#access_key
      └── path ─────┘ └─ field ─┘
```

The path is relative to the OpenBao root, not to the KV mount, and must be listed in `KEYHOLDER_DYNAMIC`. Within one run each path is read once, so `#access_key` and `#secret_key` come from the same credential. See [Dynamic credentials](#dynamic-credentials).

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

No input. Output: `scope`, `entries` (each with `ref` and `version`), `dynamic` (each allowed dynamic path with a `use` hint, for example `credentials: ["aws:aws/creds/deploy"]`; listing never reads it) and a `note` for the agent. It walks the scope (up to 500 entries), reading each secret to learn its field names and discarding the values.

### `keyholder_run`

Input:

| Field | Type | Meaning |
| --- | --- | --- |
| `command` | string | Name of an allowed command. No path, no placeholder |
| `args` | list of strings | Arguments; may contain `{{vault://...}}` and `{{bao://...}}` |
| `env` | map | Extra variables for the child; values may contain placeholders. Names `BAO_*`, `VAULT_*`, `KEYHOLDER_*` are refused |
| `files` | map | Variables that receive the **path** of a temporary file holding the value (mode 0600, deleted after the run), for tools that read a credential from a file |
| `credentials` | list of strings | Cloud credentials set the way each CLI expects: `aws:<path>` or `gcp:<path>`. See [Dynamic credentials](#dynamic-credentials) |

Output:

| Field | Meaning |
| --- | --- |
| `exit_code` | The child's exit code |
| `stdout`, `stderr` | Up to 64 KiB each, with every resolved value replaced by `[hidden:<reference>]` |
| `refs_used` | References resolved in this run |
| `truncated` | Output went over 64 KiB |
| `timed_out` | The run hit `KEYHOLDER_TIMEOUT` |
| `notes` | What happened after the run, for example `revoked 1 dynamic credential lease(s) after the run` |

The command runs **without a shell**: `exec` with the argument list, stdin from the null device. Pipes, redirections and `$VAR` are not interpreted, and special characters inside a secret are never treated as code.

Errors (not allowed, out of scope, missing reference, access denied) come back as tool errors with a message the agent can act on, without URLs or vault response bodies.

## Dynamic credentials

With OpenBao's [AWS](https://github.com/openbao/openbao-plugins/tree/main/secrets/aws) and [Google Cloud](https://github.com/openbao/openbao-plugins/tree/main/secrets/gcp) secrets plugins, a command can run with a cloud credential created for that run only.

| `credentials` entry | Reads | Sets in the child |
| --- | --- | --- |
| `aws:<path>`, for example `aws:aws/creds/deploy` | `access_key`, `secret_key`, `session_token` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN` when the credential type has one |
| `gcp:<path>`, for example `gcp:gcp/roleset/viewer/token` | `token` | `CLOUDSDK_AUTH_ACCESS_TOKEN_FILE` (a temporary file for gcloud) and `GOOGLE_OAUTH_ACCESS_TOKEN` |

A preset also drops the inherited variables of that cloud (`AWS_*`; `CLOUDSDK_AUTH_*`, `GOOGLE_OAUTH_ACCESS_TOKEN`, `GOOGLE_APPLICATION_CREDENTIALS`), so the fresh credential never mixes with a profile or token from the server's environment.

When the command ends, Keyholder revokes every lease the run created. What that does depends on the credential:

| Credential | After the run |
| --- | --- |
| AWS `iam_user` | Revoked: OpenBao deletes the IAM user and its key |
| AWS `assumed_role`, `federation_token`, `session_token` (STS) | Cannot be revoked; valid until the TTL of the role (AWS minimum is 15 minutes) |
| GCP roleset or account access token | Not leased by the plugin; valid until it expires (up to one hour) |

Prefer `iam_user` roles when the credential must die with the command. A lease that cannot be revoked is reported in `notes`. Full walkthrough: [Cloud credentials tutorial](tutorials/cloud.md).

## Policy

The [example policy](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) grants exactly what the three tools need for one project:

| Path | Capabilities | Used by |
| --- | --- | --- |
| `secret/data/my-project/*` | read | `keyholder_run` (resolve), `keyholder_list` (field names) |
| `secret/metadata/my-project`, `.../*` | list, read | `keyholder_list` |
| `auth/token/lookup-self` | read | `keyholder_status` (already in OpenBao's `default` policy) |

For dynamic credentials, add a second policy like [`deploy/policy-cloud.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-cloud.hcl): `read` on each path in `KEYHOLDER_DYNAMIC` and `update` on `sys/leases/revoke`, which Keyholder uses to revoke after the run.

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
