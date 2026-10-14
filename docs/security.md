# Security model

Keyholder reduces how far a credential leak can go: values stay out of the conversation, access expires, and only named commands in one project can use them. It does **not** stop a hostile agent that already runs as your operating system user. This page says exactly where the line is.

## What is protected

| Threat | Mitigation | Where it is shown |
| --- | --- | --- |
| The value shows up in the chat, a tool description or a tool result | Tools return names and status only; output is masked | [Quickstart](tutorials/quickstart.md) |
| The value lands in logs or errors | The audit line carries command, references and exit code; vault errors are reduced to a status code | [Configuration](configuration.md#audit-trail) |
| A permanent credential leaks | Session token with TTL and hard maximum (`-explicit-max-ttl`) | [Temporary access](tutorials/access.md) |
| The token is intercepted on its way to the server | Response wrapping: single use, short TTL; a failed unwrap reveals interception | [Temporary access](tutorials/access.md) |
| The agent reaches another project's secrets | `KEYHOLDER_SCOPE` checked before any read, even if the vault policy is broad | [Guardrails](tutorials/guardrails.md) |
| The agent runs arbitrary code with the secret | Allow list by exact name; no shell; no path | [Guardrails](tutorials/guardrails.md) |
| Special characters in a secret become code | Arguments go to `exec` as a list | [Configuration](configuration.md#keyholder_run) |
| The child process reads the vault token | `BAO_*`, `VAULT_*`, `KEYHOLDER_*` removed from its environment | [Guardrails](tutorials/guardrails.md) |
| The agent redirects the child to another vault | Reserved variable names refused in `env` | [Guardrails](tutorials/guardrails.md) |
| A runaway command | Timeout and 64 KiB output cap | [Configuration](configuration.md#keyholder_run) |

## Known limits

These are not bugs; they are the edge of what this design can do.

- **The executed command receives the value.** Any real use of a credential does. The guarantee is about the conversation with the model, not the process that uses the secret.
- **Masking matches the exact value only.** An allowed interpreter can encode it and escape; the [masking tutorial](tutorials/masking.md) demonstrates it. Keep shells and interpreters out of `KEYHOLDER_ALLOW`.
- **A hostile agent with your OS user can bypass the boundary.** It could read the server's memory or start processes outside MCP. Keyholder does not run as a separate user.
- **KV secrets have no lease.** A static secret is valid until you rotate it; only the session token expires. Dynamic credentials (AWS, GCP) are on the [roadmap](spec.md#11-roadmap).

## Execution modes

| Mode | What it is | Guarantee |
| --- | --- | --- |
| **A. Runner** (this project) | The agent sends a command with references; keyholder resolves and runs it | The value never passes through the conversation |
| B. Native tools | AWS CLI, gcloud or ssh-agent read credentials through their own mechanisms | The value may sit in that tool's process and debug output; no absolute isolation is promised |
| C. Browser | A local component fills a form field | Depends on whether the agent can read the page afterwards; out of scope |

An integration without safe late binding must say so instead of pretending the credential never reached the process.

## Second factor

OpenBao's [Login MFA](https://openbao.org/docs/auth/login-mfa/) supports TOTP (built in, free) as well as Duo, Okta and PingID (third-party services). Enforce it on the login you use to issue wrapped tokens. The OpenBao documentation warns that built-in MFA does not prevent TOTP brute force by default and recommends per-client rate limits on `/sys/mfa/validate`; it allows 5 consecutive failed TOTP validations by default. Use OpenBao 2.3.2 or later: earlier versions accepted TOTP codes with whitespace (CVE-2025-55003, reported by secondary sources).

## Reporting a vulnerability

See [SECURITY.md](https://github.com/Mpaape/keyholder-mcp/blob/main/SECURITY.md): use GitHub's private vulnerability report, not a public issue.
