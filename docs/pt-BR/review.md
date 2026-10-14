# Revisão de código

Todo pull request é revisado pelo [AurumCode](https://github.com/Mpaape/AurumCode) v2.0.0 com as regras do próprio projeto. O check `aurumcode/policy-gate` reprova o PR quando há um achado `error`.

## Como está ligado

| Arquivo | Papel |
| --- | --- |
| `.github/workflows/aurumcode.yml` | Chama o workflow reutilizável `Mpaape/AurumCode/.github/workflows/review.yml@v2.0.0` em cada PR |
| `.aurumcode/config.yml` | Idioma, publicação, gate, scanners, deliberação e exceções |
| `.aurumcode/prompt.md` | Contexto do projeto enviado ao modelo |
| `.aurumcode/skills/<nome>/SKILL.md` | Regras do projeto; cada seção `## ` vira uma regra citável |

A configuração e as skills são lidas do **branch base** do PR: uma regra nova só vale depois de entrar no `main`.

| Job | Faz |
| --- | --- |
| `review` | Revisão pelo modelo, passe determinístico de segurança, gitleaks e Semgrep |
| `upload-sarif` | Envia o SARIF para a aba Security (code scanning); pulado em PRs de fork |
| `provider` | Falha fechado: sem `LLM_API_KEY` e `LLM_BASE_URL`, marca `aurumcode/policy-gate` como reprovado |

## Secrets e variáveis

Os mesmos nomes do repositório do AurumCode. Settings → Secrets and variables → Actions.

| Tipo | Nome | Obrigatório | Valor com a API da Anthropic |
| --- | --- | --- | --- |
| Secret | `LLM_API_KEY` | Sim | A sua chave da API do Claude |
| Secret | `LLM_BASE_URL` | Sim | `https://api.anthropic.com/v1` |
| Variável | `LLM_PROVIDER` | Não | `anthropic` |
| Variável | `LLM_MODEL` | Depende do provedor | Id do modelo, por exemplo `claude-haiku-5-5` |
| Secret | `LLM_FALLBACK_1_API_KEY`, `LLM_FALLBACK_1_BASE_URL` | Não | Provedor reserva |
| Variável | `LLM_FALLBACK_1_PROVIDER`, `LLM_FALLBACK_1_MODEL` | Não | Provedor reserva |

Com a [GitHub CLI](https://cli.github.com/); `gh secret set` sem `--body` pede o valor sem ecoá-lo:

```bash
gh secret set LLM_API_KEY -R Mpaape/keyholder-mcp
gh secret set LLM_BASE_URL -R Mpaape/keyholder-mcp --body "https://api.anthropic.com/v1"
gh variable set LLM_PROVIDER -R Mpaape/keyholder-mcp --body "anthropic"
gh variable set LLM_MODEL -R Mpaape/keyholder-mcp --body "claude-haiku-5-5"
```

Perfis e URLs de provedores: [provedores do AurumCode](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/provedores.md). PRs do Dependabot só recebem *Dependabot secrets*; para revisá-los, cadastre `LLM_API_KEY` e `LLM_BASE_URL` lá também.

## Gate

| Configuração | Efeito |
| --- | --- |
| `gate.fail_on: [error]` | Um achado `error` do modelo, de uma skill ou de um scanner reprova o check |
| `gate.inconclusive: block` | Uma revisão que não terminou (provedor fora, scanner falhou) nunca passa |
| `gitleaks`, `required: true` | Segredos em qualquer commit do PR, não só na árvore final |
| `semgrep` | SAST; reprova só em linhas que o PR acrescentou |
| `govet` | Desligado até o `go.sum` entrar no repositório |

Para o check bloquear merges, depois do primeiro PR: Settings → Branches → regra do `main` → *Require status checks to pass before merging* → `aurumcode/policy-gate`.

## Regras do projeto

Id citável: `<skill>#<slug do título>`, por exemplo `security#sec-001-a-secret-value-never-goes-back-to-the-agent`. As regras estão em inglês, a língua principal do repositório. As marcadas "Sim" têm `severity: error` e bloqueiam o merge; as outras são avisos.

### `security` — caminhos ["**"]

| Regra | Bloqueia | Em resumo |
| --- | --- | --- |
| SEC-001 A secret value never goes back to the agent | Sim | Nenhum valor de segredo volta ao agente |
| SEC-002 A secret never goes to a log, an error or a file | Sim | Segredo nunca vai para log, erro ou arquivo |
| SEC-003 Execution without a shell | Sim | Execução sem shell |
| SEC-004 Command allow list | Sim | Lista de comandos permitidos |
| SEC-005 The vault token stays out of the child process | Sim | O token do cofre fica fora do processo filho |
| SEC-006 Project scope | Sim | Escopo do projeto |
| SEC-007 Reference validation | Sim | Validação das referências |
| SEC-008 Masking covers all output | Sim | O mascaramento cobre toda a saída |
| SEC-009 Token in memory only | Sim | Token só em memória |
| SEC-010 TLS and vault address | Sim | TLS e endereço do cofre |
| SEC-011 Bounded output and time | Não | Saída e tempo limitados |
| SEC-012 New dependencies are justified | Não | Dependência nova tem justificativa |
| SEC-013 Public repository: nothing internal | Sim | Repositório público: nada interno |
| SEC-014 Examples and tests use fake values only | Sim | Exemplos e testes só com valores falsos |

### `review` — caminhos ["cmd/**", "internal/**"]

| Regra | Bloqueia | Em resumo |
| --- | --- | --- |
| REV-001 Errors are handled and carry context | Sim | Erros são tratados e levam contexto |
| REV-002 New behavior comes with a test | Sim | Comportamento novo vem com teste |
| REV-003 Context is propagated | Não | O `context` é propagado |
| REV-004 stdout belongs to the MCP protocol | Sim | O stdout é do protocolo MCP |
| REV-005 No panic on a request path | Sim | Sem panic no caminho de uma requisição |
| REV-006 Windows, Linux and macOS | Não | Windows, Linux e macOS |
| REV-007 Messages to the agent are actionable | Não | Mensagens ao agente dizem o que fazer |
| REV-008 Formatted and vetted code | Não | Código formatado e passado no vet |
| REV-009 Docs follow the change | Não | A documentação acompanha a mudança |

### `solid` — caminhos ["cmd/**", "internal/**"]

| Regra | Bloqueia | Em resumo |
| --- | --- | --- |
| SOLID-S One responsibility per package | Não | Uma responsabilidade por pacote |
| SOLID-O Open for extension, closed for modification | Não | Aberto para extensão, fechado para modificação |
| SOLID-L Substitution without surprise | Não | Substituição sem surpresa |
| SOLID-I Small interfaces, defined by the consumer | Não | Interfaces pequenas, definidas por quem consome |
| SOLID-D Depend on abstractions | Sim | Dependa de abstrações |

Para mudar uma regra, edite a seção dela no `SKILL.md`. Para criar uma, acrescente uma seção `## ` com id e título; `severity: error` na primeira linha do corpo a torna bloqueante.

## Exceções aprovadas

Um falso positivo revisado entra em `exceptions` no `.aurumcode/config.yml`, com a regra e o caminho exatos, dono, motivo e validade. Ele deixa de valer sozinho na data.

| Regra | Arquivo | Motivo | Expira |
| --- | --- | --- | --- |
| `analysis/hardcoded-secret` | `internal/bao/bao.go` | `EnvWrappingToken` e `EnvToken` guardam nomes de variáveis de ambiente, não segredos | 2027-04-13 |

## Rodar localmente

Com a CLI do AurumCode (Go 1.27 ou a imagem Docker dele), na raiz do repositório:

```bash
aurumcode review --base origin/main --seguranca
```

Sem `LLM_API_KEY` e `LLM_BASE_URL` só a análise determinística roda; sem `gitleaks` e `semgrep` no `PATH` esses scanners aparecem como inconclusivos. Veja o [início rápido do AurumCode](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/getting-started.md).
