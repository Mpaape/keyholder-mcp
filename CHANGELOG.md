# Changelog

## Unreleased

## 0.4.0 - 2026-10-15

### Added

- `keyholder-mcp exec -e NAME='{{vault://path/KEY}}' -- command [args]`: launches a long-running process (a service on the host) with its secrets resolved from the vault at start and placed only in that process's environment. No `.env` file, no argument, no shell; `BAO_*`, `VAULT_*` and `KEYHOLDER_*` never reach the child; signals are forwarded and the exit code is the child's; dynamic leases are revoked on exit. Covers the host-service case: the key is not on disk and not in the agent's reach. Limit: visible to processes of the same OS user; passing it on to `docker run -e` puts it in `docker inspect`.
- `deploy/ui-theme/`: a reverse proxy (nginx) that serves OpenBao's web UI with a Keyholder theme — black-and-white, line-based, the Keyholder mark in place of the OpenBao logo. Cosmetic only: it injects one same-origin stylesheet and serves the logo under `/kh/`, respects OpenBao's Content-Security-Policy, and changes no behaviour. `docker compose -f deploy/ui-theme/compose.yml up`.
- Themed vault UI: hairline cards and inputs, monospace section labels, 2px black tab indicator.

## 0.3.0 - 2026-10-14

### Added

- Persistent vault guide (EN and pt-BR): one OpenBao node with Raft storage and TLS on localhost, initialized once, unsealed by hand with 2 of 3 keys, the root token revoked after setup, an administrator login, restarts and backups.
- `deploy/server.hcl` (server config) and `deploy/policy-admin.hcl` (administrator policy).
- Persistent vault tutorial, run in CI against `deploy/server.hcl`: init, unseal, setup without the root token, Keyholder over TLS, a restart that seals the vault, snapshot backup.
- Temporary access tutorial, case 4: token expiry revokes the leases the token created.

### Fixed

- A sealed vault was reported as a used or expired wrapped token. The session error now says the vault is sealed, that it must be unsealed and the server restarted, and that the wrapped token was not used.
- A vault certificate that is not trusted was also blamed on the token, and the message carried the request URL. It now reads `cannot reach the vault at <addr>` and points to `BAO_CACERT`.
- `keyholder_status` without a session reported an empty address and `initialized: false`. It now reads the vault's state from the unauthenticated `sys/health`.

### Verified

- Real MCP clients end to end with the v0.2.0 release binary: Claude Code and Codex (spec AC-08 met, by hand).
- Token expiry revokes the leases the token created (spec section 7, was "to verify").

### Changed

- Open decision 1 (how to unseal) is decided: manual Shamir for the one-person vault; auto unseal documented as an option, not tested.
- MCP clients page: `codex exec` needs `default_tools_approval_mode = "approve"` for the server's tools.
- Two branches only: `dev` takes every change, `main` changes only through a pull request from `dev`. Rulesets versioned in `.github/rulesets/`; CI and the docs build also run on `dev`.
- The no-AI-attribution guard is the maintainer's own: `attribution.yml` acts only on the repository owner's pull requests, issues and comments and is not a required check or a contribution rule.
- AurumCode publishes its report as a pull request comment (`publication: comments`) instead of a formal review, so the bot never approves a pull request; its verdict is only the `aurumcode/policy-gate` status.
- The AurumCode gate is a stage of `ci.yml`: it runs on a pull request only after `test` and `tutorials` pass. `aurumcode.yml` is removed.

## 0.2.0 - 2026-10-14

### Added

- Dynamic secrets: `{{bao://path#field}}` reads an OpenBao path when the command runs, once per run, and every lease it created is revoked when the command ends. Paths must be listed in `KEYHOLDER_DYNAMIC`.
- `credentials` in `keyholder_run`: `aws:<path>` sets `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and `AWS_SESSION_TOKEN`; `gcp:<path>` gives gcloud an access token file. Inherited variables of that cloud are dropped.
- `files` in `keyholder_run`: a variable receives the path of a temporary 0600 file holding the value, deleted after the run.
- `notes` in the run result, and the dynamic paths in `keyholder_list`.
- Cloud credentials tutorial (EN and pt-BR): OpenBao's real AWS plugin against a local AWS emulator (moto), and gcloud with a token file; run in CI.
- Example policy `deploy/policy-cloud.hcl`.

## 0.1.0 - 2026-10-14

### Added

- Documentation home built around one recorded request seen by the API and by the agent.
- "How it compares" page (EN and pt-BR) against mcp-vault, openbao-mcp and vault-mcp-server, with sources; linked from the README and the security model.
- Documentation site (MkDocs Material, published to GitHub Pages) in English, with a Portuguese version: getting started, MCP clients, configuration, Windows and WSL, architecture, security model, specification, development and code review.
- Executable tutorials under `demo/tutorials/` (quickstart, temporary access, guardrails, masking), run against a real OpenBao dev server; CI fails when an output drifts from `expected/`.
- `khcall`, a minimal MCP client for debugging and for the tutorials.
- `README.pt-BR.md`, `CONTRIBUTING.md`.
- `.mcp.json` with the AurumCode gate (`aurumcode mcp`) for Claude Code.
- Release workflow: `keyholder-mcp` and `khcall` for Linux, macOS and Windows (amd64, arm64) with `SHA256SUMS`, on each version tag.
- `go.sum` committed; CI runs `go mod verify`, and the AurumCode review runs `go vet`.
- No-AI-attribution guard: `.githooks/commit-msg`, a Claude Code hook and the `attribution.yml` workflow, sharing `scripts/attribution/patterns.txt`.
- CI with `gofmt`, `go vet`, `go test` and `go build` on Linux, Windows and macOS.
- MCP server in Go with `keyholder_status`, `keyholder_list` and `keyholder_run`.
- Runner with `{{vault://...}}` placeholders resolved only at execution time, without a shell, with masked output and time and output limits.
- Single-use wrapped token via `BAO_WRAPPING_TOKEN`; `BAO_*`, `VAULT_*` and `KEYHOLDER_*` kept out of the child process.
- Audit line per run on stderr: command, references, exit code, duration; never values.
- Example OpenBao policy in `deploy/policy-project-read.hcl`.
- Pull request review with AurumCode: `security`, `review` and `solid` rules; gitleaks and Semgrep.
- MIT license.

### Changed

- English is now the primary language of the repository: code, messages, rules and docs. Portuguese is secondary.
- Tools renamed: `keyholder_listar` → `keyholder_list`, `keyholder_executar` → `keyholder_run`.
- Masked values now read `[hidden:<ref>]`.
- AurumCode skill `seguranca` renamed to `security`; policy file renamed to `policy-project-read.hcl`.
- Review rule headings carry only the id (`security#sec-001`), so the rule catalog fits AurumCode's 512-token budget ([Mpaape/AurumCode#179](https://github.com/Mpaape/AurumCode/issues/179)).

### Fixed

- The AurumCode review was canceled by a concurrency deadlock with the reusable workflow's own group.
- The SARIF upload is skipped, instead of failing, when the review wrote no SARIF.
