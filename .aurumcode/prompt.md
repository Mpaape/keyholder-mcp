Keyholder MCP lets AI agents use credentials stored in OpenBao without seeing their values.
The agent only knows `vault://path/KEY` references; the value is resolved in the runner,
when an allowed command runs, and the output comes back masked.

When reviewing, the main question is: can this change let a secret value, the vault token or
the wrapped token reach the agent, a log, an error, a file or a process that should not get it?
Follow the data from `bao.Store.Resolve` to the MCP response before concluding.

Sensitive parts of the code:
- `internal/runner`: allow list, execution without a shell, environment filter
  (`BAO_*`, `VAULT_*`, `KEYHOLDER_*`), output cap and masking.
- `internal/bao`: token unwrap, removal of the token variables from the environment,
  reference scope, and error messages without URLs or response bodies.
- `cmd/keyholder-mcp`: what each tool returns and what goes to the audit log.
- `internal/ref`: reference validation (no `..`, no empty segment).

Limits the project accepts: the executed process receives the value; masking only matches the
exact value; there is no protection against a hostile agent running as the same OS user. Do not
report these limits as defects unless the change makes them worse.

The repository is in English; the documentation also has Portuguese pages under `docs/pt-BR/`.
Use the existing tests and the tutorials' `expected/` outputs to support each finding, and say
when a test is missing.
