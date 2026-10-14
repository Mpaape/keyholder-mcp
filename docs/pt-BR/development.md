# Desenvolvimento

## Requisitos

| Ferramenta | Versão | Uso |
| --- | --- | --- |
| Go | 1.25 ou superior | Compilar e testar |
| OpenBao (`bao`) | 2.7.x testado | Tutoriais e testes manuais |
| `curl`, `python3` | qualquer | Tutoriais |
| Docker | qualquer | Gerar o site de documentação |

## Primeira compilação

O `go.sum` está no repositório; o `go mod verify` confere os módulos baixados contra ele:

```bash
go mod verify
go test ./...
go vet ./...
gofmt -l .
```

## Estrutura

| Caminho | O quê |
| --- | --- |
| `cmd/keyholder-mcp` | O servidor MCP |
| `cmd/khcall` | O cliente MCP de depuração |
| `internal/` | `runner`, `bao`, `ref`, `mask` ([Arquitetura](architecture.md#pacotes)) |
| `deploy/` | Política de exemplo do OpenBao |
| `demo/tutorials/` | Tutoriais executáveis: `run.sh`, `out/`, `expected/` |
| `docs/` | Este site (inglês), `docs/pt-BR/` (português) |
| `.aurumcode/` | Regras de revisão de código ([Revisão de código](review.md)) |
| `.github/workflows/` | CI, revisão de código, site de documentação |

## Testes

```bash
go test ./...
```

Os testes do runner usam uma cópia do próprio binário de teste como processo filho (`TestMain` com `KH_TEST_HELPER`), então rodam igual no Windows, Linux e macOS, sem shell.

## Tutoriais

```bash
for t in quickstart access guardrails masking; do
  bash demo/tutorials/$t/run.sh all && bash demo/tutorials/$t/run.sh --check
done
```

Quando uma mudança altera uma saída de propósito, rode o tutorial, revise `out/`, copie para `expected/` e faça commit dos dois. O CI falha quando `out/` e `expected/` divergem.

## Site de documentação

O site é [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) num contêiner fixado por digest (`scripts/docs/image.lock`):

```bash
scripts/docs/build.sh     # gera ./site com --strict
scripts/docs/serve.sh     # http://127.0.0.1:8000 com recarga automática
```

As páginas de tutorial incluem os arquivos de `demo/tutorials/*/expected/` na geração, então as saídas no site são sempre as conferidas. Cada página em inglês tem uma equivalente em `docs/pt-BR/`; altere as duas no mesmo pull request.

## CI

| Workflow | Roda |
| --- | --- |
| `ci.yml` | `gofmt`, `go vet`, `go test`, `go build` em Linux, Windows e macOS; cada tutorial e seu `--check` no Linux |
| `aurumcode.yml` | Revisão de código por IA com as regras do projeto, gitleaks e Semgrep |
| `release.yml` | Quando a `main` recebe uma versão do CHANGELOG ainda sem release (ou numa tag `v*`): testa, compila os binários para seis plataformas e publica com `SHA256SUMS` e a seção do CHANGELOG |
| `docs.yml` | Gera o site com `--strict` em todo PR; publica no GitHub Pages a partir do `main` |
| `attribution.yml` | Remove atribuição de IA de PRs, issues e comentários; reprova um PR cujos commits a tragam |

## Gate do AurumCode no seu agente

O `.mcp.json` registra o [`aurumcode mcp`](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/agentes.md) para o Claude Code, então o agente consulta o mesmo gate do CI (`aurum_gate`, `aurum_review`, `aurum_rules`, `aurum_explain`) antes de commitar. Precisa do binário `aurumcode` no `PATH` e da sua chave no shell que abre o Claude Code; a chave nunca vai para o arquivo:

```bash
read -s LLM_API_KEY && export LLM_API_KEY     # padrões do provedor: Anthropic, claude-haiku-5-5
claude                                         # aprove o servidor de projeto "aurum" uma vez
```

Sem provedor funcionando o gate responde `inconclusive`, nunca `pass`.

## Publicar uma versão

Mova as entradas de `## Unreleased` para um novo título `## X.Y.Z - data` no `CHANGELOG.md` e faça o merge. O `release.yml` vê uma versão sem release, cria a tag `vX.Y.Z` nesse commit e publica os binários. Enviar você mesmo uma tag `vX.Y.Z` também funciona.

## Sem atribuição de IA

Commits, pull requests, issues e comentários não levam atribuição de IA. Ative o hook local uma vez por clone:

```bash
git config core.hooksPath .githooks
```

O Claude Code carrega o `.claude/settings.json`, que desliga a atribuição dele e bloqueia comandos que a publicariam. No GitHub, o `attribution.yml` a remove dos textos e reprova um PR cujos commits a tragam. Os três leem `scripts/attribution/patterns.txt`.

## Contribuir

1. Crie um branch a partir do `main`.
2. Abra um pull request. O AurumCode o revisa; o check `aurumcode/policy-gate` precisa passar.
3. Uma mudança numa tool, numa variável ou no fluxo de acesso atualiza o `README.md`, a documentação e as versões em português no mesmo PR.
4. Acrescente uma linha ao `CHANGELOG.md` em `## Unreleased`.

O inglês é a língua principal do repositório: código, comentários, mensagens e commits. O português é a segunda língua da documentação.
