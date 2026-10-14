---
name: security
version: 1
paths: ["**"]
---
Security rules of Keyholder MCP. Each section is a rule the gate can cite.

## SEC-001
severity: error
**A secret value never goes back to the agent.** No MCP tool result, structured content, error message or tool description may contain a resolved reference value, the vault token or the wrapped token. Only `vault://` names, status and masked output.

## SEC-002
severity: error
**A secret never goes to a log, an error or a file.** Logs (including the audit line), errors formatted with `%v`/`%w`, panics and temporary files must not include resolved values, expanded arguments, the child's environment or the token. The audit line records command, references, exit code and duration.

## SEC-003
severity: error
**Execution without a shell.** Commands run with `exec.CommandContext(ctx, path, args...)` and arguments as a list. Building a command by concatenation, calling `sh -c`, `bash -c`, `cmd /c`, `powershell -Command`, or interpreting an argument's text is forbidden.

## SEC-004
severity: error
**Command allow list.** Every command must pass `Runner.Allow` before `exec.LookPath`. The name may not contain a path separator or a placeholder. Changes that loosen this check, or that allow interpreters by default (`sh`, `bash`, `python`, `node`, `env`), block the merge.

## SEC-005
severity: error
**The vault token stays out of the child process.** The child's environment goes through `filterEnv`, which removes `BAO_*`, `VAULT_*` and `KEYHOLDER_*`, and a request may not set variables with those prefixes. The token variables are removed from the process right after `bao.Open`.

## SEC-006
severity: error
**Project scope.** Every resolved reference must pass `ref.InScope` with the configured scope. A direct KV read that bypasses the runner or the scope blocks the merge.

## SEC-007
severity: error
**Reference validation.** `ref.Parse` rejects empty segments, `.`, `..` and characters outside `[A-Za-z0-9_.-]`. A change to this validation needs a test that covers path traversal.

## SEC-008
severity: error
**Masking covers all output.** stdout, stderr and execution error messages go through `mask.Masker` with every value resolved in that run before they reach the agent.

## SEC-009
severity: error
**Token in memory only.** The unwrapped token is never written to disk, a cache, a config file or an environment variable. Prefer `BAO_WRAPPING_TOKEN`, single use, over `BAO_TOKEN`.

## SEC-010
severity: error
**TLS and vault address.** Do not turn off TLS verification in code (`InsecureSkipVerify`, forcing `BAO_SKIP_VERIFY`). TLS settings come from the environment, as with the `bao` CLI.

## SEC-011
**Bounded output and time.** Every run has a timeout and an output cap (`Runner.Timeout`, `Runner.MaxOutput`). Removing or raising them without bound lets a command hang the MCP server.

## SEC-012
**New dependencies are justified.** A new dependency in `go.mod` needs a reason in the PR. Prefer the standard library and the official SDKs (MCP Go SDK, OpenBao client).

## SEC-013
severity: error
**Public repository: nothing internal.** The repository is public. An added line does not name a company, an internal domain or host, a private repository or a real vault address. Examples use `example.com`, `localhost` or `127.0.0.1`.

## SEC-014
severity: error
**Examples and tests use fake values only.** Documentation, example policies, tutorials and tests use obviously fake values (`tok-fake-not-a-real-secret`, `root-dev`). Never a real token, key or credential URL, not even a fragment.
