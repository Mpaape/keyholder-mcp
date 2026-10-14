# Development

## Requirements

| Tool | Version | Use |
| --- | --- | --- |
| Go | 1.25 or later | Build and test |
| OpenBao (`bao`) | 2.7.x tested | Tutorials and manual tests |
| `curl`, `python3` | any | Tutorials |
| Docker | any | Building the documentation site |

## First build

The repository does not ship `go.sum` yet. Create it the first time; Go checks every module against the checksum database:

```bash
go mod tidy
go test ./...
go vet ./...
gofmt -l .
```

## Layout

| Path | What |
| --- | --- |
| `cmd/keyholder-mcp` | The MCP server |
| `cmd/khcall` | The debugging MCP client |
| `internal/` | `runner`, `bao`, `ref`, `mask` ([Architecture](architecture.md#packages)) |
| `deploy/` | Example OpenBao policy |
| `demo/tutorials/` | Executable tutorials: `run.sh`, `out/`, `expected/` |
| `docs/` | This site (English), `docs/pt-BR/` (Portuguese) |
| `.aurumcode/` | Code review rules ([Code review](review.md)) |
| `.github/workflows/` | CI, code review, documentation site |

## Tests

```bash
go test ./...
```

The runner tests use a copy of the test binary itself as the child process (`TestMain` with `KH_TEST_HELPER`), so they run the same on Windows, Linux and macOS, without a shell.

## Tutorials

```bash
for t in quickstart access guardrails masking; do
  bash demo/tutorials/$t/run.sh all && bash demo/tutorials/$t/run.sh --check
done
```

When a change alters an output on purpose, run the tutorial, review `out/`, copy it to `expected/` and commit both. CI fails when `out/` and `expected/` disagree.

## Documentation site

The site is [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) in a container pinned by digest (`scripts/docs/image.lock`):

```bash
scripts/docs/build.sh     # builds ./site with --strict
scripts/docs/serve.sh     # http://127.0.0.1:8000 with live reload
```

Tutorial pages include the files in `demo/tutorials/*/expected/` at build time, so the outputs on the site are always the checked ones. Every English page has a Portuguese counterpart under `docs/pt-BR/`; change both in the same pull request.

## CI

| Workflow | Runs |
| --- | --- |
| `ci.yml` | `gofmt`, `go vet`, `go test`, `go build` on Linux, Windows and macOS; every tutorial and its `--check` on Linux |
| `aurumcode.yml` | AI code review with the project rules, gitleaks and Semgrep |
| `docs.yml` | Builds the site with `--strict` on every PR; publishes it to GitHub Pages from `main` |

## AurumCode gate in your agent

`.mcp.json` registers [`aurumcode mcp`](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/agentes.md) for Claude Code, so the agent can ask the same gate the CI uses (`aurum_gate`, `aurum_review`, `aurum_rules`, `aurum_explain`) before it commits. It needs the `aurumcode` binary on the `PATH` and your key in the shell that starts Claude Code; the key is never written to the file:

```bash
read -s LLM_API_KEY && export LLM_API_KEY     # provider defaults: Anthropic, claude-haiku-5-5
claude                                         # approve the project server "aurum" once
```

Without a working provider the gate answers `inconclusive`, never `pass`.

## Contributing

1. Branch from `main`.
2. Open a pull request. AurumCode reviews it; the `aurumcode/policy-gate` check must pass.
3. A change to a tool, a variable or the access flow updates `README.md`, the docs and their Portuguese versions in the same PR.
4. Add a line to `CHANGELOG.md` under `## Unreleased`.

English is the primary language of the repository: code, comments, messages and commits. Portuguese is the second language of the documentation.
