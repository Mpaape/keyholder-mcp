---
name: review
version: 1
paths: ["cmd/**", "internal/**"]
---
General Go code review rules of Keyholder MCP.

## REV-001
severity: error
**Errors are handled and carry context.** No error ignored with `_` without a comment that justifies it. Errors are wrapped with `fmt.Errorf("context: %w", err)` so `errors.Is`/`errors.As` work, without exposing secrets (see security#sec-002).

## REV-002
severity: error
**New behavior comes with a test.** A behavior change in `internal/` brings a test that would fail without it. A fixed bug brings a regression test. Tests do not depend on an external network or a shell; the runner uses the test binary itself as the child process.

## REV-003
**Context is propagated.** Functions that do I/O take `context.Context` as the first parameter and pass it on (`ReadWithContext`, `CommandContext`). Do not create `context.Background()` in the middle of a call that already has a context.

## REV-004
severity: error
**stdout belongs to the MCP protocol.** The server speaks MCP over stdio: nothing may write to `os.Stdout` except the SDK. Logs and the audit trail go to `os.Stderr` through `log`.

## REV-005
severity: error
**No panic on a request path.** Tool handlers and the runner return errors instead of calling `panic`, `log.Fatal` or `os.Exit`. `log.Fatal` only at startup, in `main`.

## REV-006
**Windows, Linux and macOS.** No hard-coded `/` paths, `sh`, POSIX signals or system commands in code and tests. Use `filepath`, `os.PathListSeparator` and `exec.LookPath`.

## REV-007
**Messages to the agent are actionable.** Errors returned by tools say what happened and the next step (for example, renew access outside the chat), without internal URLs or HTTP response bodies.

## REV-008
**Formatted and vetted code.** Code passes `gofmt` and `go vet`. Exported names have doc comments. English in code, comments and messages.

## REV-009
**Docs follow the change.** A change to an MCP tool, an environment variable or the access flow updates `README.md`, `docs/` and `docs/pt-BR/` in the same PR. A change to an output updates the tutorials' `expected/`.
