# Changelog

## Unreleased

### Added

- Documentation site (MkDocs Material, published to GitHub Pages) in English, with a Portuguese version: getting started, MCP clients, configuration, Windows and WSL, architecture, security model, specification, development and code review.
- Executable tutorials under `demo/tutorials/` (quickstart, temporary access, guardrails, masking), run against a real OpenBao dev server; CI fails when an output drifts from `expected/`.
- `khcall`, a minimal MCP client for debugging and for the tutorials.
- `README.pt-BR.md`, `CONTRIBUTING.md`.
- `.mcp.json` with the AurumCode gate (`aurumcode mcp`) for Claude Code.
- `Review failure reason` job: copies the AurumCode error lines of a failed review into a PR annotation.
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
