# Contributing

Thanks for helping. English is the primary language of this repository: code, comments, error messages, commits and pull requests. The documentation also has a Portuguese version under `docs/pt-BR/`.

## Setup

```bash
go mod verify
go test ./...
go vet ./...
gofmt -l .
```

Requirements, layout and the documentation site: [Development](https://mpaape.github.io/keyholder-mcp/development/).

## Branches

The repository has two branches and no others.

| Branch | Is | Who writes to it |
| --- | --- | --- |
| `dev` | Where work lands. Every change goes here first. | The maintainer pushes to it; everyone else opens a pull request into it from a fork |
| `main` | What is released and published (site, binaries) | Only through a pull request from `dev`, merged with a merge commit |

Both are protected against deletion and force pushes, and `main` only changes through a pull request whose checks passed. The repository admin can bypass both rules. The rules are versioned in [`.github/rulesets/`](.github/rulesets/).

## Working on a change

1. Fork the repository and branch from `dev` in your fork. (The maintainer works on `dev` directly.)
2. Add or update tests for any new behavior.
3. If the change alters a tool, a variable, an error message or the access flow:
   - update the docs in `docs/` **and** `docs/pt-BR/`;
   - rerun the affected tutorials and commit `out/` and `expected/` together:
     `bash demo/tutorials/<name>/run.sh all && cp -r demo/tutorials/<name>/out/. demo/tutorials/<name>/expected/`.
4. Add a line to `CHANGELOG.md` under `## Unreleased`.
5. Open the pull request **into `dev`**. CI runs on Linux, Windows and macOS, and [AurumCode](https://github.com/Mpaape/AurumCode) reviews it against the rules in `.aurumcode/skills/`. The `aurumcode/policy-gate` check must pass.

## Shipping to main

1. On `dev`, move the `## Unreleased` entries under `## X.Y.Z - date` in `CHANGELOG.md` when the change should be released.
2. Open a pull request from `dev` to `main`. The required checks: `go test` on the three systems, `tutorials` and `aurumcode/policy-gate`.
3. Merge it with **Create a merge commit**, the only method `main` allows, so `dev` can be fast-forwarded afterwards.
4. Fast-forward `dev` to `main`: `git fetch origin && git push origin origin/main:dev`.

When the merge reaches `main`, the site is published, and if `CHANGELOG.md` has a version without a release, the release workflow tags it and publishes the binaries.

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

**Branches.** O repositório tem só dois branches:

| Branch | É | Quem escreve nele |
| --- | --- | --- |
| `dev` | Onde o trabalho entra. Toda mudança passa por ele primeiro. | O mantenedor faz push direto; os demais abrem pull request para ele a partir de um fork |
| `main` | O que está publicado (site, binários) | Só por pull request de `dev`, com merge commit |

Os dois são protegidos contra exclusão e force push, e o `main` só muda por um pull request com os checks aprovados. O admin do repositório pode passar por cima das duas regras. As regras ficam versionadas em [`.github/rulesets/`](.github/rulesets/).

**Uma mudança:**

1. Faça um fork e crie seu branch a partir de `dev` no fork. (O mantenedor trabalha direto no `dev`.)
2. Acrescente ou atualize testes para todo comportamento novo.
3. Se a mudança altera uma tool, uma variável, uma mensagem de erro ou o fluxo de acesso, atualize `docs/` **e** `docs/pt-BR/`, e rode de novo os tutoriais afetados, fazendo commit de `out/` e `expected/` juntos.
4. Acrescente uma linha ao `CHANGELOG.md` em `## Unreleased`.
5. Abra o pull request **para o `dev`**. O CI roda em Linux, Windows e macOS e o AurumCode o revisa; o check `aurumcode/policy-gate` precisa passar.

**Levar para o `main`:**

1. No `dev`, mova as entradas de `## Unreleased` para `## X.Y.Z - data` no `CHANGELOG.md` quando a mudança deve virar release.
2. Abra um pull request de `dev` para `main`. Checks obrigatórios: `go test` nos três sistemas, `tutorials` e `aurumcode/policy-gate`.
3. Faça o merge com **Create a merge commit**, o único método que o `main` aceita, para o `dev` poder avançar depois.
4. Avance o `dev` até o `main`: `git fetch origin && git push origin origin/main:dev`.

Quando o merge chega ao `main`, o site é publicado e, se o `CHANGELOG.md` tiver uma versão sem release, o workflow de release cria a tag e publica os binários.

Detalhes: [Desenvolvimento](https://mpaape.github.io/keyholder-mcp/pt-BR/development/) e [Revisão de código](https://mpaape.github.io/keyholder-mcp/pt-BR/review/). Vulnerabilidades: veja o [SECURITY.md](SECURITY.md#português), nunca uma issue pública.
