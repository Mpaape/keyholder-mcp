# Contributing

Thanks for helping. English is the primary language of this repository: code, comments, error messages, commits and pull requests. The documentation also has a Portuguese version under `docs/pt-BR/`.

## Setup

```bash
go mod tidy          # the repository does not ship go.sum yet
go test ./...
go vet ./...
gofmt -l .
```

Requirements, layout and the documentation site: [Development](https://mpaape.github.io/keyholder-mcp/development/).

## Pull requests

1. Branch from `main` and keep the change focused.
2. Add or update tests for any new behavior.
3. If the change alters a tool, a variable, an error message or the access flow:
   - update the docs in `docs/` **and** `docs/pt-BR/`;
   - rerun the affected tutorials and commit `out/` and `expected/` together:
     `bash demo/tutorials/<name>/run.sh all && cp -r demo/tutorials/<name>/out/. demo/tutorials/<name>/expected/`.
4. Add a line to `CHANGELOG.md` under `## Unreleased`.
5. Open the pull request. CI runs on Linux, Windows and macOS, and [AurumCode](https://github.com/Mpaape/AurumCode) reviews it against the rules in `.aurumcode/skills/`. The `aurumcode/policy-gate` check must pass.

## Rules that block a merge

The full list is on [Code review](https://mpaape.github.io/keyholder-mcp/review/). The most important ones:

- A secret value never goes back to the agent, to a log, an error or a file.
- Commands run without a shell, only from the allow list, only with references in scope.
- Examples and tests use fake values and `example.com`, `localhost` or `127.0.0.1` only. Nothing internal goes into this public repository.

## Security

Never report a vulnerability in a public issue. See [SECURITY.md](SECURITY.md).

---

## Português

Obrigado por ajudar. O inglês é a língua principal do repositório (código, comentários, mensagens de erro, commits e pull requests); a documentação tem uma versão em português em `docs/pt-BR/`.

1. Crie um branch a partir do `main` com uma mudança focada.
2. Acrescente ou atualize testes para todo comportamento novo.
3. Se a mudança altera uma tool, uma variável, uma mensagem de erro ou o fluxo de acesso, atualize `docs/` **e** `docs/pt-BR/`, e rode de novo os tutoriais afetados, fazendo commit de `out/` e `expected/` juntos.
4. Acrescente uma linha ao `CHANGELOG.md` em `## Unreleased`.
5. Abra o pull request. O CI roda em Linux, Windows e macOS e o AurumCode o revisa; o check `aurumcode/policy-gate` precisa passar.

Detalhes: [Desenvolvimento](https://mpaape.github.io/keyholder-mcp/pt-BR/development/) e [Revisão de código](https://mpaape.github.io/keyholder-mcp/pt-BR/review/). Vulnerabilidades: veja o [SECURITY.md](SECURITY.md#português), nunca uma issue pública.
