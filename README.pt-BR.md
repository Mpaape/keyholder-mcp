# Keyholder MCP

[![CI](https://github.com/Mpaape/keyholder-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/Mpaape/keyholder-mcp/actions/workflows/ci.yml)
[![Docs](https://github.com/Mpaape/keyholder-mcp/actions/workflows/docs.yml/badge.svg)](https://mpaape.github.io/keyholder-mcp/pt-BR/)
[![Licença: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Deixe agentes de IA usarem suas credenciais sem nunca vê-las.**

[English](README.md) · [Documentação](https://mpaape.github.io/keyholder-mcp/pt-BR/) · [Tutoriais](https://mpaape.github.io/keyholder-mcp/pt-BR/tutorials/)

O Keyholder MCP é um servidor [MCP](https://modelcontextprotocol.io) sobre o [OpenBao](https://openbao.org). O agente vê os **nomes** das credenciais e pede para rodar comandos com elas; o valor só entra quando o comando executa (late binding), e a saída volta com o valor mascarado. O acesso é temporário: um token embrulhado, de uso único, com TTL.

```text
o agente pede: curl -H "Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}" https://api.example.com/health
o agente lê:   api says: hello, caller with header Authorization: Bearer [hidden:vault://my-project/dev/API_TOKEN]
```

> Situação: v0.1.0, a primeira release. Testado de ponta a ponta contra o OpenBao 2.7.1; cada tutorial é uma execução real e reconferida.

## Tools

| Tool | Faz | Devolve |
| --- | --- | --- |
| `keyholder_status` | Estado do cofre e da sessão | Selado ou não, TTL restante, políticas; nunca o token |
| `keyholder_list` | Referências no escopo do projeto | Só nomes `vault://` |
| `keyholder_run` | Roda um comando permitido com `{{vault://...}}` em `args` ou `env` | Código de saída, stdout e stderr mascarados, referências usadas |

## Início rápido

Baixe o `keyholder-mcp` (e o cliente de depuração `khcall`) para Linux, macOS ou Windows em [Releases](https://github.com/Mpaape/keyholder-mcp/releases) e confira com o `SHA256SUMS`, ou compile com Go 1.25+. Também é preciso o [OpenBao](https://openbao.org/docs/install/).

```bash
go build -o keyholder-mcp ./cmd/keyholder-mcp     # ou use o binário de uma release

bao server -dev -dev-root-token-id=root-dev        # em memória, nunca com segredos reais

export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put -mount=secret my-project/dev API_TOKEN=seu-token-de-api
bao policy write my-project-read deploy/policy-project-read.hcl
bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token
```

Depois cadastre o `keyholder-mcp` no seu cliente MCP com `BAO_ADDR`, `BAO_WRAPPING_TOKEN`, `KEYHOLDER_SCOPE=my-project` e `KEYHOLDER_ALLOW=curl`. Passo a passo: [Primeiros passos](https://mpaape.github.io/keyholder-mcp/pt-BR/getting-started/). Configuração no Claude Code, Cursor e Codex: [Clientes MCP](https://mpaape.github.io/keyholder-mcp/pt-BR/clients/).

## Garantias e limites

- Os valores nunca passam pelas mensagens MCP; o comando executado os recebe, como em qualquer uso real de credencial.
- Os comandos rodam sem shell, só se estiverem em `KEYHOLDER_ALLOW`, e só com referências dentro de `KEYHOLDER_SCOPE`.
- `BAO_*`, `VAULT_*` e `KEYHOLDER_*` nunca chegam ao processo filho.
- O mascaramento casa só o valor exato. Nunca libere shells ou interpretadores (`sh`, `bash`, `python`, `env`).
- Não protege contra um agente hostil rodando com o seu próprio usuário do sistema.

Detalhes: [Modelo de segurança](https://mpaape.github.io/keyholder-mcp/pt-BR/security/).

## Documentação

| Página | Conteúdo |
| --- | --- |
| [Primeiros passos](https://mpaape.github.io/keyholder-mcp/pt-BR/getting-started/) | Do zero a um agente usando uma credencial |
| [Tutoriais](https://mpaape.github.io/keyholder-mcp/pt-BR/tutorials/) | Início rápido, acesso temporário, proteções, mascaramento; saídas reais |
| [Configuração](https://mpaape.github.io/keyholder-mcp/pt-BR/configuration/) | Cada variável, tool e campo |
| [Modelo de segurança](https://mpaape.github.io/keyholder-mcp/pt-BR/security/) | O que é protegido e o que não é |
| [Arquitetura](https://mpaape.github.io/keyholder-mcp/pt-BR/architecture/) | Componentes, fluxo de acesso, pacotes |
| [Especificação](https://mpaape.github.io/keyholder-mcp/pt-BR/spec/) | Requisitos, decisões, critérios de aceite, roteiro |
| [Revisão de código](https://mpaape.github.io/keyholder-mcp/pt-BR/review/) | Regras do AurumCode de segurança, review e SOLID |

O inglês é a língua principal do repositório; esta é a versão secundária em português. As fontes do site estão em [`docs/`](docs/) (inglês) e [`docs/pt-BR/`](docs/pt-BR/) (português).

## Contribuir

Veja o [CONTRIBUTING.md](CONTRIBUTING.md#português). Todo pull request é revisado pelo [AurumCode](https://github.com/Mpaape/AurumCode) com as regras em [`.aurumcode/skills/`](.aurumcode/skills/). Falhas de segurança: [SECURITY.md](SECURITY.md#português).

## Licença

[MIT](LICENSE).
