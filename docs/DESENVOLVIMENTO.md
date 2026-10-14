# Desenvolvimento

## Requisitos

| Ferramenta | Versão | Uso |
| --- | --- | --- |
| Go | 1.25 ou superior | Compilar e testar |
| OpenBao (`bao`) | 2.7.x testado | Cofre; o modo dev basta para desenvolver |
| `curl` | qualquer | Exemplos e teste manual |

## Primeira compilação

O repositório ainda não tem `go.sum`. Gere-o na primeira vez; o Go confere cada módulo no checksum database oficial:

```sh
go mod tidy
go test ./...
go build -o keyholder-mcp ./cmd/keyholder-mcp      # Windows: -o keyholder-mcp.exe
./keyholder-mcp -version
```

Depois, faça commit do `go.sum` e ligue o scanner `govet` em `.aurumcode/config.yml` (ver [REVISAO.md](REVISAO.md)).

## Estrutura

| Caminho | Responsabilidade | Depende de |
| --- | --- | --- |
| `cmd/keyholder-mcp` | Lê a configuração, abre a sessão e registra as tools MCP | todos abaixo, MCP Go SDK |
| `internal/runner` | Lista de comandos permitidos, expansão dos placeholders, execução sem shell, filtro de ambiente, limites | `ref`, `mask` |
| `internal/bao` | Cliente OpenBao: desembrulho do token, leitura no KV v2, listagem, status | `ref`, cliente OpenBao |
| `internal/ref` | Interpretação de `vault://caminho/CHAVE` e de `{{vault://...}}` | biblioteca padrão |
| `internal/mask` | Troca valores conhecidos por `[oculto:<ref>]` | biblioteca padrão |
| `deploy/` | Política de exemplo para o OpenBao | — |

`internal/runner`, `internal/ref` e `internal/mask` não importam o OpenBao nem o MCP. O runner recebe um `runner.Resolver`; só `cmd/keyholder-mcp` liga o `bao.Store` a ele. Essa direção é regra bloqueante da revisão (`solid#solid-d-dependencia-de-abstracoes`).

## Testes

```sh
go test ./...
go vet ./...
gofmt -l .
```

Os testes do runner usam o próprio binário de teste como processo filho (`TestMain` com a variável `KH_TEST_HELPER`), então rodam igual em Windows, Linux e macOS, sem shell.

## Teste de ponta a ponta com OpenBao dev

```sh
bao server -dev -dev-root-token-id=root-dev        # só para desenvolver: dados em memória, sem TLS
```

Em outro terminal:

```sh
export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put secret/meu-projeto/dev API_TOKEN=valor-de-teste
bao policy write meu-projeto-leitura deploy/policy-projeto-leitura.hcl
WRAP=$(bao token create -policy=meu-projeto-leitura -ttl=10m -wrap-ttl=5m -field=wrapping_token)

unset BAO_TOKEN
BAO_WRAPPING_TOKEN=$WRAP KEYHOLDER_SCOPE=meu-projeto KEYHOLDER_ALLOW=curl ./keyholder-mcp
```

O servidor fala MCP por stdin/stdout. Para exercitar as tools, ligue-o a um cliente MCP (configuração no [README](../README.md#configuração-do-cliente-mcp)).

O que já foi verificado assim está em [SPEC.md, seção 8.2](SPEC.md#82-prova-de-conceito-verificado-em-2026-10-13).

## Fluxo de contribuição

1. Crie uma branch a partir da `main`.
2. Abra um PR. O AurumCode revisa e publica o parecer; o check `aurumcode/policy-gate` precisa passar.
3. Mudança em tool MCP, variável de ambiente ou fluxo de acesso atualiza `README.md` e `docs/SPEC.md` no mesmo PR (`review#rev-009`).
4. Registre a mudança em `CHANGELOG.md`, seção `## Unreleased`.
