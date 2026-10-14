# Specification

Version 0.3, 2026-10-13. Items not yet verified are marked **[to verify]**.

## 1. Goal

Stop the manual copying of passwords, tokens, keys and URLs into chats, repositories, `.env` files and commands written by a model. The agent sees names and metadata, never values. Resolution happens only at execution time (late binding), and access expires.

MCP is the discovery and invocation interface. What protects and injects the secrets is the vault and the runner.

## 2. Honest limits

- It does not protect against a deliberately hostile agent running as your OS user.
- "Not exposed to the model" does not mean "no process ever touches the secret": the executed process receives the value. Each execution mode states its guarantee ([Security model](security.md#execution-modes)).
- Output masking is a defense, not a mathematical guarantee: an encoded value escapes.
- Out of scope: managing AWS or GCP IAM, team use, monetization.

## 3. Base decision

Build on OpenBao instead of writing a vault and daemon from scratch. Every reason below is verified in OpenBao's official documentation:

- Tokens with TTL, hard maximum (`-explicit-max-ttl`) and use limit (`-use-limit`) ([token create](https://openbao.org/docs/commands/token/create/)).
- Response wrapping: single-use token with its own TTL ([response wrapping](https://openbao.org/docs/concepts/response-wrapping/)).
- Leases with automatic revocation; revoking a token revokes the leases created with it ([lease](https://openbao.org/docs/concepts/lease/)).
- Login MFA with built-in TOTP (free), plus Duo, Okta and PingID (third-party services) ([login MFA](https://openbao.org/docs/auth/login-mfa/)).
- AWS and GCP exist as official **external** plugins ([openbao-plugins](https://github.com/openbao/openbao-plugins)), installed through a `plugin` block pinned by digest; they are not built in ([plugins](https://openbao.org/docs/plugins/)).
- Binaries for Windows, macOS and Linux; container images; web UI ([install](https://openbao.org/docs/install/)).

Alternative considered: KeePassXC with [`kdbx`](https://github.com/yarrasys/kdbx) (MIT). No server, but no documented access expiry.

## 4. Requirements

| # | Requirement | Status |
| --- | --- | --- |
| R1 | Build on an existing open source vault | Done (OpenBao) |
| R2 | Local MCP server that gives credentials to local AI agents | Done |
| R3 | Simple environment variables (AWS, GCP, LLM keys) | Done: static values, and dynamic AWS and GCP credentials through `credentials` |
| R4 | Values never appear in the chat | Done |
| R5 | Graphical interface on the server | Done (OpenBao web UI) |
| R6 | Windows, Linux and macOS | Done (CI on the three) |
| R7 | Server reachable from WSL | Documented ([Windows and WSL](windows-wsl.md)) |
| R8 | Temporary access: token with expiry and a free second factor | Done (wrapped token + TTL; TOTP in OpenBao) |
| R9 | `vault://path/KEY` references and `{{...}}` placeholders in commands | Done |
| R10 | Workspace, project and environment as OpenBao paths and policies | Done (path convention + scope) |
| R11 | Session per project and client, renewable (24 h or 72 h) | Partly: TTL per session; renewal is a new wrapped token |
| R12 | Safe argument substitution, no shell concatenation | Done |
| R13 | Audit without values | Done |
| R14 | Contribute to an upstream project | Planned |

## 5. MCP surface

Names and actions, never values. No tool returns a value in clear.

| Tool | Does | Returns | Status |
| --- | --- | --- | --- |
| `keyholder_status` | Vault and session status | Sealed or not, remaining TTL, policies | Implemented |
| `keyholder_list` | Lists the references in the project scope | `vault://` names and versions | Implemented |
| `keyholder_run` | Runs an allowed command, resolving `{{vault://...}}` in `args` and `env` at execution time | Masked output, exit code, references used | Implemented |
| `keyholder_request_access` | Asks the human for temporary access | Pending, approved or denied; never the token | Planned |

Field-by-field reference: [Configuration](configuration.md#tools).

## 6. Execution modes

- **Mode A, runner (preferred):** the agent sends a command with references; another process resolves and runs it. The value does not pass through the conversation.
- **Mode B, native tools:** AWS CLI, gcloud or ssh-agent use their own mechanisms. The value may live in the tool's process and appear in debug output; no absolute isolation is promised. Verified: `gcloud` reads an access token from the file named by `CLOUDSDK_AUTH_ACCESS_TOKEN_FILE`, which the runner writes for the run and deletes after it ([cloud tutorial](tutorials/cloud.md)).
- **Mode C, browser:** a local component fills a field. Secrecy depends on whether the agent can read the page afterwards. Out of the MVP.

An integration without safe late binding must state the limit instead of pretending secrecy.

## 7. Temporary access flow

1. You log in to OpenBao with TOTP.
2. You issue a child token with a minimal policy, TTL, optional use limit and `wrap-ttl`.
3. The wrapped token reaches the server through the MCP client's environment, outside the chat.
4. The server unwraps it and keeps the token only in memory.
5. `keyholder_run` reads the secret and injects it into the command.
6. The token expires; reads return 403 and the tools tell the agent to have access renewed. **[to verify]** whether expiry (not explicit revocation) also revokes leases created with the token; the documentation states it for revocation.

## 8. Proof of concept, verified

Against `bao server -dev` 2.7.1, recorded and re-checked by the [tutorials](tutorials/index.md):

- The destination API received the right `Authorization: Bearer` value while the agent read `[hidden:vault://my-project/dev/API_TOKEN]`.
- No stored value and no vault token appeared in any MCP response.
- A reference outside `KEYHOLDER_SCOPE` and a command outside `KEYHOLDER_ALLOW` were refused, even with a broad vault policy.
- After the TTL, `keyholder_status` reported the expiry and `keyholder_run` failed with access denied.
- Reusing a wrapped token failed to unwrap.
- `BAO_*`, `VAULT_*` and `KEYHOLDER_*` never reach the child process.
- CI builds and tests on Linux, Windows and macOS.

- A dynamic AWS credential from OpenBao's real AWS plugin ran the AWS CLI against a local AWS emulator (moto) and was deleted when the command ended.

Not yet tested: real MCP clients end to end; the GCP plugin and real AWS accounts.

## 9. Open decisions

1. How to unseal after a restart: manual Shamir or auto unseal.
2. Second factor: built-in TOTP or an external service.
3. GCP credential: service account JSON key or dynamic OAuth token.
4. Agent writes to the vault: a dedicated path, or none.
5. Token delivery: today `BAO_WRAPPING_TOKEN` in the client's environment, worthless once unwrapped. A local prompt or socket may follow.

Decided: OpenBao as the vault; Go as the language; MIT license; OpenBao inside WSL or a container on Windows.

## 10. Acceptance criteria

| # | Criterion | Status |
| --- | --- | --- |
| AC-01 | A stored token does not appear in the chat, a tool description or an MCP response | Met |
| AC-02 | Real API call with a reference: the server gets the right token, the agent only the result | Met |
| AC-03 | `aws sts get-caller-identity` and `gcloud projects list` work without the agent writing keys | Met for AWS against an emulator, in CI; gcloud token handling met; the GCP plugin not run |
| AC-04 | A project shows only the references in its scope | Met |
| AC-05 | Within a session, repeated operations need no re-approval; after expiry, renewal outside the chat | Met |
| AC-06 | Logs, MCP messages and product artifacts contain no values | Met |
| AC-07 | An integration without safe late binding states its limit | Met (documented) |
| AC-08 | Works with at least two MCP clients without changing storage | Open (documented, not run in CI) |

## 11. Roadmap

1. ~~Proof of concept and minimal MCP~~: done, released as v0.1.0.
2. ~~**Dynamic credentials:** OpenBao AWS and GCP plugins mapped to environment variables~~: done in v0.2.0. AWS `iam_user` credentials are revoked after the run; STS and GCP tokens expire on their own.
3. **`keyholder_request_access`:** the agent asks, the human approves with TOTP, a new wrapped token reaches the server.
4. **Persistent server guide:** storage, TLS and unseal.
5. **SSH and browser adapters.**
6. **Upstream:** propose "run with injected variables" and temporary access to an existing project.
