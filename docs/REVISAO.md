# Revisão de código com AurumCode

Todo pull request para este repositório é revisado pelo [AurumCode](https://github.com/Mpaape/AurumCode) v2.0.0. O check `aurumcode/policy-gate` reprova o PR quando há achado de severidade `error`.

## Como está ligado

| Arquivo | Papel |
| --- | --- |
| `.github/workflows/aurumcode.yml` | Chama o workflow reutilizável `Mpaape/AurumCode/.github/workflows/review.yml@v2.0.0` em cada PR |
| `.aurumcode/config.yml` | Idioma, publicação, gate, scanners, deliberação e exceções |
| `.aurumcode/prompt.md` | Contexto do projeto enviado ao modelo |
| `.aurumcode/skills/<nome>/SKILL.md` | Regras do projeto; cada seção `## ` vira uma regra citável |

A configuração e as skills são lidas da **branch base** do PR. Uma regra nova só passa a valer depois de integrada na `main`.

O workflow tem três jobs:

| Job | O que faz |
| --- | --- |
| `review` | Revisão pelo modelo, passada de segurança determinística, gitleaks e Semgrep |
| `upload-sarif` | Envia o SARIF para a aba Security (Code scanning); pulado em PR de fork |
| `provider` | Falha fechada: sem `LLM_API_KEY` e `LLM_BASE_URL`, publica `aurumcode/policy-gate` como failure |

## Secrets e variáveis

Mesmos nomes usados no repositório do AurumCode. Configure em Settings → Secrets and variables → Actions.

| Tipo | Nome | Obrigatório | Valor com a API da Anthropic |
| --- | --- | --- | --- |
| Secret | `LLM_API_KEY` | Sim | Chave da API do Claude |
| Secret | `LLM_BASE_URL` | Sim | `https://api.anthropic.com/v1` |
| Variável | `LLM_PROVIDER` | Não | `anthropic` |
| Variável | `LLM_MODEL` | Depende do provedor | Id do modelo, por exemplo `claude-haiku-5-5` |
| Secret | `LLM_FALLBACK_1_API_KEY` | Não | Chave do provedor reserva |
| Secret | `LLM_FALLBACK_1_BASE_URL` | Não | URL do provedor reserva |
| Variável | `LLM_FALLBACK_1_PROVIDER` | Não | Perfil do provedor reserva |
| Variável | `LLM_FALLBACK_1_MODEL` | Não | Modelo do provedor reserva |

Pelo terminal, com o [GitHub CLI](https://cli.github.com/) (o `gh secret set` sem `--body` pede o valor sem mostrá-lo):

```sh
gh secret set LLM_API_KEY -R Mpaape/keyholder-mcp
gh secret set LLM_BASE_URL -R Mpaape/keyholder-mcp --body "https://api.anthropic.com/v1"
gh variable set LLM_PROVIDER -R Mpaape/keyholder-mcp --body "anthropic"
gh variable set LLM_MODEL -R Mpaape/keyholder-mcp --body "claude-haiku-5-5"
```

Perfis aceitos e URLs de cada provedor: [docs/provedores.md do AurumCode](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/provedores.md).

PRs do Dependabot só recebem os *Dependabot secrets*. Para revisá-los, cadastre `LLM_API_KEY` e `LLM_BASE_URL` também em Settings → Secrets and variables → Dependabot.

## Gate

| Configuração | Efeito |
| --- | --- |
| `gate.fail_on: [error]` | Achado `error` do modelo, de uma skill ou de um scanner reprova o check |
| `gate.inconclusive: block` | Revisão que não concluiu (provedor fora, scanner que falhou) nunca aprova |
| `gitleaks` com `required: true` | Segredo em qualquer commit do PR, não só no estado final |
| `semgrep` | SAST; só reprova pelo que o PR adicionou |
| `govet` | Desligado até o `go.sum` estar no repositório |

Para o check bloquear o merge, depois do primeiro PR: Settings → Branches → regra da `main` → *Require status checks to pass before merging* → `aurumcode/policy-gate`.

## Regras do projeto

Id citável: `<skill>#<slug do título>`, por exemplo `seguranca#seg-001-valor-de-segredo-nunca-volta-ao-agente`. Regras marcadas "bloqueia" têm `severity: error`; as demais são `warning`.

### `seguranca` (todos os arquivos)

| Regra | Bloqueia |
| --- | --- |
| SEG-001 Valor de segredo nunca volta ao agente | Sim |
| SEG-002 Segredo nunca vai para log, erro ou arquivo | Sim |
| SEG-003 Execução sem shell | Sim |
| SEG-004 Lista de comandos permitidos | Sim |
| SEG-005 Token do cofre fora do processo filho | Sim |
| SEG-006 Escopo do projeto | Sim |
| SEG-007 Validação de referências | Sim |
| SEG-008 Mascaramento cobre toda saída | Sim |
| SEG-009 Token só em memória | Sim |
| SEG-010 TLS e endereço do cofre | Sim |
| SEG-011 Saída e tempo limitados | Não |
| SEG-012 Dependências novas justificadas | Não |
| SEG-013 Repositório público: nada interno | Sim |
| SEG-014 Exemplos e testes só com valores falsos | Sim |

### `review` (`cmd/**`, `internal/**`)

| Regra | Bloqueia |
| --- | --- |
| REV-001 Erros tratados e com contexto | Sim |
| REV-002 Comportamento novo vem com teste | Sim |
| REV-003 Contexto propagado | Não |
| REV-004 stdout é do protocolo MCP | Sim |
| REV-005 Sem panic em caminho de requisição | Sim |
| REV-006 Portabilidade Windows, Linux e macOS | Não |
| REV-007 Mensagens ao agente são acionáveis | Não |
| REV-008 Código formatado e verificado | Não |
| REV-009 Spec e README acompanham a mudança | Não |

### `solid` (`cmd/**`, `internal/**`)

| Regra | Bloqueia |
| --- | --- |
| SOLID-S Responsabilidade única por pacote | Não |
| SOLID-O Aberto para extensão, fechado para modificação | Não |
| SOLID-L Substituição sem surpresa | Não |
| SOLID-I Interfaces pequenas, definidas por quem consome | Não |
| SOLID-D Dependência de abstrações | Sim |

Para mudar uma regra, edite a seção no `SKILL.md`. Para criar uma, acrescente uma seção `## ` com id e título; `severity: error` na primeira linha do corpo a torna bloqueante.

## Exceções aprovadas

Falso positivo ou risco aceito entra em `exceptions` no `.aurumcode/config.yml`, com regra e caminho exatos, dono, motivo e validade. A exceção vence sozinha na data e o achado volta a reprovar.

| Regra | Arquivo | Motivo | Validade |
| --- | --- | --- | --- |
| `analysis/hardcoded-secret` | `internal/bao/bao.go` | `EnvWrappingToken` e `EnvToken` guardam nomes de variáveis de ambiente, não segredos | 2027-04-13 |

## Rodar localmente

Com o CLI do AurumCode (Go 1.27 ou a imagem Docker dele), na raiz deste repositório:

```sh
aurumcode review --base origin/main --seguranca
```

Sem `LLM_API_KEY` e `LLM_BASE_URL` roda só a análise determinística. Sem `gitleaks` e `semgrep` no `PATH`, os dois scanners aparecem como inconclusivos. Uso local: [getting-started do AurumCode](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/getting-started.md).
