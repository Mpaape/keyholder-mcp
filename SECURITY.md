# Security policy

Keyholder MCP handles credentials. If you found a way for an agent to see a secret value, the vault token or the wrapped token, or to run something outside the command allow list or the project scope, **do not open a public issue**.

Report it through [GitHub private vulnerability reporting](https://github.com/Mpaape/keyholder-mcp/security/advisories/new), with steps to reproduce. Use fake secrets in the report.

## Supported versions

The project is a proof of concept. Fixes go to `main`; there are no maintained release branches yet.

## Known limits (not vulnerabilities)

- The executed command receives the secret value, as any real use of a credential does.
- Masking only matches the exact value; a command that encodes the secret exposes it. That is why shells and interpreters (`sh`, `bash`, `python`, `env`) must never be in `KEYHOLDER_ALLOW`.
- There is no protection against a hostile agent running as the same OS user.

Details: [Security model](https://mpaape.github.io/keyholder-mcp/security/).

---

## Português

Se você encontrou uma forma de um agente ver um valor de segredo, o token do cofre ou o token embrulhado, ou rodar algo fora da lista de comandos ou do escopo do projeto, **não abra uma issue pública**. Reporte pelo [aviso privado de vulnerabilidade do GitHub](https://github.com/Mpaape/keyholder-mcp/security/advisories/new), com os passos para reproduzir e segredos falsos.

Limites conhecidos, que não são vulnerabilidades: o comando executado recebe o valor; o mascaramento só casa o valor exato; não há proteção contra um agente hostil com o mesmo usuário do sistema. Detalhes no [Modelo de segurança](https://mpaape.github.io/keyholder-mcp/pt-BR/security/).
