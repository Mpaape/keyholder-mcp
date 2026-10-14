# Tutorials

Every tutorial is **executable**. Its commands are `demo/tutorials/<tutorial>/run.sh`, each case writes the real output of the last run to `out/`, and `run.sh --check` compares `out/` with `expected/`. The outputs on these pages are included straight from `expected/` when the site is built, so they cannot drift from what the code does. CI runs every tutorial and the check on each change.

Values that change between runs are shown by their shape: `<timestamp>`, `<duration>`, `<ttl>`, `<time>`, and `<wrapped token>` for the token that is never printed.

| Tutorial | Cases |
| --- | --- |
| [Quickstart](quickstart.md) | 1. the human stores a secret and a policy; 2. the agent sees the tools, the session and the names; 3. the agent calls an API with the token, masked |
| [Temporary access](access.md) | 1. a wrapped token opens one session, reusing it fails; 2. a 5-second token expires mid-session; 3. without a token the server explains and refuses |
| [Guardrails](guardrails.md) | 1. only the project scope, even with a broad policy; 2. commands outside the list, with a path or a placeholder; 3. reserved variables; 4. the vault token never reaches the child |
| [Masking and its limit](masking.md) | 1. a reflected value is masked; 2. LIMIT: an allowed interpreter encodes the value and escapes the mask |
| [Cloud credentials](cloud.md) | 1. the human enables the AWS engine; 2. the agent sees the dynamic path; 3. the AWS CLI runs with a credential made for the run and revoked after it; 4. unlisted paths are refused; 5. gcloud gets its token from a temporary file |

## Running them

You need `bash`, `go`, `bao` (OpenBao CLI), `curl` and `python3`; the cloud tutorial also needs `aws`, `gcloud`, [moto](https://github.com/getmoto/moto) and the OpenBao AWS plugin (see its page). Nothing touches a real vault: each run starts its own OpenBao dev server, in memory, on `127.0.0.1:18200`, with fake secrets, and stops it on exit.

```bash
bash demo/tutorials/quickstart/run.sh all       # every case; writes out/
bash demo/tutorials/quickstart/run.sh --check   # compares out/ with expected/
bash demo/tutorials/quickstart/run.sh 03-run    # a single case
```

`KH_BIN_DIR=<dir>` uses prebuilt `keyholder-mcp` and `khcall` instead of building them.
