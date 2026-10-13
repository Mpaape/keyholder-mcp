# Keyholder MCP

Cofre local de credenciais para agentes de IA. O agente descobre **nomes** de credenciais e as usa na execução, mas o **valor nunca entra no chat**. O acesso é temporário: expira sozinho.

> Status: especificação inicial. Ainda não há código. Veja [`docs/SPEC.md`](docs/SPEC.md).

## Ideia em uma frase

Um `.env` global e criptografado, organizado por workspace, projeto e ambiente, usado por referências (`vault://workspace/projeto/nome`) e resolvido só no momento da execução.

## Decisão base (a confirmar)

Em vez de construir um cofre do zero, o projeto se apoia no [OpenBao](https://openbao.org) (open source, MPL-2.0) e constrói por cima dele a única peça nova: um **MCP + runner** que injeta credenciais sem devolver valores ao modelo.

| Peça | Origem |
| --- | --- |
| Cofre, Web UI, auditoria, tokens com TTL, leases, MFA (TOTP) | OpenBao (existente) |
| Credenciais dinâmicas de AWS e GCP | Plugins externos oficiais do OpenBao |
| MCP, runner com `vault://` e `{{...}}`, mascaramento, sessões | **Este projeto (novo)** |

## Licença

Ainda não definida. Decidir antes do primeiro código (ver decisão aberta em `docs/SPEC.md`).

## Contribuição

Ponto natural de contribuição upstream: um equivalente ao `kdbx run` para o OpenBao, que ainda não foi encontrado e teria de ser construído.
