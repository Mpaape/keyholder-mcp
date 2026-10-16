# Development

## Requirements

| Tool | Version | Use |
| --- | --- | --- |
| Go | 1.25 or later | Build and test |
| OpenBao (`bao`) | 2.7.x tested | Tutorials and manual tests |
| `curl`, `python3` | any | Tutorials |
| Docker | any | Building the documentation site |

## First build

`go.sum` is committed; `go mod verify` checks the downloaded modules against it:

```bash
go mod verify
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
| `ci.yml` | The pipeline: `gofmt`, `go vet`, `go test`, `go build` on Linux, Windows and macOS; every tutorial and its `--check` on Linux; then, on a pull request, the AurumCode gate (project rules, gitleaks, Semgrep), which never approves |
| `release.yml` | When `main` gets a CHANGELOG version with no release yet (or on a `v*` tag): tests, builds the binaries for six platforms, publishes them with `SHA256SUMS` and the CHANGELOG section |
| `docs.yml` | Builds the site with `--strict` on every PR; publishes it to GitHub Pages from `main` |
| `attribution.yml` | The maintainer's own: strips AI attribution from the maintainer's pull requests, issues and comments. Contributors are not affected |

## AurumCode gate in your agent

`.mcp.json` registers [`aurumcode mcp`](https://github.com/Mpaape/AurumCode/blob/v2.1.0/docs/agentes.md) for Claude Code, so the agent can ask the same gate the CI uses (`aurum_gate`, `aurum_review`, `aurum_rules`, `aurum_explain`) before it commits. It needs the `aurumcode` binary on the `PATH` and your key in the shell that starts Claude Code; the key is never written to the file:

```bash
read -s LLM_API_KEY && export LLM_API_KEY     # provider defaults: Anthropic, claude-haiku-5-5
claude                                         # approve the project server "aurum" once
```

Without a working provider the gate answers `inconclusive`, never `pass`.

## Branches and releases

Two branches only: `dev`, where every change lands, and `main`, what is released. `main` changes only through a pull request from `dev`, merged with a merge commit after its checks pass; then `dev` is fast-forwarded to `main`. Contributors open pull requests into `dev` from a fork. Details and the exact steps: [CONTRIBUTING.md](https://github.com/Mpaape/keyholder-mcp/blob/main/CONTRIBUTING.md#branches).

To release, move the `## Unreleased` entries under `## X.Y.Z - date` in `CHANGELOG.md` on `dev` and ship it to `main`. `release.yml` sees a version without a release, creates the tag `vX.Y.Z` on that commit and publishes the binaries.

## Contributing

See [CONTRIBUTING.md](https://github.com/Mpaape/keyholder-mcp/blob/main/CONTRIBUTING.md): branch from `dev`, test, update both languages of the docs and the CHANGELOG, open the pull request into `dev`.

English is the primary language of the repository: code, comments, messages and commits. Portuguese is the second language of the documentation.
